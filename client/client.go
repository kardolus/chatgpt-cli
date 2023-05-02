package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-cli/http"
	"github.com/kardolus/chatgpt-cli/types"
)

const (
	model = "gpt-3.5-turbo"
	role  = "user"
	URL   = "https://api.openai.com/v1/chat/completions"
)

type Client struct {
	caller http.Caller
}

func New(caller http.Caller) *Client {
	return &Client{caller: caller}
}

// Query sends a query to the API and returns the response as a string.
// It takes an input string as a parameter and returns a string containing
// the API response or an error if there's any issue during the process.
// The method creates a request body with the input and then makes an API
// call using the Post method. If the response is not empty, it decodes the
// response JSON and returns the content of the first choice.
func (c *Client) Query(input string) (string, error) {
	body, err := CreateBody(input, false)
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

	return response.Choices[0].Message.Content, nil
}

// Stream sends a query to the API and processes the response as a stream.
// It takes an input string as a parameter and returns an error if there's
// any issue during the process. The method creates a request body with the
// input and then makes an API call using the Post method. The actual
// processing of the streamed response is done in the Post method.
func (c *Client) Stream(input string) error {
	body, err := CreateBody(input, true)
	if err != nil {
		return err
	}

	_, err = c.caller.Post(URL, body, true)
	if err != nil {
		return err
	}

	return nil
}

func CreateBody(query string, stream bool) ([]byte, error) {
	message := types.Message{
		Role:    role,
		Content: query,
	}

	body := types.Request{
		Model:    model,
		Messages: []types.Message{message},
		Stream:   stream,
	}

	result, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	return result, nil
}
