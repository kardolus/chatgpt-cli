package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/http"
	"github.com/kardolus/chatgpt-cli/types"
)

const (
	AssistantContent = "You are a helpful assistant."
	AssistantRole    = "assistant"
	GPTModel         = "gpt-3.5-turbo"
	SystemRole       = "system"
	URL              = "https://api.openai.com/v1/chat/completions"
	UserRole         = "user"
)

type Client struct {
	caller     http.Caller
	readWriter history.Store
	history    []types.Message
}

func New(caller http.Caller, rw history.Store) *Client {
	return &Client{
		caller:     caller,
		readWriter: rw,
	}
}

// Query sends a query to the API and returns the response as a string.
// It takes an input string as a parameter and returns a string containing
// the API response or an error if there's any issue during the process.
// The method creates a request body with the input and then makes an API
// call using the Post method. If the response is not empty, it decodes the
// response JSON and returns the content of the first choice.
func (c *Client) Query(input string) (string, error) {
	c.initHistory(input)

	body, err := c.createBody(false)
	if err != nil {
		return "", err
	}

	raw, err := c.caller.Post(URL, body, false)
	if err != nil {
		return "", err
	}

	if raw == nil {
		return "", errors.New("empty response")
	}

	var response types.Response
	if err := json.Unmarshal(raw, &response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
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
	c.initHistory(input)

	body, err := c.createBody(true)
	if err != nil {
		return err
	}

	result, err := c.caller.Post(URL, body, true)
	if err != nil {
		return err
	}

	c.updateHistory(string(result))

	return nil
}

func (c *Client) createBody(stream bool) ([]byte, error) {
	body := types.Request{
		Model:    GPTModel,
		Messages: c.history,
		Stream:   stream,
	}

	return json.Marshal(body)
}

func (c *Client) initHistory(query string) {
	message := types.Message{
		Role:    UserRole,
		Content: query,
	}

	c.history, _ = c.readWriter.Read()
	if len(c.history) == 0 {
		c.history = []types.Message{{
			Role:    SystemRole,
			Content: AssistantContent,
		}}
	}

	// TODO Write history specific tests
	// TODO Write delete-specific tests (on store)
	// TODO Test the string returned from Stream
	// TODO implement sliding window

	c.history = append(c.history, message)
}

func (c *Client) updateHistory(response string) {
	c.history = append(c.history, types.Message{
		Role:    AssistantRole,
		Content: response,
	})
	_ = c.readWriter.Write(c.history)
}
