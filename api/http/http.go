package http

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/config"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"strings"
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
	Post(url string, body []byte, stream bool) ([]byte, error)
	PostWithHeaders(url string, body []byte, headers map[string]string) ([]byte, error)
	Get(url string) ([]byte, error)
}

type RestCaller struct {
	client *http.Client
	config config.Config
}

// Ensure RestCaller implements Caller interface
var _ Caller = &RestCaller{}

func New(cfg config.Config) *RestCaller {
	var client *http.Client
	if cfg.SkipTLSVerify {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{
			Transport: transport,
		}
	} else {
		client = &http.Client{}
	}

	return &RestCaller{
		client: client,
		config: cfg,
	}
}

type CallerFactory func(cfg config.Config) Caller

func RealCallerFactory(cfg config.Config) Caller {
	return New(cfg)
}

func (r *RestCaller) Get(url string) ([]byte, error) {
	return r.doRequest(http.MethodGet, url, nil, false)
}

func (r *RestCaller) Post(url string, body []byte, stream bool) ([]byte, error) {
	return r.doRequest(http.MethodPost, url, body, stream)
}

func (r *RestCaller) PostWithHeaders(url string, body []byte, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf(errFailedToCreateRequest, err)
	}

	// Add custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errFailedToMakeRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorResponse, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf(errHTTPStatus, resp.StatusCode)
		}

		var errorData api.ErrorResponse
		if err := json.Unmarshal(errorResponse, &errorData); err != nil {
			return nil, fmt.Errorf(errHTTPStatus, resp.StatusCode)
		}

		return errorResponse, fmt.Errorf(errHTTP, resp.StatusCode, errorData.Error.Message)
	}

	return io.ReadAll(resp.Body)
}

func (r *RestCaller) ProcessResponse(reader io.Reader, writer io.Writer) []byte {
	var result []byte

	sugar := zap.S()
	sugar.Debugln("\nResponse\n")

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		if zap.L().Core().Enabled(zap.DebugLevel) {
			sugar.Debugln(line)
			continue
		}

		if strings.HasPrefix(line, "data:") {
			line = line[6:] // Skip the "data: " prefix
			if len(line) < 6 {
				continue
			}

			if line == "[DONE]" {
				_, _ = writer.Write([]byte("\n"))
				result = append(result, []byte("\n")...)
				break
			}

			var data api.Data
			err := json.Unmarshal([]byte(line), &data)
			if err != nil {
				_, _ = fmt.Fprintf(writer, "Error: %s\n", err.Error())
				continue
			}

			for _, choice := range data.Choices {
				if content, ok := choice.Delta["content"].(string); ok {
					_, _ = writer.Write([]byte(content))
					result = append(result, []byte(content)...)
				}
			}
		}
	}
	return result
}

func (r *RestCaller) doRequest(method, url string, body []byte, stream bool) ([]byte, error) {
	req, err := r.newRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf(errFailedToCreateRequest, err)
	}

	response, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errFailedToMakeRequest, err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		errorResponse, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf(errHTTPStatus, response.StatusCode)
		}

		var errorData api.ErrorResponse
		if err := json.Unmarshal(errorResponse, &errorData); err != nil {
			return nil, fmt.Errorf(errHTTPStatus, response.StatusCode)
		}

		return errorResponse, fmt.Errorf(errHTTP, response.StatusCode, errorData.Error.Message)
	}

	if stream {
		return r.ProcessResponse(response.Body, os.Stdout), nil
	}

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf(errFailedToRead, err)
	}

	return result, nil
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
