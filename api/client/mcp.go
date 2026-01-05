package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/api/http"
	"github.com/kardolus/chatgpt-cli/history"
)

/* =========================
   Client entrypoint
   ========================= */

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

	resp, err := c.transport.Call(mcp.Endpoint, req, mcp.Headers)
	if err != nil {
		return err
	}

	if rawResp, err := json.Marshal(resp.Message); err == nil {
		c.printResponseDebugInfo(rawResp)
	}

	formatted := formatMCPResponse(resp.Message.Result, mcp.Tool)

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

/* =========================
   MCP message building
   ========================= */

func (c *Client) buildMCPMessage(mcp api.MCPRequest) (api.MCPMessage, error) {
	rawParams, err := json.Marshal(map[string]any{
		"name":      mcp.Tool,
		"arguments": mcp.Params,
	})
	if err != nil {
		return api.MCPMessage{}, fmt.Errorf("failed to marshal mcp params: %w", err)
	}

	return api.MCPMessage{
		JSONRPC: "2.0",
		ID:      uuid.NewString(),
		Method:  "tools/call",
		Params:  rawParams,
	}, nil
}

/* =========================
   Transport interfaces
   ========================= */

type MCPTransport interface {
	Call(endpoint string, req api.MCPMessage, headers map[string]string) (api.MCPResponse, error)
}

type SessionStore interface {
	GetSessionID(endpoint string) (string, error)
	SetSessionID(endpoint, sessionID string) error
	DeleteSessionID(endpoint string) error
}

/* =========================
   Session transport
   ========================= */

type SessionTransport struct {
	inner MCPTransport
	store SessionStore
}

func NewSessionTransport(inner MCPTransport, store SessionStore) *SessionTransport {
	return &SessionTransport{inner: inner, store: store}
}

func (t *SessionTransport) Call(endpoint string, req api.MCPMessage, headers map[string]string) (api.MCPResponse, error) {
	// Explicit session header → passthrough
	if _, ok := headerGet(headers, "mcp-session-id"); ok {
		return t.inner.Call(endpoint, req, headers)
	}

	// Try cached session
	if sid, err := t.store.GetSessionID(endpoint); err == nil && strings.TrimSpace(sid) != "" {
		h := cloneHeaders(headers)
		h["Mcp-Session-Id"] = sid

		resp, err := t.inner.Call(endpoint, req, h)
		if err == nil {
			t.maybeStoreSession(endpoint, resp)
			return resp, nil
		}

		// If the server rejected the session, clear and proceed to init.
		if looksLikeInvalidSession(err) {
			_ = t.store.DeleteSessionID(endpoint)
		} else {
			return resp, err
		}
	}

	// Initialize session
	sid, err := t.initialize(endpoint, headers)
	if err != nil {
		return api.MCPResponse{}, err
	}

	h := cloneHeaders(headers)
	h["Mcp-Session-Id"] = sid

	resp, err := t.inner.Call(endpoint, req, h)
	if err == nil {
		t.maybeStoreSession(endpoint, resp)
	}
	return resp, err
}

func (t *SessionTransport) initialize(endpoint string, headers map[string]string) (string, error) {
	raw, err := json.Marshal(map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "chatgpt-cli",
			"version": "dev",
		},
	})
	if err != nil {
		return "", err
	}

	resp, err := t.inner.Call(endpoint, api.MCPMessage{
		JSONRPC: "2.0",
		ID:      uuid.NewString(),
		Method:  "initialize",
		Params:  raw,
	}, headers)
	if err != nil {
		return "", err
	}

	sid, ok := headerGet(resp.Headers, "mcp-session-id")
	if !ok || strings.TrimSpace(sid) == "" {
		return "", fmt.Errorf("mcp initialize did not return session id")
	}

	_ = t.store.SetSessionID(endpoint, sid)
	return sid, nil
}

func (t *SessionTransport) maybeStoreSession(endpoint string, resp api.MCPResponse) {
	if sid, ok := headerGet(resp.Headers, "mcp-session-id"); ok && strings.TrimSpace(sid) != "" {
		_ = t.store.SetSessionID(endpoint, sid)
	}
}

/* =========================
   HTTP transport
   ========================= */

type MCPHTTPTransport struct {
	caller  http.Caller
	headers map[string]string
}

func NewMCPTransport(endpoint string, caller http.Caller, headers map[string]string) (*MCPHTTPTransport, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported mcp transport: %s", u.Scheme)
	}
	return &MCPHTTPTransport{caller: caller, headers: headers}, nil
}

func (t *MCPHTTPTransport) Call(endpoint string, req api.MCPMessage, extra map[string]string) (api.MCPResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return api.MCPResponse{}, fmt.Errorf("failed to marshal mcp request: %w", err)
	}

	merged := map[string]string{}
	for k, v := range t.headers {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}

	httpResp, postErr := t.caller.PostWithHeadersResponse(endpoint, body, buildMCPHeaders(merged))

	out := api.MCPResponse{
		Headers: httpResp.Headers,
		Status:  httpResp.Status,
	}

	// Even on non-2xx, parse body if possible so SessionTransport can reason about it.
	if len(httpResp.Body) > 0 {
		var msg api.MCPMessage
		if err := json.Unmarshal(httpResp.Body, &msg); err == nil {
			out.Message = msg
		} else if dataJSON, ok := extractFirstSSEDataJSON(httpResp.Body); ok {
			if err := json.Unmarshal(dataJSON, &msg); err == nil {
				out.Message = msg
			}
		}
	}

	// Prefer JSON-RPC error if present.
	if out.Message.Error != nil {
		return out, out.Message.Error
	}

	// Otherwise propagate HTTP-layer error.
	if postErr != nil {
		return out, postErr
	}

	return out, nil
}

/* =========================
   Helpers
   ========================= */

func looksLikeInvalidSession(err error) bool {
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "session") {
		return false
	}

	return strings.Contains(msg, "missing") ||
		strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "no valid") ||
		strings.Contains(msg, "expired") ||
		strings.Contains(msg, "unknown")
}

func cloneHeaders(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func headerGet(h map[string]string, key string) (string, bool) {
	for k, v := range h {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return "", false
}

func headerDelCI(h map[string]string, key string) {
	for k := range h {
		if strings.EqualFold(k, key) {
			delete(h, k)
		}
	}
}

func buildMCPHeaders(in map[string]string) map[string]string {
	h := cloneHeaders(in)

	if _, ok := headerGet(h, "Content-Type"); !ok {
		h["Content-Type"] = "application/json"
	}
	if _, ok := headerGet(h, "Accept"); !ok {
		h["Accept"] = "application/json, text/event-stream"
	}

	// Canonicalize mcp-session-id → Mcp-Session-Id
	if v, ok := headerGet(h, "mcp-session-id"); ok {
		headerDelCI(h, "mcp-session-id")
		h["Mcp-Session-Id"] = v
	}

	return h
}

func extractFirstSSEDataJSON(raw []byte) ([]byte, bool) {
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	var data []string
	for _, l := range lines {
		if strings.HasPrefix(l, "data:") {
			data = append(data, strings.TrimSpace(strings.TrimPrefix(l, "data:")))
		}
	}
	if len(data) == 0 {
		return nil, false
	}
	return []byte(strings.Join(data, "\n")), true
}

func formatMCPResponse(raw json.RawMessage, tool string) string {
	if len(raw) == 0 {
		return fmt.Sprintf("[MCP: %s] (empty result)", tool)
	}

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

	return fmt.Sprintf("[MCP: %s]\n%s", tool, prettyJSONOrRaw(raw))
}

func normalizeMaybeJSON(s string) string {
	txt := strings.TrimSpace(s)
	if txt == "" {
		return txt
	}

	var v any
	if json.Unmarshal([]byte(txt), &v) == nil {
		if b, err := json.MarshalIndent(v, "", "  "); err == nil {
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
	if b, err := json.MarshalIndent(v, "", "  "); err == nil {
		return string(bytes.TrimSpace(b))
	}
	return string(raw)
}
