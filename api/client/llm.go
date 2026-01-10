package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/history"
	"sort"
	"strings"
)

const (
	ErrEmptyResponse   = "empty response"
	ErrRealTime        = "model %q requires the Realtime API (WebSocket/WebRTC) and is not supported yet"
	SearchModelPattern = "-search"
	gptPrefix          = "gpt"
	o1Prefix           = "o1"
	o1ProPattern       = "o1-pro"
	gpt5Pattern        = "gpt-5"
	realTimePattern    = "realtime"
	messageType        = "message"
	outputTextType     = "output_text"
)

// ListModels retrieves a list of all available models from the OpenAI API.
// The models are returned as a slice of strings, each entry representing a model ID.
// Models that have an ID starting with 'gpt' are included.
// The currently active model is marked with an asterisk (*) in the list.
// In case of an error during the retrieval or processing of the models,
// the method returns an error. If the API response is empty, an error is returned as well.
func (c *Client) ListModels() ([]string, error) {
	var result []string

	endpoint := c.getEndpoint(c.Config.ModelsPath)

	c.printRequestDebugInfo(endpoint, nil, nil)

	raw, err := c.Caller.Get(c.getEndpoint(c.Config.ModelsPath))
	c.printResponseDebugInfo(raw)

	if err != nil {
		return nil, err
	}

	var response api.ListModelsResponse
	if err := c.processResponse(raw, &response); err != nil {
		return nil, err
	}

	sort.Slice(response.Data, func(i, j int) bool {
		return response.Data[i].Id < response.Data[j].Id
	})

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

	endpoint := c.getChatEndpoint()

	c.printRequestDebugInfo(endpoint, body, nil)

	raw, err := c.Caller.Post(endpoint, body, false)
	c.printResponseDebugInfo(raw)

	if err != nil {
		return "", 0, err
	}

	var (
		response   string
		tokensUsed int
	)

	caps := GetCapabilities(c.Config.Model)

	if caps.UsesResponsesAPI {
		var res api.ResponsesResponse
		if err := c.processResponse(raw, &res); err != nil {
			return "", 0, err
		}
		tokensUsed = res.Usage.TotalTokens

		for _, output := range res.Output {
			if output.Type != messageType {
				continue
			}
			for _, content := range output.Content {
				if content.Type == outputTextType {
					response = content.Text
					break
				}
			}
		}

		if response == "" {
			return "", tokensUsed, errors.New("no response returned")
		}
	} else {
		var res api.CompletionsResponse
		if err := c.processResponse(raw, &res); err != nil {
			return "", 0, err
		}
		tokensUsed = res.Usage.TotalTokens

		if len(res.Choices) == 0 {
			return "", tokensUsed, errors.New("no responses returned")
		}

		var ok bool
		response, ok = res.Choices[0].Message.Content.(string)
		if !ok {
			return "", tokensUsed, errors.New("response cannot be converted to a string")
		}
	}

	c.updateHistory(response)

	return response, tokensUsed, nil
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

	endpoint := c.getChatEndpoint()

	c.printRequestDebugInfo(endpoint, body, nil)

	result, err := c.Caller.Post(endpoint, body, true)
	if err != nil {
		return err
	}

	c.updateHistory(string(result))

	return nil
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

func (c *Client) createBody(ctx context.Context, stream bool) ([]byte, error) {
	caps := GetCapabilities(c.Config.Model)

	if caps.IsRealtime {
		return nil, fmt.Errorf(ErrRealTime, c.Config.Model)
	}

	if caps.UsesResponsesAPI {
		req, err := c.createResponsesRequest(ctx, stream)
		if err != nil {
			return nil, err
		}
		return json.Marshal(req)
	}

	req, err := c.createCompletionsRequest(ctx, stream)
	if err != nil {
		return nil, err
	}
	return json.Marshal(req)
}

func (c *Client) createCompletionsRequest(ctx context.Context, stream bool) (*api.CompletionsRequest, error) {
	var messages []api.Message
	caps := GetCapabilities(c.Config.Model)

	for index, item := range c.History {
		if caps.OmitFirstSystemMsg && index == 0 {
			continue
		}
		messages = append(messages, item.Message)
	}

	messages, err := c.appendMediaMessages(ctx, messages)
	if err != nil {
		return nil, err
	}

	req := &api.CompletionsRequest{
		Messages:         messages,
		Model:            c.Config.Model,
		MaxTokens:        c.Config.MaxTokens,
		FrequencyPenalty: c.Config.FrequencyPenalty,
		PresencePenalty:  c.Config.PresencePenalty,
		Seed:             c.Config.Seed,
		Stream:           stream,
	}

	if caps.SupportsTemperature {
		req.Temperature = c.Config.Temperature
	}
	if caps.SupportsTopP {
		req.TopP = c.Config.TopP
	}

	return req, nil
}

func (c *Client) createResponsesRequest(ctx context.Context, stream bool) (*api.ResponsesRequest, error) {
	var messages []api.Message
	caps := GetCapabilities(c.Config.Model)

	for index, item := range c.History {
		if caps.OmitFirstSystemMsg && index == 0 {
			continue
		}
		messages = append(messages, item.Message)
	}

	messages, err := c.appendMediaMessages(ctx, messages)
	if err != nil {
		return nil, err
	}

	req := &api.ResponsesRequest{
		Model:           c.Config.Model,
		Input:           messages,
		MaxOutputTokens: c.Config.MaxTokens,
		Reasoning: api.Reasoning{
			Effort: c.Config.Effort,
		},
		Stream: stream,
	}

	if caps.SupportsTemperature {
		req.Temperature = c.Config.Temperature
	}
	if caps.SupportsTopP {
		req.TopP = c.Config.TopP
	}

	return req, nil
}

func (c *Client) getChatEndpoint() string {
	caps := GetCapabilities(c.Config.Model)

	var endpoint string
	if caps.UsesResponsesAPI {
		endpoint = c.getEndpoint(c.Config.ResponsesPath)
	} else {
		endpoint = c.getEndpoint(c.Config.CompletionsPath)
	}
	return endpoint
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

type ModelCapabilities struct {
	SupportsTemperature bool
	SupportsTopP        bool
	SupportsStreaming   bool
	UsesResponsesAPI    bool
	OmitFirstSystemMsg  bool
	IsRealtime          bool
}

func GetCapabilities(model string) ModelCapabilities {
	isSearch := strings.Contains(model, SearchModelPattern)
	isGpt5 := strings.Contains(model, gpt5Pattern)

	supportsTemp := !isSearch
	supportsTopP := !isSearch && !isGpt5

	return ModelCapabilities{
		SupportsTemperature: supportsTemp,
		SupportsTopP:        supportsTopP,
		SupportsStreaming:   !strings.Contains(model, o1ProPattern),
		UsesResponsesAPI:    strings.Contains(model, o1ProPattern) || isGpt5,
		OmitFirstSystemMsg:  strings.HasPrefix(model, o1Prefix) && !strings.Contains(model, o1ProPattern),
		IsRealtime:          strings.Contains(model, realTimePattern),
	}
}
