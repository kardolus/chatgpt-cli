package http

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kardolus/chatgpt-cli/types"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type Caller interface {
	Post(url string, body []byte, stream bool) ([]byte, error)
}

type RestCaller struct {
	client *http.Client
	secret string
}

// Ensure RestCaller implements Caller interface
var _ Caller = &RestCaller{}

func New() *RestCaller {
	return &RestCaller{
		client: &http.Client{},
	}
}

func (r *RestCaller) WithSecret(secret string) *RestCaller {
	r.secret = secret
	return r
}

func (r *RestCaller) Post(url string, body []byte, stream bool) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if r.secret != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.secret))
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		if stream {
			ProcessResponse(response.Body, os.Stdout)
			return nil, nil
		} else {
			result, err := ioutil.ReadAll(response.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response: %w", err)
			}
			return result, nil
		}
	}

	return nil, fmt.Errorf("http error: %d", response.StatusCode)
}

func ProcessResponse(r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			line = line[6:] // Skip the "data: " prefix
			if len(line) < 6 {
				continue
			}

			if line == "[DONE]" {
				_, _ = w.Write([]byte("\n"))
				break
			}

			var data types.Data
			err := json.Unmarshal([]byte(line), &data)
			if err != nil {
				_, _ = fmt.Fprintf(w, "Error: %s\n", err.Error())
				continue
			}

			for _, choice := range data.Choices {
				if content, ok := choice.Delta["content"]; ok {
					_, _ = w.Write([]byte(content))
				}
			}
		}
	}
}
