package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kardolus/chatgpt-cli/http"
	"github.com/kardolus/chatgpt-cli/types"
)

type Provider interface {
	Generate(history []types.Message, cfg types.Config, stream bool) (string, int, error)
	ListModels(cfg types.Config) ([]string, error)
}

type OpenAIProvider struct {
	caller http.Caller
}

var _ Provider = (*OpenAIProvider)(nil)

func (p *OpenAIProvider) Generate(history []types.Message, cfg types.Config, stream bool) (string, int, error) {
	req := types.CompletionsRequest{
		Messages:         history,
		Model:            cfg.Model,
		MaxTokens:        cfg.MaxTokens,
		Temperature:      cfg.Temperature,
		TopP:             cfg.TopP,
		FrequencyPenalty: cfg.FrequencyPenalty,
		PresencePenalty:  cfg.PresencePenalty,
		Stream:           stream,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", 0, err
	}
	res, err := p.caller.Post(getEndpoint(cfg, cfg.CompletionsPath), body, stream)
	if err != nil {
		return "", 0, err
	}
	// TODO writing to a stream shouldn't be hidden away in http.Caller
	if stream {
		return string(res), 0, nil
	}
	var response types.CompletionsResponse
	if err := processResponse(res, &response); err != nil {
		return "", 0, err
	}
	if len(response.Choices) == 0 {
		return "", response.Usage.TotalTokens, errors.New("no responses returned")
	}
	return response.Choices[0].Message.Content, response.Usage.TotalTokens, nil
}

func (p *OpenAIProvider) ListModels(cfg types.Config) ([]string, error) {
	var result []string

	raw, err := p.caller.Get(getEndpoint(cfg, cfg.ModelsPath))
	if err != nil {
		return nil, err
	}

	var response types.ListModelsResponse
	if err := processResponse(raw, &response); err != nil {
		return nil, err
	}

	for _, model := range response.Data {
		if strings.HasPrefix(model.Id, gptPrefix) {
			if model.Id != cfg.Model {
				result = append(result, fmt.Sprintf("- %s", model.Id))
				continue
			}
			result = append(result, fmt.Sprintf("* %s (current)", model.Id))
		}
	}

	return result, nil
}

func getEndpoint(cfg types.Config, path string) string {
	return cfg.URL + path
}

func processResponse(raw []byte, v interface{}) error {
	if raw == nil {
		return errors.New(ErrEmptyResponse)
	}

	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}
