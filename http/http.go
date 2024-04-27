package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kardolus/chatgpt-cli/types"
)

const (
	contentType              = "application/json"
	errFailedToRead          = "failed to read response: %w"
	errFailedToCreateRequest = "failed to create request: %w"
	errFailedToMakeRequest   = "failed to make request: %w"
	errHTTP                  = "http status %d: %s"
	errHTTPStatus            = "http status: %d"
	headerContentType        = "Content-Type"
)

type Caller interface {
	Post(url string, body []byte) (io.ReadCloser, error)
	Get(url string) ([]byte, error)
}

type RestCaller struct {
	client *http.Client
	config types.Config
}

// Ensure RestCaller implements Caller interface
var _ Caller = &RestCaller{}

func New(cfg types.Config) *RestCaller {
	return &RestCaller{
		client: &http.Client{},
		config: cfg,
	}
}

type CallerFactory func(cfg types.Config) Caller

func RealCallerFactory(cfg types.Config) Caller {
	return New(cfg)
}

func (r *RestCaller) Get(url string) ([]byte, error) {
	reader, err := r.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf(errFailedToRead, err)
	}
	return result, nil
}

func (r *RestCaller) Post(url string, body []byte) (io.ReadCloser, error) {
	return r.doRequest(http.MethodPost, url, body)
}

func (r *RestCaller) doRequest(method, url string, body []byte) (io.ReadCloser, error) {
	req, err := r.newRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf(errFailedToCreateRequest, err)
	}

	response, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errFailedToMakeRequest, err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		errorResponse, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf(errHTTPStatus, response.StatusCode)
		}

		var errorData types.ErrorResponse
		if err := json.Unmarshal(errorResponse, &errorData); err != nil {
			return nil, fmt.Errorf(errHTTPStatus, response.StatusCode)
		}

		return io.NopCloser(bytes.NewReader(errorResponse)), fmt.Errorf(errHTTP, response.StatusCode, errorData.Error.Message)
	}
	return response.Body, nil
}

func (r *RestCaller) newRequest(method, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	if r.config.APIKey != "" {
		req.Header.Set(r.config.AuthHeader, r.config.AuthTokenPrefix+r.config.APIKey)
	}
	req.Header.Set(headerContentType, contentType)

	return req, nil
}
