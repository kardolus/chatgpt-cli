package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/api/http"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/internal"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kardolus/chatgpt-cli/history"
	stdhttp "net/http"
)

const (
	AssistantRole            = "assistant"
	ErrEmptyResponse         = "empty response"
	MaxTokenBufferPercentage = 20
	SystemRole               = "system"
	UserRole                 = "user"
	InteractiveThreadPrefix  = "int_"
	gptPrefix                = "gpt"
	o1Prefix                 = "o1"
	imageURLType             = "image_url"
	imageContent             = "data:%s;base64,%s"
	httpScheme               = "http"
	httpsScheme              = "https"
	bufferSize               = 512
)

type Timer interface {
	Now() time.Time
}

type RealTime struct {
}

func (r *RealTime) Now() time.Time {
	return time.Now()
}

type ImageReader interface {
	ReadFile(name string) ([]byte, error)
	ReadBufferFromFile(file *os.File) ([]byte, error)
	Open(name string) (*os.File, error)
}

type RealImageReader struct{}

func (r *RealImageReader) Open(name string) (*os.File, error) {
	return os.Open(name)
}

func (r *RealImageReader) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (r *RealImageReader) ReadBufferFromFile(file *os.File) ([]byte, error) {
	buffer := make([]byte, bufferSize)
	_, err := file.Read(buffer)

	return buffer, err
}

type Client struct {
	Config       config.Config
	History      []history.History
	caller       http.Caller
	historyStore history.Store
	timer        Timer
	reader       ImageReader
}

func New(callerFactory http.CallerFactory, hs history.Store, t Timer, r ImageReader, cfg config.Config, interactiveMode bool) *Client {
	caller := callerFactory(cfg)

	if interactiveMode && cfg.AutoCreateNewThread {
		hs.SetThread(GenerateUniqueSlug(InteractiveThreadPrefix))
	} else {
		hs.SetThread(cfg.Thread)
	}

	return &Client{
		Config:       cfg,
		caller:       caller,
		historyStore: hs,
		timer:        t,
		reader:       r,
	}
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
	var result []string

	endpoint := c.getEndpoint(c.Config.ModelsPath)

	if c.Config.Debug {
		c.printRequestDebugInfo(endpoint, nil)
	}

	raw, err := c.caller.Get(c.getEndpoint(c.Config.ModelsPath))
	if c.Config.Debug {
		c.printResponseDebugInfo(raw)
	}

	if err != nil {
		return nil, err
	}

	var response api.ListModelsResponse
	if err := c.processResponse(raw, &response); err != nil {
		return nil, err
	}

	for _, model := range response.Data {
		if strings.HasPrefix(model.Id, gptPrefix) || strings.HasPrefix(model.Id, o1Prefix) {
			if model.Id != c.Config.Model {
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
	historyEntries := c.createHistoryEntriesFromString(context)
	c.History = append(c.History, historyEntries...)
}

// Query sends a query to the API, returning the response as a string along with the token usage.
//
// It takes a context `ctx` and an input string, constructs a request body, and makes a POST API call.
// The context allows for request scoping, timeouts, and cancellation handling.
//
// Returns the API response string, the number of tokens used, and an error if any issues occur.
// If the response contains choices, it decodes the JSON and returns the content of the first choice.
//
// Parameters:
//   - ctx: A context.Context that controls request cancellation and deadlines.
//   - input: The query string to send to the API.
//
// Returns:
//   - string: The content of the first response choice from the API.
//   - int: The total number of tokens used in the request.
//   - error: An error if the request fails or the response is invalid.
func (c *Client) Query(ctx context.Context, input string) (string, int, error) {
	c.prepareQuery(input)

	body, err := c.createBody(ctx, false)
	if err != nil {
		return "", 0, err
	}

	endpoint := c.getEndpoint(c.Config.CompletionsPath)

	if c.Config.Debug {
		c.printRequestDebugInfo(endpoint, body)
	}

	raw, err := c.caller.Post(endpoint, body, false)
	if c.Config.Debug {
		c.printResponseDebugInfo(raw)
	}

	if err != nil {
		return "", 0, err
	}

	var response api.CompletionsResponse
	if err := c.processResponse(raw, &response); err != nil {
		return "", 0, err
	}

	if len(response.Choices) == 0 {
		return "", response.Usage.TotalTokens, errors.New("no responses returned")
	}

	c.updateHistory(response.Choices[0].Message.Content.(string))

	return response.Choices[0].Message.Content.(string), response.Usage.TotalTokens, nil
}

// Stream sends a query to the API and processes the response as a stream.
//
// It takes a context `ctx` and an input string, constructs a request body, and makes a POST API call.
// The context allows for request scoping, timeouts, and cancellation handling.
//
// The method creates a request body with the input and calls the API using the `Post` method.
// The actual processing of the streamed response is handled inside the `Post` method.
//
// Parameters:
//   - ctx: A context.Context that controls request cancellation and deadlines.
//   - input: The query string to send to the API.
//
// Returns:
//   - error: An error if the request fails or the response is invalid.
func (c *Client) Stream(ctx context.Context, input string) error {
	c.prepareQuery(input)

	body, err := c.createBody(ctx, true)
	if err != nil {
		return err
	}

	endpoint := c.getEndpoint(c.Config.CompletionsPath)

	if c.Config.Debug {
		c.printRequestDebugInfo(endpoint, body)
	}

	result, err := c.caller.Post(endpoint, body, true)
	if err != nil {
		return err
	}

	c.updateHistory(string(result))

	return nil
}

func (c *Client) createBody(ctx context.Context, stream bool) ([]byte, error) {
	var messages []api.Message

	for index, item := range c.History {
		if strings.HasPrefix(c.Config.Model, o1Prefix) && index == 0 {
			continue
		}
		messages = append(messages, item.Message)
	}

	body := api.CompletionsRequest{
		Messages:         messages,
		Model:            c.Config.Model,
		MaxTokens:        c.Config.MaxTokens,
		Temperature:      c.Config.Temperature,
		TopP:             c.Config.TopP,
		FrequencyPenalty: c.Config.FrequencyPenalty,
		PresencePenalty:  c.Config.PresencePenalty,
		Seed:             c.Config.Seed,
		Stream:           stream,
	}

	if data, ok := ctx.Value(internal.BinaryDataKey).([]byte); ok {
		content, err := c.createImageContentFromBinary(data)
		if err != nil {
			return nil, err
		}
		body.Messages = append(body.Messages, api.Message{
			Role:    UserRole,
			Content: []api.ImageContent{content},
		})
	} else if path, ok := ctx.Value(internal.ImagePathKey).(string); ok {
		content, err := c.createImageContentFromURLOrFile(path)
		if err != nil {
			return nil, err
		}
		body.Messages = append(body.Messages, api.Message{
			Role:    UserRole,
			Content: []api.ImageContent{content},
		})
	}

	return json.Marshal(body)
}

func (c *Client) createImageContentFromBinary(binary []byte) (api.ImageContent, error) {
	mime, err := c.getMimeTypeFromBytes(binary)
	if err != nil {
		return api.ImageContent{}, err
	}

	encoded := base64.StdEncoding.EncodeToString(binary)
	content := api.ImageContent{
		Type: imageURLType,
		ImageURL: struct {
			URL string `json:"url"`
		}{
			URL: fmt.Sprintf(imageContent, mime, encoded),
		},
	}

	return content, nil
}

func (c *Client) createImageContentFromURLOrFile(image string) (api.ImageContent, error) {
	var content api.ImageContent

	if isValidURL(image) {
		content = api.ImageContent{
			Type: imageURLType,
			ImageURL: struct {
				URL string `json:"url"`
			}{
				URL: image,
			},
		}
	} else {
		mime, err := c.getMimeTypeFromFileContent(image)
		if err != nil {
			return content, err
		}

		encodedImage, err := c.base64EncodeImage(image)
		if err != nil {
			return content, err
		}

		content = api.ImageContent{
			Type: imageURLType,
			ImageURL: struct {
				URL string `json:"url"`
			}{
				URL: fmt.Sprintf(imageContent, mime, encodedImage),
			},
		}
	}

	return content, nil
}

func (c *Client) initHistory() {
	if len(c.History) != 0 {
		return
	}

	if !c.Config.OmitHistory {
		c.History, _ = c.historyStore.Read()
	}

	if len(c.History) == 0 {
		c.History = []history.History{{
			Message: api.Message{
				Role: SystemRole,
			},
			Timestamp: c.timer.Now(),
		}}
	}

	c.History[0].Content = c.Config.Role
}

func (c *Client) addQuery(query string) {
	message := api.Message{
		Role:    UserRole,
		Content: query,
	}

	c.History = append(c.History, history.History{
		Message:   message,
		Timestamp: c.timer.Now(),
	})
	c.truncateHistory()
}

func (c *Client) getEndpoint(path string) string {
	return c.Config.URL + path
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
	c.History = append(c.History, history.History{
		Message: api.Message{
			Role:    AssistantRole,
			Content: response,
		},
		Timestamp: c.timer.Now(),
	})

	if !c.Config.OmitHistory {
		_ = c.historyStore.Write(c.History)
	}
}

func (c *Client) base64EncodeImage(path string) (string, error) {
	imageData, err := c.reader.ReadFile(path)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(imageData), nil
}

func (c *Client) createHistoryEntriesFromString(input string) []history.History {
	var result []history.History

	words := strings.Fields(input)

	for i := 0; i < len(words); i += 100 {
		end := i + 100
		if end > len(words) {
			end = len(words)
		}

		content := strings.Join(words[i:end], " ")

		item := history.History{
			Message: api.Message{
				Role:    UserRole,
				Content: content,
			},
			Timestamp: c.timer.Now(),
		}
		result = append(result, item)
	}

	return result
}

func (c *Client) getMimeTypeFromBytes(data []byte) (string, error) {
	mimeType := stdhttp.DetectContentType(data)

	return mimeType, nil
}

func (c *Client) getMimeTypeFromFileContent(path string) (string, error) {
	file, err := c.reader.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer, err := c.reader.ReadBufferFromFile(file)
	if err != nil {
		return "", err
	}

	mimeType := stdhttp.DetectContentType(buffer)

	return mimeType, nil
}

func (c *Client) printRequestDebugInfo(endpoint string, body []byte) {
	fmt.Printf("\nGenerated cURL command:\n\n")
	method := "POST"
	if body == nil {
		method = "GET"
	}
	fmt.Printf("curl --location --insecure --request %s '%s' \\\n", method, endpoint)
	fmt.Printf("  --header \"Authorization: Bearer ${%s_API_KEY}\" \\\n", strings.ToUpper(c.Config.Name))
	fmt.Printf("  --header 'Content-Type: application/json'")

	if body != nil {
		bodyString := strings.ReplaceAll(string(body), "'", "'\"'\"'") // Escape single quotes
		fmt.Printf(" \\\n  --data-raw '%s'", bodyString)
	}
	fmt.Println() // Print a newline at the end
}

func (c *Client) printResponseDebugInfo(raw []byte) {
	fmt.Printf("\nResponse\n\n")
	fmt.Printf("%s\n\n", raw)
}

func GenerateUniqueSlug(prefix string) string {
	guid := uuid.New()
	return prefix + guid.String()[:4]
}

func calculateEffectiveContextWindow(window int, bufferPercentage int) int {
	adjustedPercentage := 100 - bufferPercentage
	effectiveContextWindow := (window * adjustedPercentage) / 100
	return effectiveContextWindow
}

func countTokens(entries []history.History) (int, []int) {
	var result int
	var rolling []int

	for _, entry := range entries {
		charCount, wordCount := 0, 0
		words := strings.Fields(entry.Content.(string))
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

func isValidURL(input string) bool {
	parsedURL, err := url.ParseRequestURI(input)
	if err != nil {
		return false
	}

	// Ensure that the URL has a valid scheme
	schemes := []string{httpScheme, httpsScheme}
	for _, scheme := range schemes {
		if strings.HasPrefix(parsedURL.Scheme, scheme) {
			return true
		}
	}

	return false
}
