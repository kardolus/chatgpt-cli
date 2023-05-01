package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-poc/http"
	"github.com/kardolus/chatgpt-poc/types"
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

func (c *Client) Query(input string) (string, error) {
	body, err := CreateBody(input)
	if err != nil {
		return "", err
	}

	raw, err := c.caller.Post(URL, body)
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

func CreateBody(query string) ([]byte, error) {
	message := types.Message{
		Role:    role,
		Content: query,
	}

	body := types.Request{
		Model:    model,
		Messages: []types.Message{message},
	}

	result, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	return result, nil
}
