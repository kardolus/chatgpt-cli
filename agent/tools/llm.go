package tools

import (
	"context"
	apiclient "github.com/kardolus/chatgpt-cli/api/client"
)

type LLM interface {
	Complete(ctx context.Context, prompt string) (string, int, error)
}

type ClientLLM struct {
	c *apiclient.Client
}

func NewClientLLM(c *apiclient.Client) *ClientLLM { return &ClientLLM{c: c} }

func (l *ClientLLM) Complete(ctx context.Context, prompt string) (string, int, error) {
	// save
	prevOmit := l.c.Config.OmitHistory
	prevTemp := l.c.Config.Temperature

	// set for agent internals
	l.c.Config.OmitHistory = true
	l.c.Config.Temperature = 0

	// restore no matter what
	defer func() {
		l.c.Config.OmitHistory = prevOmit
		l.c.Config.Temperature = prevTemp
	}()

	out, tokens, err := l.c.Query(ctx, prompt)
	if err != nil {
		return "", 0, err
	}
	return out, tokens, nil
}
