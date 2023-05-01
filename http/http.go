package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Caller interface {
	Post(url string, body []byte) ([]byte, error)
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

func (r *RestCaller) Post(url string, body []byte) ([]byte, error) {
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
		result, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		return result, nil
	}

	return nil, fmt.Errorf("http error: %d", response.StatusCode)
}
