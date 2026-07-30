package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kardolus/chatgpt-cli/api"
)

// MCPToolExecutor bridges an MCP server's tools to the model's function-calling
// interface: Definitions() advertises the server's `tools/list` as function
// definitions, and Execute() dispatches a model tool call via `tools/call`.
// This lets the model invoke MCP tools autonomously, a step up from the
// one-shot `--mcp-tool` context injection.
type MCPToolExecutor struct {
	transport MCPTransport
	endpoint  string
	headers   map[string]string
}

func NewMCPToolExecutor(transport MCPTransport, endpoint string, headers map[string]string) *MCPToolExecutor {
	return &MCPToolExecutor{transport: transport, endpoint: endpoint, headers: headers}
}

func (m *MCPToolExecutor) Definitions(_ context.Context) ([]api.FunctionTool, error) {
	req := api.MCPMessage{
		JSONRPC: "2.0",
		ID:      uuid.NewString(),
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	}

	resp, err := m.transport.Call(m.endpoint, req, m.headers)
	if err != nil {
		return nil, fmt.Errorf("mcp tools/list failed: %w", err)
	}
	if resp.Message.Error != nil {
		return nil, fmt.Errorf("mcp tools/list error: %w", resp.Message.Error)
	}

	var out struct {
		Tools []struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			InputSchema json.RawMessage `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(resp.Message.Result, &out); err != nil {
		return nil, fmt.Errorf("mcp tools/list: cannot parse result: %w", err)
	}

	defs := make([]api.FunctionTool, 0, len(out.Tools))
	for _, t := range out.Tools {
		params := t.InputSchema
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object"}`)
		}
		defs = append(defs, api.FunctionTool{
			Type: "function",
			Function: api.FunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}
	return defs, nil
}

func (m *MCPToolExecutor) Execute(_ context.Context, name, argsJSON string) (string, error) {
	// The model emits arguments as a JSON string; MCP tools/call requires an
	// arguments OBJECT, so decode into a map and default to {} on empty/invalid
	// or non-object JSON (e.g. an array or bare value).
	args := map[string]any{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		args = map[string]any{}
	}

	params, err := json.Marshal(map[string]any{"name": name, "arguments": args})
	if err != nil {
		return "", err
	}

	req := api.MCPMessage{
		JSONRPC: "2.0",
		ID:      uuid.NewString(),
		Method:  "tools/call",
		Params:  params,
	}

	resp, err := m.transport.Call(m.endpoint, req, m.headers)
	if err != nil {
		return "", fmt.Errorf("mcp tools/call %q failed: %w", name, err)
	}
	if resp.Message.Error != nil {
		return "", fmt.Errorf("mcp tools/call %q error: %w", name, resp.Message.Error)
	}

	return extractToolResultText(resp.Message.Result), nil
}

// extractToolResultText pulls the plain text out of an MCP tools/call result
// (the {"content":[{"type":"text","text":...}]} shape) so the model receives
// the raw tool output as the role:"tool" content — no decorative prefix. Falls
// back to the raw JSON when there are no text blocks.
func extractToolResultText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var r struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &r); err == nil && len(r.Content) > 0 {
		var parts []string
		for _, b := range r.Content {
			if strings.TrimSpace(b.Text) != "" {
				parts = append(parts, b.Text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return string(raw)
}
