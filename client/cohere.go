package client

import (
	"context"
	"io"
	"log"

	"github.com/kardolus/chatgpt-cli/types"

	co "github.com/cohere-ai/cohere-go/v2"
	cohereclient "github.com/cohere-ai/cohere-go/v2/client"
	core "github.com/cohere-ai/cohere-go/v2/core"
)

type CohereProvider struct {
	client *cohereclient.Client
}

var _ Provider = (*CohereProvider)(nil)

func newCohereProvider(cfg types.Config) (*CohereProvider, error) {
	client := cohereclient.NewClient(cohereclient.WithToken(cfg.APIKey))
	return &CohereProvider{
		client: client,
	}, nil
}

func (p *CohereProvider) Generate(history []types.Message, cfg types.Config) (string, int, error) {
	ctx := context.Background()
	req := &co.ChatRequest{
		Message:     history[len(history)-1].Content,
		ChatHistory: coHistory(history),
	}
	res, err := p.client.Chat(ctx, req)
	if err != nil {
		return "", 0, err
	}
	return res.Text, int(*res.Meta.BilledUnits.InputTokens + *res.Meta.BilledUnits.OutputTokens), nil
}

func (p *CohereProvider) Stream(history []types.Message, cfg types.Config) (io.Reader, error) {
	ctx := context.Background()
	req := &co.ChatStreamRequest{
		Message:     history[len(history)-1].Content,
		ChatHistory: coHistory(history),
	}
	stream, err := p.client.ChatStream(ctx, req)
	if err != nil {
		return nil, err
	}
	return &coStreamReader{stream: stream}, nil
}

func (p *CohereProvider) ListModels(cfg types.Config) ([]string, error) {
	ctx := context.Background()
	endpoint := co.CompatibleEndpointChat
	res, err := p.client.Models.List(ctx, &co.ModelsListRequest{
		Endpoint: &endpoint,
	})
	if err != nil {
		return nil, err
	}
	var models []string
	for _, model := range res.Models {
		models = append(models, *model.Name)
	}
	return models, nil
}

func coHistory(history []types.Message) []*co.ChatMessage {
	var chatHistory []*co.ChatMessage
	for _, msg := range history {
		switch msg.Role {
		case AssistantRole:
			chatHistory = append(chatHistory, &co.ChatMessage{
				Role:    co.ChatMessageRoleChatbot,
				Message: msg.Content,
			})
		case UserRole:
			chatHistory = append(chatHistory, &co.ChatMessage{
				Role:    co.ChatMessageRoleUser,
				Message: msg.Content,
			})
		case SystemRole:
			chatHistory = append(chatHistory, &co.ChatMessage{
				Role:    co.ChatMessageRoleSystem,
				Message: msg.Content,
			})
		default:
			log.Fatalf("unknown role: %s", msg.Role)
		}
	}
	return chatHistory
}

type coStreamReader struct {
	stream *core.Stream[co.StreamedChatResponse]
}

var _ io.Reader = (*coStreamReader)(nil)

func (r *coStreamReader) Read(p []byte) (n int, err error) {
	resp, err := r.stream.Recv()
	if err != nil {
		return 0, err
	}
	if resp.TextGeneration == nil {
		return 0, nil
	}
	n = copy(p, []byte(resp.TextGeneration.Text))
	return n, nil
}
