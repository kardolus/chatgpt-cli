package client

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/configmanager"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/http"
	"github.com/kardolus/chatgpt-cli/types"
)

const (
	AssistantRole            = "assistant"
	ErrEmptyResponse         = "empty response"
	MaxTokenBufferPercentage = 20
	SystemRole               = "system"
	UserRole                 = "user"
	gptPrefix                = "gpt"
)

type Client struct {
	Config       types.Config
	History      []types.Message
	provider     Provider
	historyStore history.HistoryStore
}

func New(callerFactory http.CallerFactory, cs config.ConfigStore, hs history.HistoryStore) (*Client, error) {
	cm := configmanager.New(cs).WithEnvironment()
	configuration := cm.Config

	if configuration.APIKey == "" {
		return nil, errors.New("missing environment variable: " + cm.APIKeyEnvVarName())
	}

	var (
		provider Provider
		err      error
	)
	switch configuration.Provider {
	case types.OpenAI:
		caller := callerFactory(configuration)
		provider = &OpenAIProvider{caller: caller}
	case types.Cohere:
		provider, err = newCohereProvider(configuration)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported provider: %v", configuration.Provider)
	}

	hs.SetThread(configuration.Thread)

	return &Client{
		Config:       configuration,
		provider:     provider,
		historyStore: hs,
	}, nil
}

func (c *Client) WithContextWindow(window int) *Client {
	c.Config.ContextWindow = window
	return c
}

func (c *Client) WithServiceURL(url string) *Client {
	c.Config.URL = url
	return c
}

// ListModels retrieves a list of all available models from the OpenAI API.
// The models are returned as a slice of strings, each entry representing a model ID.
// Models that have an ID starting with 'gpt' are included.
// The currently active model is marked with an asterisk (*) in the list.
// In case of an error during the retrieval or processing of the models,
// the method returns an error. If the API response is empty, an error is returned as well.
func (c *Client) ListModels() ([]string, error) {
	return c.provider.ListModels(c.Config)
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

// Query sends a query to the API, returning the response as a string along with the token usage.
// It takes an input string, constructs a request body, and makes a POST API call.
// Returns the API response string, the number of tokens used, and an error if any issues occur.
// If the response contains choices, it decodes the JSON and returns the content of the first choice.
func (c *Client) Query(input string) (string, int, error) {
	c.prepareQuery(input)

	response, usage, err := c.provider.Generate(c.History, c.Config)
	if err != nil {
		return "", 0, err
	}

	c.updateHistory(response)

	return response, usage, nil
}

// Stream sends a query to the API and processes the response as a stream.
// It takes an input string as a parameter and returns an error if there's
// any issue during the process. The method creates a request body with the
// input and then makes an API call using the Post method. The actual
// processing of the streamed response is done in the Post method.
func (c *Client) Stream(input string) error {
	c.prepareQuery(input)

	r, err := c.provider.Stream(c.History, c.Config)
	if err != nil {
		return err
	}

	r = io.TeeReader(r, os.Stdout)

	response, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	c.updateHistory(string(response))

	return nil
}

func (c *Client) initHistory() {
	if len(c.History) != 0 {
		return
	}

	if !c.Config.OmitHistory {
		c.History, _ = c.historyStore.Read()
	}

	if len(c.History) == 0 {
		c.History = []types.Message{{
			Role: SystemRole,
		}}
	}

	c.History[0].Content = c.Config.Role
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
	return c.Config.URL + path
}

func (c *Client) prepareQuery(input string) {
	c.initHistory()
	c.addQuery(input)
}

func (c *Client) truncateHistory() {
	tokens, rolling := countTokens(c.History)
	effectiveTokenSize := calculateEffectiveContextWindow(c.Config.ContextWindow, MaxTokenBufferPercentage)

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

	if !c.Config.OmitHistory {
		_ = c.historyStore.Write(c.History)
	}
}

func calculateEffectiveContextWindow(window int, bufferPercentage int) int {
	adjustedPercentage := 100 - bufferPercentage
	effectiveContextWindow := (window * adjustedPercentage) / 100
	return effectiveContextWindow
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
