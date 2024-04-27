package client

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/kardolus/chatgpt-cli/http"
	"github.com/kardolus/chatgpt-cli/types"
)

type Provider interface {
	Generate(history []types.Message, cfg types.Config) (string, int, error)
	Stream(history []types.Message, cfg types.Config) (io.Reader, error)
	ListModels(cfg types.Config) ([]string, error)
}

type OpenAIProvider struct {
	caller http.Caller
}

var _ Provider = (*OpenAIProvider)(nil)

func (p *OpenAIProvider) Generate(history []types.Message, cfg types.Config) (string, int, error) {
	req := types.CompletionsRequest{
		Messages:         history,
		Model:            cfg.Model,
		MaxTokens:        cfg.MaxTokens,
		Temperature:      cfg.Temperature,
		TopP:             cfg.TopP,
		FrequencyPenalty: cfg.FrequencyPenalty,
		PresencePenalty:  cfg.PresencePenalty,
		Stream:           false,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", 0, err
	}
	res, err := p.caller.Post(getEndpoint(cfg, cfg.CompletionsPath), body)
	if err != nil {
		return "", 0, err
	}
	defer res.Close()
	raw, err := io.ReadAll(res)
	if err != nil {
		return "", 0, err
	}
	var response types.CompletionsResponse
	if err := processResponse(raw, &response); err != nil {
		return "", 0, err
	}
	if len(response.Choices) == 0 {
		return "", response.Usage.TotalTokens, errors.New("no responses returned")
	}
	return response.Choices[0].Message.Content, response.Usage.TotalTokens, nil
}

func (p *OpenAIProvider) Stream(history []types.Message, cfg types.Config) (io.Reader, error) {
	req := types.CompletionsRequest{
		Messages:         history,
		Model:            cfg.Model,
		MaxTokens:        cfg.MaxTokens,
		Temperature:      cfg.Temperature,
		TopP:             cfg.TopP,
		FrequencyPenalty: cfg.FrequencyPenalty,
		PresencePenalty:  cfg.PresencePenalty,
		Stream:           true,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	res, err := p.caller.Post(getEndpoint(cfg, cfg.CompletionsPath), body)
	if err != nil {
		return nil, err
	}
	return newStreamReader(res), nil
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
	if len(raw) == 0 {
		return errors.New(ErrEmptyResponse)
	}

	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

type streamReader struct {
	closer  io.Closer
	scanner *bufio.Scanner
	buffer  []byte
}

var _ io.Reader = (*streamReader)(nil)

func newStreamReader(r io.Reader) *streamReader {
	closer, _ := r.(io.Closer)
	return &streamReader{
		closer:  closer,
		scanner: bufio.NewScanner(r),
	}
}

func (s *streamReader) Read(p []byte) (int, error) {
	if len(s.buffer) > 0 {
		n := copy(p, s.buffer)
		s.buffer = s.buffer[n:]
		return n, nil
	}

	if s.scanner == nil || !s.scanner.Scan() {
		if s.closer != nil {
			s.closer.Close()
		}
		return 0, io.EOF
	}

	line := s.scanner.Text()
	if strings.HasPrefix(line, "data:") {
		line = line[6:] // Skip the "data: " prefix
		if len(line) < 6 {
			return 0, nil
		}

		if line == "[DONE]" {
			s.buffer = append(s.buffer, []byte("\n")...)
			s.scanner = nil
			return 0, nil
		}

		var data types.Data
		err := json.Unmarshal([]byte(line), &data)
		if err != nil {
			s.buffer = append(s.buffer, []byte(fmt.Sprintf("Error: %s\n", err.Error()))...)
			s.scanner = nil
			return 0, nil
		}

		for _, choice := range data.Choices {
			if content, ok := choice.Delta["content"]; ok {
				s.buffer = append(s.buffer, []byte(content)...)
			}
		}
	}

	n := copy(p, s.buffer)
	s.buffer = s.buffer[n:]
	return n, nil
}
