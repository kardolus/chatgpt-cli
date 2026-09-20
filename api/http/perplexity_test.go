package http

import (
	"context"
	"net/http"
	"testing"

	"github.com/kardolus/chatgpt-cli/config"
)

func TestPerplexityIntegrationHeader(t *testing.T) {
	tests := []struct {
		name, url, want string
		customHeaders   map[string]string
	}{
		{"Perplexity API", "https://api.perplexity.ai/chat/completions", "chatgpt-cli", nil},
		{"caller override", "https://api.perplexity.ai/chat/completions", "custom", map[string]string{"x-pplx-integration": "custom"}},
		{"lookalike host", "https://api.perplexity.ai.example.com/chat/completions", "", nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caller := RestCaller{config: config.Config{CustomHeaders: test.customHeaders}}
			request, err := caller.newRequest(context.Background(), http.MethodPost, test.url, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := request.Header.Get("X-Pplx-Integration"); got != test.want {
				t.Fatalf("X-Pplx-Integration = %q, want %q", got, test.want)
			}
		})
	}
}
