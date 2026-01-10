package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"

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
	// Session headers are HTTP-only; stdio (and other non-http schemes) have no headers.
	if u, err := url.Parse(endpoint); err == nil {
		if u.Scheme != "http" && u.Scheme != "https" {
			return t.inner.Call(endpoint, req, headers)
		}
	}

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

func NewMCPTransport(endpoint string, caller http.Caller, headers map[string]string) (MCPTransport, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "http", "https":
		return NewMCPHTTPTransport(endpoint, caller, headers)
	case "stdio":
		return NewMCPStdioTransport(endpoint)
	default:
		return nil, fmt.Errorf("unsupported mcp transport: %s", u.Scheme)
	}
}

/* =========================
   HTTP transport
   ========================= */

type MCPHTTPTransport struct {
	caller  http.Caller
	headers map[string]string
}

func NewMCPHTTPTransport(endpoint string, caller http.Caller, headers map[string]string) (*MCPHTTPTransport, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "http", "https":
		// ok
	default:
		return nil, fmt.Errorf("unsupported mcp http transport: %s", u.Scheme)
	}

	// Defensive copy so callers can reuse/modify their input map safely.
	h := map[string]string{}
	for k, v := range headers {
		h[k] = v
	}

	return &MCPHTTPTransport{
		caller:  caller,
		headers: h,
	}, nil
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
   STDIO transport
   ========================= */

type MCPStdioTransport struct {
	// Endpoint is "stdio:<cmdline...>"
	endpoint string

	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	initialized bool

	// response routing
	pending map[string]chan api.MCPMessage
	done    chan struct{}
}

func NewMCPStdioTransport(endpoint string) (*MCPStdioTransport, error) {
	if !strings.HasPrefix(endpoint, "stdio:") {
		return nil, fmt.Errorf("invalid stdio endpoint: %s", endpoint)
	}
	return &MCPStdioTransport{endpoint: endpoint}, nil
}

func (t *MCPStdioTransport) Call(endpoint string, req api.MCPMessage, headers map[string]string) (api.MCPResponse, error) {
	// headers are ignored for stdio
	if endpoint != t.endpoint {
		return api.MCPResponse{}, fmt.Errorf("stdio transport called with unexpected endpoint")
	}

	if err := t.ensureStarted(); err != nil {
		return api.MCPResponse{}, err
	}
	if err := t.ensureInitialized(); err != nil {
		return api.MCPResponse{}, err
	}

	if strings.TrimSpace(req.ID) == "" {
		req.ID = uuid.NewString()
	}

	msg, err := t.roundTrip(req, 30*time.Second)
	out := api.MCPResponse{
		Message: msg,
		Status:  0,
		Headers: nil,
	}
	return out, err
}

func (t *MCPStdioTransport) ensureStarted() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cmd != nil {
		return nil
	}

	cmdline := strings.TrimSpace(strings.TrimPrefix(t.endpoint, "stdio:"))
	if cmdline == "" {
		return fmt.Errorf("stdio endpoint missing command: %s", t.endpoint)
	}

	argv, err := splitCommandLine(cmdline)
	if err != nil {
		return err
	}
	if len(argv) == 0 {
		return fmt.Errorf("stdio endpoint missing command: %s", t.endpoint)
	}

	cmd := exec.Command(argv[0], argv[1:]...) // #nosec G204
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mcp stdio server: %w", err)
	}

	t.cmd = cmd
	t.stdin = stdin
	t.stdout = stdout
	t.stderr = stderr
	t.pending = map[string]chan api.MCPMessage{}
	t.done = make(chan struct{})

	go t.readLoop()
	go t.drainStderr()

	return nil
}

func (t *MCPStdioTransport) ensureInitialized() error {
	t.mu.Lock()
	if t.initialized {
		t.mu.Unlock()
		return nil
	}
	t.mu.Unlock()

	initParams, _ := json.Marshal(map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "chatgpt-cli",
			"version": "dev",
		},
	})

	initReq := api.MCPMessage{
		JSONRPC: "2.0",
		ID:      uuid.NewString(),
		Method:  "initialize",
		Params:  initParams,
	}

	if _, err := t.roundTrip(initReq, 10*time.Second); err != nil {
		return err
	}

	// notifications/initialized (no response expected)
	notif := api.MCPMessage{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	if err := t.sendOneWay(notif); err != nil {
		return err
	}

	t.mu.Lock()
	t.initialized = true
	t.mu.Unlock()
	return nil
}

func (t *MCPStdioTransport) sendOneWay(msg api.MCPMessage) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	_, err = t.stdin.Write(append(b, '\n'))
	return err
}

func (t *MCPStdioTransport) roundTrip(req api.MCPMessage, timeout time.Duration) (api.MCPMessage, error) {
	ch := make(chan api.MCPMessage, 1)

	t.mu.Lock()
	t.pending[req.ID] = ch

	b, err := json.Marshal(req)
	if err == nil {
		_, err = t.stdin.Write(append(b, '\n'))
	}
	t.mu.Unlock()

	if err != nil {
		t.mu.Lock()
		delete(t.pending, req.ID)
		t.mu.Unlock()
		return api.MCPMessage{}, fmt.Errorf("failed to write to mcp stdio: %w", err)
	}

	select {
	case msg, ok := <-ch:
		if !ok {
			return api.MCPMessage{}, fmt.Errorf("mcp stdio server closed")
		}
		if msg.Error != nil {
			return msg, msg.Error
		}
		return msg, nil
	case <-time.After(timeout):
		t.mu.Lock()
		delete(t.pending, req.ID)
		t.mu.Unlock()
		return api.MCPMessage{}, fmt.Errorf("mcp stdio call timed out")
	}
}

func (t *MCPStdioTransport) readLoop() {
	defer close(t.done)

	scanner := bufio.NewScanner(t.stdout)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 5*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var msg api.MCPMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		// ignore notifications (no id)
		if strings.TrimSpace(msg.ID) == "" {
			continue
		}

		t.mu.Lock()
		ch := t.pending[msg.ID]
		if ch != nil {
			delete(t.pending, msg.ID)
		}
		t.mu.Unlock()

		if ch != nil {
			ch <- msg
			close(ch)
		}
	}

	// server exited: unblock all waiters
	t.mu.Lock()
	for id, ch := range t.pending {
		delete(t.pending, id)
		close(ch)
	}
	t.mu.Unlock()
}

func (t *MCPStdioTransport) drainStderr() {
	r := bufio.NewReader(t.stderr)
	for {
		_, err := r.ReadString('\n')
		if err != nil {
			return
		}
		// optionally log when debug
	}
}

// Minimal shell-ish arg splitting supporting:
// - spaces
// - single quotes '...'
// - double quotes "..."
// No escapes yet (good enough for v1).
func splitCommandLine(s string) ([]string, error) {
	var out []string
	var cur strings.Builder

	inSingle := false
	inDouble := false

	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}

	for i := 0; i < len(s); i++ {
		ch := s[i]

		switch ch {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
				continue
			}
			cur.WriteByte(ch)
		case '"':
			if !inSingle {
				inDouble = !inDouble
				continue
			}
			cur.WriteByte(ch)
		case ' ', '\t', '\n':
			if inSingle || inDouble {
				cur.WriteByte(ch)
				continue
			}
			flush()
		default:
			cur.WriteByte(ch)
		}
	}

	if inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quote in stdio command")
	}

	flush()
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
