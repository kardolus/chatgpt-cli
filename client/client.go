package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/configmanager"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/http"
	"github.com/kardolus/chatgpt-cli/types"
	"strings"
	"unicode/utf8"
)

const (
	AssistantContent         = "You are a helpful assistant."
	AssistantRole            = "assistant"
	ErrEmptyResponse         = "empty response"
	DefaultGPTModel          = "gpt-3.5-turbo"
	DefaultServiceURL        = "https://api.openai.com"
	CompletionPath           = "/v1/chat/completions"
	ModelPath                = "/v1/models"
	MaxTokenBufferPercentage = 20
	MaxTokenSize             = 4096
	SystemRole               = "system"
	UserRole                 = "user"
	gptPrefix                = "gpt"
)

type Client struct {
	History      []types.Message
	Model        string
	caller       http.Caller
	capacity     int
	historyStore history.HistoryStore
	serviceURL   string
}

func New(caller http.Caller, cs config.ConfigStore, hs history.HistoryStore) *Client {
	result := &Client{
		caller:       caller,
		historyStore: hs,
		capacity:     MaxTokenSize,
		serviceURL:   DefaultServiceURL,
	}

	// do not error out when the config cannot be read
	result.Model, _ = configmanager.New(cs).ReadModel()

	if result.Model == "" {
		result.Model = DefaultGPTModel
	}

	return result
}

func (c *Client) WithCapacity(capacity int) *Client {
	c.capacity = capacity
	return c
}

func (c *Client) WithModel(model string) *Client {
	c.Model = model
	return c
}

func (c *Client) WithServiceURL(url string) *Client {
	c.serviceURL = url
	return c
}

// ListModels retrieves a list of all available models from the OpenAI API.
// The models are returned as a slice of strings, each entry representing a model ID.
// Models that have an ID starting with 'gpt' are included.
// The currently active model is marked with an asterisk (*) in the list.
// In case of an error during the retrieval or processing of the models,
// the method returns an error. If the API response is empty, an error is returned as well.
func (c *Client) ListModels() ([]string, error) {
	var result []string

	raw, err := c.caller.Get(c.getEndpoint(ModelPath))
	if err != nil {
		return nil, err
	}

	var response types.ListModelsResponse
	if err := c.processResponse(raw, &response); err != nil {
		return nil, err
	}

	for _, model := range response.Data {
		if strings.HasPrefix(model.Id, gptPrefix) {
			if model.Id != c.Model {
				result = append(result, fmt.Sprintf("- %s", model.Id))
				continue
			}
			result = append(result, fmt.Sprintf("* %s (current)", model.Id))
		}
	}

	return result, nil
}

// ProvideContext adds custom context to the client's history by converting the
// provided string into a series of messages. This allows the ChatGPT API to have
// prior knowledge of the provided context when generating responses.
//
// The context string should contain the text you want to provide as context,
// and the method will split it into messages, preserving punctuation and special
// characters.
func (c *Client) ProvideContext(context string) {
	c.initHistory()
	messages := createMessagesFromString(context)
	c.History = append(c.History, messages...)
}

// Query sends a query to the API and returns the response as a string.
// It takes an input string as a parameter and returns a string containing
// the API response or an error if there's any issue during the process.
// The method creates a request body with the input and then makes an API
// call using the Post method. If the response is not empty, it decodes the
// response JSON and returns the content of the first choice.
func (c *Client) Query(input string) (string, error) {
	c.prepareQuery(input)

	body, err := c.createBody(false)
	if err != nil {
		return "", err
	}

	raw, err := c.caller.Post(c.getEndpoint(CompletionPath), body, false)
	if err != nil {
		return "", err
	}

	var response types.CompletionsResponse
	if err := c.processResponse(raw, &response); err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", errors.New("no responses returned")
	}

	c.updateHistory(response.Choices[0].Message.Content)

	return response.Choices[0].Message.Content, nil
}

// Stream sends a query to the API and processes the response as a stream.
// It takes an input string as a parameter and returns an error if there's
// any issue during the process. The method creates a request body with the
// input and then makes an API call using the Post method. The actual
// processing of the streamed response is done in the Post method.
func (c *Client) Stream(input string) error {
	c.prepareQuery(input)

	body, err := c.createBody(true)
	if err != nil {
		return err
	}

	result, err := c.caller.Post(c.getEndpoint(CompletionPath), body, true)
	if err != nil {
		return err
	}

	c.updateHistory(string(result))

	return nil
}

func (c *Client) createBody(stream bool) ([]byte, error) {
	body := types.CompletionsRequest{
		Messages: c.History,
		Model:    c.Model,
		Stream:   stream,
	}

	return json.Marshal(body)
}

func (c *Client) initHistory() {
	if len(c.History) != 0 {
		return
	}

	c.History, _ = c.historyStore.Read()
	if len(c.History) == 0 {
		c.History = []types.Message{{
			Role:    SystemRole,
			Content: AssistantContent,
		}}
	}
}

func (c *Client) addQuery(query string) {
	message := types.Message{
		Role:    UserRole,
		Content: query,
	}

	c.History = append(c.History, message)
	c.truncateHistory()
}

func (c *Client) getEndpoint(path string) string {
	return c.serviceURL + path
}

func (c *Client) prepareQuery(input string) {
	c.initHistory()
	c.addQuery(input)
}

func (c *Client) processResponse(raw []byte, v interface{}) error {
	if raw == nil {
		return errors.New(ErrEmptyResponse)
	}

	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *Client) truncateHistory() {
	tokens, rolling := countTokens(c.History)
	effectiveTokenSize := calculateEffectiveTokenSize(c.capacity, MaxTokenBufferPercentage)

	if tokens <= effectiveTokenSize {
		return
	}

	var index int
	var total int
	diff := tokens - effectiveTokenSize

	for i := 1; i < len(rolling); i++ {
		total += rolling[i]
		if total > diff {
			index = i
			break
		}
	}

	c.History = append(c.History[:1], c.History[index+1:]...)
}

func (c *Client) updateHistory(response string) {
	c.History = append(c.History, types.Message{
		Role:    AssistantRole,
		Content: response,
	})
	_ = c.historyStore.Write(c.History)
}

func calculateEffectiveTokenSize(maxTokenSize int, bufferPercentage int) int {
	adjustedPercentage := 100 - bufferPercentage
	effectiveTokenSize := (maxTokenSize * adjustedPercentage) / 100
	return effectiveTokenSize
}

func countTokens(messages []types.Message) (int, []int) {
	var result int
	var rolling []int

	for _, message := range messages {
		charCount, wordCount := 0, 0
		words := strings.Fields(message.Content)
		wordCount += len(words)

		for _, word := range words {
			charCount += utf8.RuneCountInString(word)
		}

		// This is a simple approximation; actual token count may differ.
		// You can adjust this based on your language and the specific tokenizer used by the model.
		tokenCountForMessage := (charCount + wordCount) / 2
		result += tokenCountForMessage
		rolling = append(rolling, tokenCountForMessage)
	}

	return result, rolling
}

func createMessagesFromString(input string) []types.Message {
	words := strings.Fields(input)
	var messages []types.Message

	for i := 0; i < len(words); i += 100 {
		end := i + 100
		if end > len(words) {
			end = len(words)
		}

		content := strings.Join(words[i:end], " ")

		message := types.Message{
			Role:    UserRole,
			Content: content,
		}
		messages = append(messages, message)
	}

	return messages
}
