package client_test

import (
	"errors"
	"github.com/kardolus/chatgpt-cli/api/http"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/api/client"
	"github.com/kardolus/chatgpt-cli/history"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

// This file only contains MCP-specific tests.
// It relies on the shared test setup (mocks/factory/config) that lives in client_test.go.

func testMCP(t *testing.T, when spec.G, it spec.S) {
	when("InjectMCPContext()", func() {
		var (
			subject          *client.Client
			mockMCPTransport *MockMCPTransport
		)

		const (
			tool     = "mock-tool"
			endpoint = "https://example.com/mcp"
		)

		newReq := func() api.MCPRequest {
			return api.MCPRequest{
				Endpoint: endpoint,
				Tool:     tool,
				Headers:  map[string]string{},
				Params: map[string]interface{}{
					"mock-param": "mock-value",
				},
			}
		}

		it.Before(func() {
			subject = factory.buildClientWithoutConfig()

			mockMCPTransport = NewMockMCPTransport(mockCtrl)
			subject = subject.WithTransport(mockMCPTransport)
			subject = subject.WithContextWindow(1000)
		})

		it("throws an error when history tracking is disabled", func() {
			subject.Config.OmitHistory = true

			err := subject.InjectMCPContext(newReq())
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(client.ErrHistoryTracking))
		})

		it("throws an error when mcp endpoint is missing", func() {
			r := newReq()
			r.Endpoint = ""

			err := subject.InjectMCPContext(r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mcp endpoint is required"))
		})

		it("throws an error when mcp tool is missing", func() {
			r := newReq()
			r.Tool = ""

			err := subject.InjectMCPContext(r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mcp tool is required"))
		})

		it("throws an error when the transport call fails", func() {
			r := newReq()
			msg := "transport error"

			mockMCPTransport.EXPECT().
				Call(endpoint, gomock.Any(), r.Headers).
				Return(api.MCPResponse{}, errors.New(msg))

			err := subject.InjectMCPContext(r)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(msg))
		})

		it("throws an error when history writing fails", func() {
			r := newReq()

			resp := api.MCPMessage{
				JSONRPC: "2.0",
				ID:      "2",
				Result:  []byte(`{"content":[{"type":"text","text":"ok"}]}`),
			}

			mockMCPTransport.EXPECT().
				Call(endpoint, gomock.Any(), gomock.Any()).
				Return(api.MCPResponse{Message: resp}, nil)

			mockHistoryStore.EXPECT().Read().Times(1)
			mockTimer.EXPECT().Now().Times(2)

			msg := "write error"
			mockHistoryStore.EXPECT().Write(gomock.Any()).Return(errors.New(msg))

			err := subject.InjectMCPContext(r)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(msg))
		})

		it("adds the formatted MCP response to history (single text block, JSON string gets pretty-printed)", func() {
			r := newReq()

			resp := api.MCPMessage{
				JSONRPC: "2.0",
				ID:      "2",
				Result: []byte(`{
					"content": [
						{"type": "text", "text": "[{\"temperature\":\"15C\",\"condition\":\"Sunny\"}]"}
					]
				}`),
			}

			mockMCPTransport.EXPECT().
				Call(endpoint, gomock.Any(), gomock.Any()).
				Return(api.MCPResponse{Message: resp}, nil)

			mockHistoryStore.EXPECT().Read().Times(1)
			mockTimer.EXPECT().Now().Times(2)

			mockHistoryStore.EXPECT().Write(gomock.Any()).
				DoAndReturn(func(h []history.History) error {
					Expect(h).NotTo(BeEmpty())
					last := h[len(h)-1]

					Expect(last.Message.Role).To(Equal(client.AssistantRole))
					Expect(last.Message.Name).To(BeEmpty()) // name is no longer set
					Expect(last.Message.Content).To(ContainSubstring("[MCP: " + tool + "]"))

					// Pretty-printed JSON produced by normalizeMaybeJSON()
					Expect(last.Message.Content).To(ContainSubstring(`"temperature": "15C"`))
					Expect(last.Message.Content).To(ContainSubstring(`"condition": "Sunny"`))

					return nil
				})

			err := subject.InjectMCPContext(r)
			Expect(err).NotTo(HaveOccurred())
		})

		it("joins multiple text blocks with a blank line between them", func() {
			r := newReq()

			resp := api.MCPMessage{
				JSONRPC: "2.0",
				ID:      "2",
				Result: []byte(`{
					"content": [
						{"type":"text","text":"first"},
						{"type":"text","text":"second"}
					]
				}`),
			}

			mockMCPTransport.EXPECT().
				Call(endpoint, gomock.Any(), gomock.Any()).
				Return(api.MCPResponse{Message: resp}, nil)

			mockHistoryStore.EXPECT().Read().Times(1)
			mockTimer.EXPECT().Now().Times(2)

			mockHistoryStore.EXPECT().Write(gomock.Any()).
				DoAndReturn(func(h []history.History) error {
					last := h[len(h)-1].Message.Content
					Expect(last).To(ContainSubstring("[MCP: " + tool + "]"))
					Expect(last).To(ContainSubstring("first\n\nsecond"))
					return nil
				})

			err := subject.InjectMCPContext(r)
			Expect(err).NotTo(HaveOccurred())
		})

		it("falls back to '(empty result)' when resp.Result is empty", func() {
			r := newReq()

			resp := api.MCPMessage{
				JSONRPC: "2.0",
				ID:      "2",
				Result:  nil,
			}

			mockMCPTransport.EXPECT().
				Call(endpoint, gomock.Any(), gomock.Any()).
				Return(api.MCPResponse{Message: resp}, nil)

			mockHistoryStore.EXPECT().Read().Times(1)
			mockTimer.EXPECT().Now().Times(2)

			mockHistoryStore.EXPECT().Write(gomock.Any()).
				DoAndReturn(func(h []history.History) error {
					Expect(h[len(h)-1].Message.Content).To(ContainSubstring("(empty result)"))
					return nil
				})

			err := subject.InjectMCPContext(r)
			Expect(err).NotTo(HaveOccurred())
		})

		_ = time.Time{}
	})
}

func testSessionTransport(t *testing.T, when spec.G, it spec.S) {
	when("SessionTransport.Call()", func() {
		var (
			endpoint string
			store    *fakeSessionStore
			inner    *fakeMCPTransport
			subject  *client.SessionTransport
		)

		it.Before(func() {
			RegisterTestingT(t)

			endpoint = "https://example.com/mcp"
			store = newFakeSessionStore()
			inner = &fakeMCPTransport{}
			subject = client.NewSessionTransport(inner, store)
		})

		it("passthrough when caller explicitly sets mcp-session-id header (no store access)", func() {
			req := api.MCPMessage{JSONRPC: "2.0", ID: "1", Method: "tools/call", Params: []byte(`{}`)}

			headers := map[string]string{"Mcp-Session-Id": "explicit-sid"}

			inner.handler = func(ep string, r api.MCPMessage, h map[string]string) (api.MCPResponse, error) {
				Expect(ep).To(Equal(endpoint))
				Expect(r.Method).To(Equal("tools/call"))
				Expect(headerGetCI(h, "mcp-session-id")).To(Equal("explicit-sid"))
				return api.MCPResponse{Status: 200, Headers: map[string]string{}}, nil
			}

			_, err := subject.Call(endpoint, req, headers)
			Expect(err).NotTo(HaveOccurred())

			Expect(store.getCalls).To(Equal(0))
			Expect(store.setCalls).To(Equal(0))
			Expect(store.delCalls).To(Equal(0))
		})

		it("attaches cached session id when caller did not provide one", func() {
			store.sessions[endpoint] = "cached-sid"

			req := api.MCPMessage{JSONRPC: "2.0", ID: "1", Method: "tools/call", Params: []byte(`{}`)}
			headers := map[string]string{}

			inner.handler = func(ep string, r api.MCPMessage, h map[string]string) (api.MCPResponse, error) {
				Expect(headerGetCI(h, "mcp-session-id")).To(Equal("cached-sid"))
				return api.MCPResponse{Status: 200, Headers: map[string]string{}}, nil
			}

			_, err := subject.Call(endpoint, req, headers)
			Expect(err).NotTo(HaveOccurred())

			Expect(store.getCalls).To(Equal(1))
		})

		it("stores rotated session id when server returns mcp-session-id header", func() {
			store.sessions[endpoint] = "cached-sid"

			req := api.MCPMessage{JSONRPC: "2.0", ID: "1", Method: "tools/call", Params: []byte(`{}`)}
			headers := map[string]string{}

			inner.handler = func(ep string, r api.MCPMessage, h map[string]string) (api.MCPResponse, error) {
				Expect(headerGetCI(h, "mcp-session-id")).To(Equal("cached-sid"))
				return api.MCPResponse{
					Status:  200,
					Headers: map[string]string{"mcp-session-id": "rotated-sid"},
				}, nil
			}

			_, err := subject.Call(endpoint, req, headers)
			Expect(err).NotTo(HaveOccurred())

			Expect(store.sessions[endpoint]).To(Equal("rotated-sid"))
			Expect(store.setCalls).To(Equal(1))
		})

		it("on invalid session: deletes cached session, initializes, retries once with new session", func() {
			store.sessions[endpoint] = "old-sid"

			origReq := api.MCPMessage{JSONRPC: "2.0", ID: "orig", Method: "tools/call", Params: []byte(`{}`)}
			headers := map[string]string{}

			callCount := 0

			inner.handler = func(ep string, r api.MCPMessage, h map[string]string) (api.MCPResponse, error) {
				callCount++

				switch callCount {
				case 1:
					// First attempt uses cached sid and fails with "invalid session"
					Expect(r.Method).To(Equal("tools/call"))
					Expect(headerGetCI(h, "mcp-session-id")).To(Equal("old-sid"))
					return api.MCPResponse{
						Status:  400,
						Headers: map[string]string{},
						Message: api.MCPMessage{
							JSONRPC: "2.0",
							ID:      "server-error",
							Error: &api.MCPError{
								Message: "Bad Request: No valid session ID provided",
								Code:    "-32600",
							},
						},
					}, errors.New("Bad Request: No valid session ID provided")
				case 2:
					// initialize call should happen next (no session header)
					Expect(r.Method).To(Equal("initialize"))
					_, ok := headerGetCIok(h, "mcp-session-id")
					Expect(ok).To(BeFalse())

					return api.MCPResponse{
						Status:  200,
						Headers: map[string]string{"mcp-session-id": "new-sid"},
						Message: api.MCPMessage{
							JSONRPC: "2.0",
							ID:      r.ID,
							Result:  []byte(`{}`),
						},
					}, nil
				case 3:
					// retry original request with new session id
					Expect(r.Method).To(Equal("tools/call"))
					Expect(headerGetCI(h, "mcp-session-id")).To(Equal("new-sid"))

					return api.MCPResponse{
						Status:  200,
						Headers: map[string]string{},
						Message: api.MCPMessage{
							JSONRPC: "2.0",
							ID:      "ok",
							Result:  []byte(`{"content":[{"type":"text","text":"ok"}]}`),
						},
					}, nil
				default:
					return api.MCPResponse{}, errors.New("unexpected extra call")
				}
			}

			resp, err := subject.Call(endpoint, origReq, headers)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Status).To(Equal(200))

			Expect(store.delCalls).To(Equal(1))
			Expect(store.sessions[endpoint]).To(Equal("new-sid"))
			Expect(callCount).To(Equal(3))
		})

		it("errors if initialize succeeds but does not return mcp-session-id header", func() {
			// no cached session
			req := api.MCPMessage{JSONRPC: "2.0", ID: "orig", Method: "tools/call", Params: []byte(`{}`)}
			headers := map[string]string{}

			callCount := 0
			inner.handler = func(ep string, r api.MCPMessage, h map[string]string) (api.MCPResponse, error) {
				callCount++
				if callCount == 1 {
					Expect(r.Method).To(Equal("initialize"))
					return api.MCPResponse{
						Status:  200,
						Headers: map[string]string{}, // missing session header
						Message: api.MCPMessage{JSONRPC: "2.0", ID: r.ID, Result: []byte(`{}`)},
					}, nil
				}
				return api.MCPResponse{}, errors.New("should not reach retry")
			}

			_, err := subject.Call(endpoint, req, headers)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("did not return"))
		})
	})
}

func testSessionTransportNonHTTP(t *testing.T, when spec.G, it spec.S) {
	when("SessionTransport.Call() with non-http scheme", func() {
		var (
			endpoint string
			store    *fakeSessionStore
			inner    *fakeMCPTransport
			subject  *client.SessionTransport
		)

		it.Before(func() {
			RegisterTestingT(t)

			endpoint = "stdio:python test/mcp/stdio/mcp_stdio_server.py"
			store = newFakeSessionStore()
			inner = &fakeMCPTransport{}
			subject = client.NewSessionTransport(inner, store)
		})

		it("bypasses session logic and does not touch the session store", func() {
			req := api.MCPMessage{JSONRPC: "2.0", ID: "1", Method: "tools/call", Params: []byte(`{}`)}
			headers := map[string]string{}

			inner.handler = func(ep string, r api.MCPMessage, h map[string]string) (api.MCPResponse, error) {
				Expect(ep).To(Equal(endpoint))
				Expect(r.Method).To(Equal("tools/call"))
				// No session header should be injected for non-http transports.
				_, ok := headerGetCIok(h, "mcp-session-id")
				Expect(ok).To(BeFalse())

				return api.MCPResponse{
					Status:  0,
					Headers: nil,
					Message: api.MCPMessage{JSONRPC: "2.0", ID: r.ID, Result: []byte(`{}`)},
				}, nil
			}

			_, err := subject.Call(endpoint, req, headers)
			Expect(err).NotTo(HaveOccurred())

			Expect(store.getCalls).To(Equal(0))
			Expect(store.setCalls).To(Equal(0))
			Expect(store.delCalls).To(Equal(0))
		})
	})
}

func testNewMCPTransport(t *testing.T, when spec.G, it spec.S) {
	when("NewMCPTransport()", func() {
		it.Before(func() {
			RegisterTestingT(t)
		})

		it("returns MCPHTTPTransport for http/https endpoints", func() {
			// We don't need to actually call it; we just want to route correctly.
			var caller http.Caller = nil

			tr, err := client.NewMCPTransport("https://example.com/mcp", caller, map[string]string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(tr).To(BeAssignableToTypeOf(&client.MCPHTTPTransport{}))

			tr, err = client.NewMCPTransport("http://example.com/mcp", caller, map[string]string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(tr).To(BeAssignableToTypeOf(&client.MCPHTTPTransport{}))
		})

		it("returns MCPStdioTransport for stdio endpoints", func() {
			var caller http.Caller = nil

			tr, err := client.NewMCPTransport("stdio:python test/mcp/stdio/mcp_stdio_server.py", caller, map[string]string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(tr).To(BeAssignableToTypeOf(&client.MCPStdioTransport{}))
		})

		it("errors for unsupported schemes", func() {
			var caller http.Caller = nil

			_, err := client.NewMCPTransport("ftp://example.com/mcp", caller, map[string]string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported mcp transport"))
		})
	})
}

/* =========================
   Fakes
   ========================= */

type fakeSessionStore struct {
	sessions map[string]string
	getCalls int
	setCalls int
	delCalls int
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{sessions: map[string]string{}}
}

func (s *fakeSessionStore) GetSessionID(endpoint string) (string, error) {
	s.getCalls++
	return s.sessions[endpoint], nil
}

func (s *fakeSessionStore) SetSessionID(endpoint, sessionID string) error {
	s.setCalls++
	s.sessions[endpoint] = sessionID
	return nil
}

func (s *fakeSessionStore) DeleteSessionID(endpoint string) error {
	s.delCalls++
	delete(s.sessions, endpoint)
	return nil
}

type fakeMCPTransport struct {
	handler func(endpoint string, req api.MCPMessage, headers map[string]string) (api.MCPResponse, error)
}

func (t *fakeMCPTransport) Call(endpoint string, req api.MCPMessage, headers map[string]string) (api.MCPResponse, error) {
	if t.handler == nil {
		return api.MCPResponse{}, errors.New("fakeMCPTransport.handler is nil")
	}
	// clone headers to avoid accidental mutation surprises across calls
	h := map[string]string{}
	for k, v := range headers {
		h[k] = v
	}
	return t.handler(endpoint, req, h)
}

/* =========================
   Header helpers (case-insensitive)
   ========================= */

func headerGetCI(h map[string]string, key string) string {
	v, _ := headerGetCIok(h, key)
	return v
}

func headerGetCIok(h map[string]string, key string) (string, bool) {
	for k, v := range h {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return "", false
}
