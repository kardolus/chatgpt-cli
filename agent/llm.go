package agent

import (
	"context"
	apiclient "github.com/kardolus/chatgpt-cli/api/client"
)

//go:generate mockgen -destination=llmmocks_test.go -package=agent_test github.com/kardolus/chatgpt-cli/agent LLM
type LLM interface {
	Complete(ctx context.Context, prompt string) (string, int, error)
}

type ClientLLM struct {
	c *apiclient.Client
}

func NewClientLLM(c *apiclient.Client) *ClientLLM { return &ClientLLM{c: c} }

func (l *ClientLLM) Complete(ctx context.Context, prompt string) (string, int, error) {
	l.c.Config.OmitHistory = true
	//l.c.Config.Temperature = 0
	out, tokens, err := l.c.Query(ctx, prompt)
	if err != nil {
		return "", 0, err
	}

	return out, tokens, nil
}
