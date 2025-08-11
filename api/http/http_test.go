package http_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kardolus/chatgpt-cli/api/http"
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
