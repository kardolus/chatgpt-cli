package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/history"
)

// defaultMaxToolCalls bounds how many tool-call rounds a single Query may run
// before giving up, so a model that keeps calling tools can't loop forever.
const defaultMaxToolCalls = 10

// ToolExecutor advertises callable functions to the model and dispatches the
// model's tool calls to their implementations. The MCP bridge (see
// mcp_tools.go) is the primary implementation; tests use a stub.
type ToolExecutor interface {
	// Definitions returns the function definitions advertised to the model.
	Definitions(ctx context.Context) ([]api.FunctionTool, error)
	// Execute runs the named tool with JSON-encoded arguments and returns its
	// textual result.
	Execute(ctx context.Context, name, argsJSON string) (string, error)
}

// WithToolExecutor attaches a tool executor (fluent, mirrors WithTransport).
func (c *Client) WithToolExecutor(e ToolExecutor) *Client {
	c.toolExecutor = e
	return c
}

// toolsEnabled reports whether model-driven function calling is active for this
// request (config flag on AND an executor wired in).
func (c *Client) toolsEnabled() bool {
	return c.Config.Tools && c.toolExecutor != nil
}

func (c *Client) maxToolCalls() int {
	if c.Config.MaxToolCalls > 0 {
		return c.Config.MaxToolCalls
	}
	return defaultMaxToolCalls
}

// queryCompletions runs the Chat Completions path. Without tools it is a single
// request; with tools enabled it loops — dispatching each round of tool calls
// and feeding the results back — until the model returns a final answer or the
// round budget is exhausted.
func (c *Client) queryCompletions(ctx context.Context) (string, int, error) {
	total := 0

	// The assistant tool-call turns and role:"tool" results appended below are
	// EPHEMERAL: they're needed in-flight for the follow-up requests, but we
	// strip them before returning so only the user turn and the model's final
	// answer are persisted. This keeps the saved history a valid message
	// sequence — truncation can never orphan a role:"tool" from its assistant
	// tool_calls turn on reload.
	baseLen := len(c.History)

	for iter := 0; ; iter++ {
		body, err := c.createBody(ctx, false)
		if err != nil {
			return "", total, err
		}

		endpoint := c.getChatEndpoint()
		c.printRequestDebugInfo(endpoint, body, nil)

		raw, err := c.Caller.Post(endpoint, body, false)
		c.printResponseDebugInfo(raw)
		if err != nil {
			return "", total, err
		}

		var res api.CompletionsResponse
		if err := c.processResponse(raw, &res); err != nil {
			return "", total, err
		}
		total += res.Usage.TotalTokens

		if len(res.Choices) == 0 {
			return "", total, errors.New("no responses returned")
		}

		msg := res.Choices[0].Message

		// No tool calls (or tools disabled) -> this is the final answer. Drop the
		// ephemeral tool-round messages before returning.
		if !c.toolsEnabled() || len(msg.ToolCalls) == 0 {
			content, ok := msg.Content.(string)
			if !ok {
				return "", total, errors.New("response cannot be converted to a string")
			}
			c.History = c.History[:baseLen]
			return content, total, nil
		}

		if iter >= c.maxToolCalls() {
			c.History = c.History[:baseLen]
			return "", total, fmt.Errorf("exceeded max tool-call rounds (%d); the model kept requesting tools", c.maxToolCalls())
		}

		// Record the assistant's tool-call turn, then run each call and append
		// its result as a role:"tool" message keyed by tool_call_id.
		c.appendMessage(msg)
		for _, tc := range msg.ToolCalls {
			out, execErr := c.toolExecutor.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
			if execErr != nil {
				// Feed the error back to the model rather than aborting the run.
				out = fmt.Sprintf("error executing tool %q: %v", tc.Function.Name, execErr)
			}
			c.appendMessage(api.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    out,
			})
		}
	}
}

// appendMessage appends a raw message to the in-memory history without
// truncating (used for the intermediate assistant/tool turns of a tool run).
func (c *Client) appendMessage(msg api.Message) {
	c.History = append(c.History, history.History{
		Message:   msg,
		Timestamp: c.timer.Now(),
	})
}
