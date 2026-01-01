package client_test

import (
	"errors"
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

		req := api.MCPRequest{
			Endpoint: endpoint,
			Tool:     tool,
			Headers:  map[string]string{},
			Params: map[string]interface{}{
				"mock-param": "mock-value",
			},
		}

		it.Before(func() {
			subject = factory.buildClientWithoutConfig()

			mockMCPTransport = NewMockMCPTransport(mockCtrl)
			subject = subject.WithTransport(mockMCPTransport)
			subject = subject.WithContextWindow(1000)
		})

		it("throws an error when history tracking is disabled", func() {
			subject.Config.OmitHistory = true

			err := subject.InjectMCPContext(req)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(client.ErrHistoryTracking))
		})

		it("throws an error when mcp endpoint is missing", func() {
			req.Endpoint = ""

			err := subject.InjectMCPContext(req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mcp endpoint is required"))
		})

		it("throws an error when mcp tool is missing", func() {
			req.Tool = ""

			err := subject.InjectMCPContext(req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mcp tool is required"))
		})

		it("throws an error when the transport call fails", func() {
			msg := "transport error"

			mockMCPTransport.EXPECT().
				Call(endpoint, gomock.Any()).
				Return(api.MCPMessage{}, errors.New(msg))

			err := subject.InjectMCPContext(req)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(msg))
		})

		it("throws an error when history writing fails", func() {
			// MCP response in the new shape: result.content[].text
			resp := api.MCPMessage{
				JSONRPC: "2.0",
				ID:      "2",
				Result:  []byte(`{"content":[{"type":"text","text":"ok"}]}`),
			}

			mockMCPTransport.EXPECT().
				Call(endpoint, gomock.Any()).
				Return(resp, nil)

			mockHistoryStore.EXPECT().Read().Times(1)
			mockTimer.EXPECT().Now().Times(2)

			msg := "write error"
			mockHistoryStore.EXPECT().Write(gomock.Any()).Return(errors.New(msg))

			err := subject.InjectMCPContext(req)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(msg))
		})

		it("adds the formatted MCP response to history (single text block, JSON string gets pretty-printed)", func() {
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
				Call(endpoint, gomock.Any()).
				Return(resp, nil)

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

			err := subject.InjectMCPContext(req)
			Expect(err).NotTo(HaveOccurred())
		})

		it("joins multiple text blocks with a blank line between them", func() {
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
				Call(endpoint, gomock.Any()).
				Return(resp, nil)

			mockHistoryStore.EXPECT().Read().Times(1)
			mockTimer.EXPECT().Now().Times(2)

			mockHistoryStore.EXPECT().Write(gomock.Any()).
				DoAndReturn(func(h []history.History) error {
					last := h[len(h)-1].Message.Content
					Expect(last).To(ContainSubstring("[MCP: " + tool + "]"))
					Expect(last).To(ContainSubstring("first\n\nsecond"))
					return nil
				})

			err := subject.InjectMCPContext(req)
			Expect(err).NotTo(HaveOccurred())
		})

		it("falls back to '(empty result)' when resp.Result is empty", func() {
			resp := api.MCPMessage{
				JSONRPC: "2.0",
				ID:      "2",
				Result:  nil,
			}

			mockMCPTransport.EXPECT().
				Call(endpoint, gomock.Any()).
				Return(resp, nil)

			mockHistoryStore.EXPECT().Read().Times(1)
			mockTimer.EXPECT().Now().Times(2)

			mockHistoryStore.EXPECT().Write(gomock.Any()).
				DoAndReturn(func(h []history.History) error {
					Expect(h[len(h)-1].Message.Content).To(ContainSubstring("(empty result)"))
					return nil
				})

			err := subject.InjectMCPContext(req)
			Expect(err).NotTo(HaveOccurred())
		})

		// This is only here to keep the file's imports stable if your shared setup
		// ever stops using time directly.
		_ = time.Time{}
	})
}
