package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"net/url"
	"strings"

	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/api/http"
	"github.com/kardolus/chatgpt-cli/history"
)

func (c *Client) InjectMCPContext(mcp api.MCPRequest) error {
	if c.Config.OmitHistory {
		return errors.New(ErrHistoryTracking)
	}
	if mcp.Endpoint == "" {
		return fmt.Errorf("mcp endpoint is required")
	}
	if mcp.Tool == "" {
		return fmt.Errorf("mcp tool is required")
	}

	req, err := c.buildMCPMessage(mcp)
	if err != nil {
		return err
	}

	if rawReq, err := json.Marshal(req); err == nil {
		c.printRequestDebugInfo(mcp.Endpoint, rawReq, buildMCPHeaders(mcp.Headers))
	}

	resp, err := c.transport.Call(mcp.Endpoint, req)
	if err != nil {
		return err
	}

	if rawResp, err := json.Marshal(resp); err == nil {
		c.printResponseDebugInfo(rawResp)
	}

	formatted := formatMCPResponse(resp.Result, mcp.Tool)

	c.initHistory()
	c.History = append(c.History, history.History{
		Message: api.Message{
			Role:    AssistantRole,
			Content: formatted,
		},
		Timestamp: c.timer.Now(),
	})
	c.truncateHistory()

	return c.historyStore.Write(c.History)
}

func (c *Client) buildMCPMessage(mcp api.MCPRequest) (api.MCPMessage, error) {
	paramsObj := map[string]any{
		"name":      mcp.Tool,
		"arguments": mcp.Params,
	}

	rawParams, err := json.Marshal(paramsObj)
	if err != nil {
		return api.MCPMessage{}, fmt.Errorf("failed to marshal mcp params: %w", err)
	}

	// JSON-RPC MCP request
	return api.MCPMessage{
		JSONRPC: "2.0",
		ID:      uuid.NewString(),
		Method:  "tools/call",
		Params:  rawParams,
	}, nil
}

// formatMCPResponse is MCP-shape aware and provider-agnostic.
// Many MCP servers return tool results as:
//
//	resp.Result = {"content":[{"type":"text","text":"..."} ...]}
//
// We prefer extracting text blocks; if that fails, we fall back to raw JSON.
func formatMCPResponse(raw json.RawMessage, tool string) string {
	if len(raw) == 0 {
		return fmt.Sprintf("[MCP: %s] (empty result)", tool)
	}

	// Common MCP result shape.
	type contentBlock struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	}
	type toolResult struct {
		Content []contentBlock `json:"content,omitempty"`
	}

	var r toolResult
	if err := json.Unmarshal(raw, &r); err == nil && len(r.Content) > 0 {
		var parts []string
		for _, b := range r.Content {
			if strings.EqualFold(b.Type, "text") && strings.TrimSpace(b.Text) != "" {
				parts = append(parts, normalizeMaybeJSON(b.Text))
			}
		}
		if len(parts) > 0 {
			return fmt.Sprintf("[MCP: %s]\n%s", tool, strings.Join(parts, "\n\n"))
		}
	}

	// Fallback: pretty print JSON if possible, otherwise raw.
	return fmt.Sprintf("[MCP: %s]\n%s", tool, prettyJSONOrRaw(raw))
}

func normalizeMaybeJSON(s string) string {
	txt := strings.TrimSpace(s)
	if txt == "" {
		return txt
	}

	var v any
	if json.Unmarshal([]byte(txt), &v) == nil {
		b, err := json.MarshalIndent(v, "", "  ")
		if err == nil {
			return string(b)
		}
	}
	return txt
}

func prettyJSONOrRaw(raw []byte) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(raw)
	}
	// Avoid trailing newline surprises; history messages look nicer without it.
	return string(bytes.TrimSpace(b))
}

type MCPTransport interface {
	Call(endpoint string, req api.MCPMessage) (api.MCPMessage, error)
}

type MCPHTTPTransport struct {
	caller  http.Caller
	headers map[string]string
}

func NewMCPTransport(endpoint string, caller http.Caller, headers map[string]string) (*MCPHTTPTransport, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "http", "https":
		return &MCPHTTPTransport{caller: caller, headers: headers}, nil
	default:
		return nil, fmt.Errorf("unsupported mcp transport: %s", u.Scheme)
	}
}

func (t *MCPHTTPTransport) Call(endpoint string, req api.MCPMessage) (api.MCPMessage, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return api.MCPMessage{}, fmt.Errorf("failed to marshal mcp request: %w", err)
	}

	raw, err := t.caller.PostWithHeaders(endpoint, body, buildMCPHeaders(t.headers))
	if err != nil {
		return api.MCPMessage{}, err
	}

	// 1) Try direct JSON first (some servers may reply application/json)
	var resp api.MCPMessage
	if err := json.Unmarshal(raw, &resp); err == nil {
		if resp.Error != nil {
			return api.MCPMessage{}, &api.MCPError{Code: resp.Error.Code, Message: resp.Error.Message, Data: resp.Error.Data}
		}
		return resp, nil
	}

	// 2) If that fails, try SSE (text/event-stream). Extract the first "data:" JSON payload.
	dataJSON, ok := extractFirstSSEDataJSON(raw)
	if !ok {
		return api.MCPMessage{}, fmt.Errorf("invalid mcp response: expected JSON or SSE data frame")
	}

	if err := json.Unmarshal(dataJSON, &resp); err != nil {
		return api.MCPMessage{}, fmt.Errorf("invalid mcp SSE json: %w", err)
	}

	if resp.Error != nil {
		return api.MCPMessage{}, &api.MCPError{Code: resp.Error.Code, Message: resp.Error.Message, Data: resp.Error.Data}
	}

	return resp, nil
}

func extractFirstSSEDataJSON(raw []byte) ([]byte, bool) {
	// Normalize CRLF
	s := strings.ReplaceAll(string(raw), "\r\n", "\n")
	lines := strings.Split(s, "\n")

	var dataLines []string
	inEvent := false

	for _, line := range lines {
		// blank line terminates an event
		if strings.TrimSpace(line) == "" {
			if inEvent && len(dataLines) > 0 {
				payload := strings.Join(dataLines, "\n")
				payload = strings.TrimSpace(payload)
				if payload != "" {
					return []byte(payload), true
				}
			}
			// reset for next event
			inEvent = false
			dataLines = nil
			continue
		}

		// once we see *any* field, we’re in an event
		inEvent = true

		if strings.HasPrefix(line, "data:") {
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload != "" {
				dataLines = append(dataLines, payload)
			}
		}
	}

	// Handle case where stream ended without trailing blank line
	if inEvent && len(dataLines) > 0 {
		payload := strings.TrimSpace(strings.Join(dataLines, "\n"))
		if payload != "" {
			return []byte(payload), true
		}
	}

	return nil, false
}

func buildMCPHeaders(in map[string]string) map[string]string {
	h := map[string]string{}

	// Copy input headers (do not mutate Caller map)
	for k, v := range in {
		h[k] = v
	}

	// Content-Type default
	if _, ok := headerGet(h, "Content-Type"); !ok {
		h["Content-Type"] = "application/json"
	}

	// MCP Streamable HTTP requires both for POST
	if _, ok := headerGet(h, "Accept"); !ok {
		h["Accept"] = "application/json, text/event-stream"
	}

	// Canonicalize session header spelling (optional)
	if v, ok := headerGet(h, "mcp-session-id"); ok {
		headerDel(h, "mcp-session-id")
		h["Mcp-Session-Id"] = v
	}

	return h
}

func headerGet(h map[string]string, key string) (string, bool) {
	for k, v := range h {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return "", false
}

func headerDel(h map[string]string, key string) {
	for k := range h {
		if strings.EqualFold(k, key) {
			delete(h, k)
		}
	}
}
