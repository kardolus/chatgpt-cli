package http_test

import (
	"bytes"
	"github.com/kardolus/chatgpt-cli/http"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitHTTP(t *testing.T) {
	spec.Run(t, "Testing the HTTP Client", testHTTP, spec.Report(report.Terminal{}))
}

func testHTTP(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("ProcessResponse()", func() {
		it("parses a stream as expected", func() {
			buf := &bytes.Buffer{}
			http.ProcessResponse(strings.NewReader(stream), buf)
			output := buf.String()
			Expect(output).To(Equal("a b c\n"))
		})
		it("throws an error when the json is invalid", func() {
			input := `data: {"invalid":"json"` // missing closing brace
			expectedOutput := "Error: unexpected end of JSON input\n"

			var buf bytes.Buffer
			http.ProcessResponse(strings.NewReader(input), &buf)
			output := buf.String()

			Expect(output).To(Equal(expectedOutput))
		})
	})
}

const stream = `
data: {"id":"id-1","object":"chat.completion.chunk","created":1,"model":"model-1","choices":[{"delta":{"role":"assistant"},"index":0,"finish_reason":null}]}

data: {"id":"id-2","object":"chat.completion.chunk","created":2,"model":"model-1","choices":[{"delta":{"content":"a"},"index":0,"finish_reason":null}]}

data: {"id":"id-3","object":"chat.completion.chunk","created":3,"model":"model-1","choices":[{"delta":{"content":" b"},"index":0,"finish_reason":null}]}

data: {"id":"id-4","object":"chat.completion.chunk","created":4,"model":"model-1","choices":[{"delta":{"content":" c"},"index":0,"finish_reason":null}]}

data: {"id":"id-5","object":"chat.completion.chunk","created":5,"model":"model-1","choices":[{"delta":{},"index":0,"finish_reason":"stop"}]}

data: [DONE]
`
