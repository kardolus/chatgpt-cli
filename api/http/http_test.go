package http_test

import (
	"bytes"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kardolus/chatgpt-cli/api/http"
	chatgpthttp "github.com/kardolus/chatgpt-cli/api/http"
	"github.com/kardolus/chatgpt-cli/config"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitHTTP(t *testing.T) {
	spec.Run(t, "Testing the HTTP Client", testHTTP, spec.Report(report.Terminal{}))
}

func testHTTP(t *testing.T, when spec.G, it spec.S) {
	var subject http.RestCaller

	const responsesPath = "/v1/responses" // use http.ResponsesPath if you export it

	it.Before(func() {
		RegisterTestingT(t)
		subject = http.RestCaller{}
	})

	when("ProcessResponse()", func() {
		it("parses a legacy stream as expected (any endpoint)", func() {
			buf := &bytes.Buffer{}
			// legacy works via both branches; use a non-responses endpoint to
			// ensure we exercise the original/legacy code path.
			subject.ProcessResponse(strings.NewReader(legacyStream), buf, "/v1/chat/completions")
			output := buf.String()
			Expect(output).To(Equal("a b c\n"))
		})

		it("parses a GPT-5 SSE stream when endpoint is /v1/responses", func() {
			buf := &bytes.Buffer{}
			subject.ProcessResponse(strings.NewReader(gpt5Stream), buf, responsesPath)
			output := buf.String()
			// deltas are "a", " b", " c" then response.completed -> newline
			Expect(output).To(Equal("a b c\n"))
		})

		it("throws an error when the legacy json is invalid", func() {
			input := `data: {"invalid":"json"` // missing closing brace
			expectedOutput := "Error: unexpected end of JSON input\n"

			var buf bytes.Buffer
			subject.ProcessResponse(strings.NewReader(input), &buf, "/v1/chat/completions")
			output := buf.String()
			Expect(output).To(Equal(expectedOutput))
		})
	})

	when("Get()", func() {
		it("returns the response body on success", func() {
			t.Parallel()

			server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				Expect(r.Method).To(Equal(stdhttp.MethodGet))
				w.WriteHeader(stdhttp.StatusOK)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer server.Close()

			subject := chatgpthttp.New(config.Config{})

			body, err := subject.Get(server.URL)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(Equal(`{"ok":true}`))
		})
	})

	when("Post()", func() {
		it("POSTs the body and returns the response body on success", func() {
			t.Parallel()

			server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				Expect(r.Method).To(Equal(stdhttp.MethodPost))

				b, err := io.ReadAll(r.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(b)).To(Equal(`{"hello":"world"}`))

				w.WriteHeader(stdhttp.StatusOK)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer server.Close()

			subject := chatgpthttp.New(config.Config{})

			out, err := subject.Post(server.URL, []byte(`{"hello":"world"}`), false)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal(`{"ok":true}`))
		})
	})

	when("PostWithHeaders()", func() {
		it("attaches headers and returns the response body on success", func() {
			t.Parallel()

			server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				Expect(r.Method).To(Equal(stdhttp.MethodPost))
				Expect(r.Header.Get("X-Test")).To(Equal("abc"))

				w.WriteHeader(stdhttp.StatusOK)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer server.Close()

			subject := chatgpthttp.New(config.Config{})

			out, err := subject.PostWithHeaders(server.URL, []byte(`{}`), map[string]string{
				"X-Test": "abc",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal(`{"ok":true}`))
		})

		it("returns body + error on non-2xx with OpenAI-style error json", func() {
			t.Parallel()

			server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				w.WriteHeader(stdhttp.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":{"message":"nope"}}`))
			}))
			defer server.Close()

			subject := chatgpthttp.New(config.Config{})

			out, err := subject.PostWithHeaders(server.URL, []byte(`{}`), nil)
			Expect(err).To(HaveOccurred())
			Expect(string(out)).To(ContainSubstring(`"nope"`))
		})
	})

	when("PostWithHeadersResponse()", func() {
		it("returns status, headers, and body on success", func() {
			t.Parallel()

			server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				Expect(r.Method).To(Equal(stdhttp.MethodPost))
				Expect(r.Header.Get("X-Test")).To(Equal("abc"))

				w.Header().Set("mcp-session-id", "sid-123")
				w.WriteHeader(stdhttp.StatusOK)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer server.Close()

			rc := http.RestCaller{} // ✅ no NewRestCaller

			resp, err := rc.PostWithHeadersResponse(server.URL, []byte(`{"hello":"world"}`), map[string]string{
				"X-Test": "abc",
			})

			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Status).To(Equal(stdhttp.StatusOK))
			Expect(resp.Headers).NotTo(BeNil())
			Expect(string(resp.Body)).To(Equal(`{"ok":true}`))

			v, ok := headerGetCI(resp.Headers, "mcp-session-id")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("sid-123"))
		})

		it("returns response + error on non-2xx with OpenAI-style error json", func() {
			t.Parallel()

			server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				w.Header().Set("mcp-session-id", "sid-rotated")
				w.WriteHeader(stdhttp.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":{"message":"nope"}}`))
			}))
			defer server.Close()

			rc := http.RestCaller{}

			resp, err := rc.PostWithHeadersResponse(server.URL, []byte(`{}`), nil)

			Expect(err).To(HaveOccurred())

			Expect(resp.Status).To(Equal(stdhttp.StatusUnauthorized))
			Expect(resp.Headers).NotTo(BeNil())
			Expect(resp.Body).NotTo(BeEmpty())

			v, ok := headerGetCI(resp.Headers, "mcp-session-id")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("sid-rotated"))
		})
	})
}

const legacyStream = `
data: {"id":"id-1","object":"chat.completion.chunk","created":1,"model":"model-1","choices":[{"delta":{"role":"assistant"},"index":0,"finish_reason":null}]}

data: {"id":"id-2","object":"chat.completion.chunk","created":2,"model":"model-1","choices":[{"delta":{"content":"a"},"index":0,"finish_reason":null}]}

data: {"id":"id-3","object":"chat.completion.chunk","created":3,"model":"model-1","choices":[{"delta":{"content":" b"},"index":0,"finish_reason":null}]}

data: {"id":"id-4","object":"chat.completion.chunk","created":4,"model":"model-1","choices":[{"delta":{"content":" c"},"index":0,"finish_reason":null}]}

data: {"id":"id-5","object":"chat.completion.chunk","created":5,"model":"model-1","choices":[{"delta":{},"index":0,"finish_reason":"stop"}]}

data: [DONE]
`

// Minimal GPT-5 SSE that your new parser should handle
const gpt5Stream = `
event: response.created
data: {"type":"response.created"}

event: response.output_item.added
data: {"type":"response.output_item.added","output_index":0,"item":{"id":"msg_1","type":"message","status":"in_progress","content":[],"role":"assistant"}}

event: response.content_part.added
data: {"type":"response.content_part.added","item_id":"msg_1","output_index":0,"content_index":0,"part":{"type":"output_text","annotations":[],"logprobs":[],"text":""}}

event: response.output_text.delta
data: {"type":"response.output_text.delta","item_id":"msg_1","output_index":0,"content_index":0,"delta":"a"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","item_id":"msg_1","output_index":0,"content_index":0,"delta":" b"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","item_id":"msg_1","output_index":0,"content_index":0,"delta":" c"}

event: response.completed
data: {"type":"response.completed","response":{"status":"completed"}}
`

func TestUnitCustomHeaders(t *testing.T) {
	spec.Run(t, "Testing Custom Headers", testCustomHeaders, spec.Report(report.Terminal{}))
}

func testCustomHeaders(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("custom headers are configured", func() {
		it("attaches custom headers to POST requests", func() {
			t.Parallel()

			var receivedHeaders stdhttp.Header
			server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				receivedHeaders = r.Header
				w.WriteHeader(stdhttp.StatusOK)
				_, _ = w.Write([]byte(`{"success": true}`))
			}))
			defer server.Close()

			cfg := config.Config{
				CustomHeaders: map[string]string{
					"X-Custom-Header":  "custom-value",
					"X-Another-Header": "another-value",
				},
			}

			subject := chatgpthttp.New(cfg)
			_, err := subject.Post(server.URL, []byte(`{"test": "data"}`), false)

			Expect(err).ToNot(HaveOccurred())
			Expect(receivedHeaders.Get("X-Custom-Header")).To(Equal("custom-value"))
			Expect(receivedHeaders.Get("X-Another-Header")).To(Equal("another-value"))
		})

		it("attaches custom headers to GET requests", func() {
			t.Parallel()

			var receivedHeaders stdhttp.Header
			server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				receivedHeaders = r.Header
				w.WriteHeader(stdhttp.StatusOK)
				_, _ = w.Write([]byte(`{"success": true}`))
			}))
			defer server.Close()

			cfg := config.Config{
				CustomHeaders: map[string]string{
					"X-API-Version": "v2",
					"X-Client-ID":   "test-client",
				},
			}

			subject := chatgpthttp.New(cfg)
			_, err := subject.Get(server.URL)

			Expect(err).ToNot(HaveOccurred())
			Expect(receivedHeaders.Get("X-API-Version")).To(Equal("v2"))
			Expect(receivedHeaders.Get("X-Client-ID")).To(Equal("test-client"))
		})

		it("works with empty custom headers map", func() {
			t.Parallel()

			var receivedHeaders stdhttp.Header
			server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				receivedHeaders = r.Header
				w.WriteHeader(stdhttp.StatusOK)
				_, _ = w.Write([]byte(`{"success": true}`))
			}))
			defer server.Close()

			cfg := config.Config{
				CustomHeaders: map[string]string{},
			}

			subject := chatgpthttp.New(cfg)
			_, err := subject.Post(server.URL, []byte(`{"test": "data"}`), false)

			Expect(err).ToNot(HaveOccurred())
			Expect(receivedHeaders).ToNot(BeNil())
		})

		it("works with nil custom headers map", func() {
			t.Parallel()

			var receivedHeaders stdhttp.Header
			server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				receivedHeaders = r.Header
				w.WriteHeader(stdhttp.StatusOK)
				_, _ = w.Write([]byte(`{"success": true}`))
			}))
			defer server.Close()

			cfg := config.Config{
				CustomHeaders: nil,
			}

			subject := chatgpthttp.New(cfg)
			_, err := subject.Post(server.URL, []byte(`{"test": "data"}`), false)

			Expect(err).ToNot(HaveOccurred())
			Expect(receivedHeaders).ToNot(BeNil())
		})

		it("does not override standard headers with custom headers", func() {
			t.Parallel()

			var receivedHeaders stdhttp.Header
			server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				receivedHeaders = r.Header
				w.WriteHeader(stdhttp.StatusOK)
				_, _ = w.Write([]byte(`{"success": true}`))
			}))
			defer server.Close()

			cfg := config.Config{
				APIKey:          "test-key",
				AuthHeader:      "Authorization",
				AuthTokenPrefix: "Bearer ",
				UserAgent:       "TestAgent/1.0",
				CustomHeaders: map[string]string{
					"X-Custom-Header": "custom-value",
				},
			}

			subject := chatgpthttp.New(cfg)
			_, err := subject.Post(server.URL, []byte(`{"test": "data"}`), false)

			Expect(err).ToNot(HaveOccurred())
			Expect(receivedHeaders.Get("Authorization")).To(Equal("Bearer test-key"))
			Expect(receivedHeaders.Get("User-Agent")).To(Equal("TestAgent/1.0"))
			Expect(receivedHeaders.Get("X-Custom-Header")).To(Equal("custom-value"))
		})
	})
}

func headerGetCI(h map[string]string, key string) (string, bool) {
	for k, v := range h {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return "", false
}
