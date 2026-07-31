package http

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/internal"
	"go.uber.org/zap"
)

const (
	errFailedToRead          = "failed to read response: %w"
	errFailedToCreateRequest = "failed to create request: %w"
	errFailedToMakeRequest   = "failed to make request: %w"
	errHTTP                  = "http status %d: %s"
	errHTTPStatus            = "http status: %d"
	errStreamRead            = "stream read error: %w"

	defaultMaxRetries       = 3
	defaultRetryBaseDelayMs = 500
	maxBackoff              = 30 * time.Second
)

type Caller interface {
	Post(ctx context.Context, url string, body []byte, stream bool) ([]byte, error)
	PostWithHeaders(ctx context.Context, url string, body []byte, headers map[string]string) ([]byte, error)
	Get(ctx context.Context, url string) ([]byte, error)
	PostWithHeadersResponse(ctx context.Context, url string, body []byte, headers map[string]string) (api.HTTPResponse, error)
}

type RestCaller struct {
	client *http.Client
	config config.Config
}

// Ensure RestCaller implements Caller interface
var _ Caller = &RestCaller{}

func New(cfg config.Config) *RestCaller {
	client := &http.Client{Timeout: time.Duration(cfg.HTTPTimeout) * time.Second}

	if cfg.SkipTLSVerify {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		client.Transport = transport
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

func (r *RestCaller) Get(ctx context.Context, url string) ([]byte, error) {
	return r.doRequest(ctx, http.MethodGet, url, nil, false)
}

func (r *RestCaller) Post(ctx context.Context, url string, body []byte, stream bool) ([]byte, error) {
	return r.doRequest(ctx, http.MethodPost, url, body, stream)
}

func (r *RestCaller) PostWithHeaders(ctx context.Context, url string, body []byte, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
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

func (r *RestCaller) PostWithHeadersResponse(ctx context.Context, url string, body []byte, headers map[string]string) (api.HTTPResponse, error) {
	// tests construct RestCaller{} (nil client) — avoid panic
	if r.client == nil {
		r.client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return api.HTTPResponse{}, fmt.Errorf(errFailedToCreateRequest, err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return api.HTTPResponse{}, fmt.Errorf(errFailedToMakeRequest, err)
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return api.HTTPResponse{}, readErr
	}

	outHeaders := map[string]string{}
	for k, vals := range resp.Header {
		if len(vals) == 0 {
			continue
		}
		outHeaders[k] = strings.Join(vals, ", ")
	}

	out := api.HTTPResponse{
		Status:  resp.StatusCode,
		Headers: outHeaders,
		Body:    respBody,
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try OpenAI error shape first
		var errorData api.ErrorResponse
		if err := json.Unmarshal(respBody, &errorData); err == nil && errorData.Error.Message != "" {
			return out, fmt.Errorf(errHTTP, resp.StatusCode, errorData.Error.Message)
		}

		// Otherwise include raw body so you can debug MCP server errors
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			return out, fmt.Errorf(errHTTPStatus, resp.StatusCode)
		}
		return out, fmt.Errorf("http status %d: %s", resp.StatusCode, msg)
	}

	return out, nil
}

func (r *RestCaller) ProcessResponse(reader io.Reader, writer io.Writer, endpoint string) ([]byte, error) {
	if strings.Contains(endpoint, r.config.ResponsesPath) {
		return r.processResponsesSSE(reader, writer)
	}
	return r.processLegacy(reader, writer)
}

func (r *RestCaller) processLegacy(reader io.Reader, writer io.Writer) ([]byte, error) {
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
				result = append(result, '\n')
				break
			}
			var data api.Data
			if err := json.Unmarshal([]byte(line), &data); err != nil {
				_, _ = fmt.Fprintf(writer, "Error: %s\n", err.Error())
				continue
			}
			for _, choice := range data.Choices {
				if content, ok := choice.Delta["content"].(string); ok {
					_, _ = writer.Write([]byte(content))
					result = append(result, content...)
				}
			}
		}
	}
	// A non-nil scanner error means the stream was cut short (e.g. a dropped
	// connection); surface it so the caller doesn't treat a truncated response
	// as complete.
	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf(errStreamRead, err)
	}
	return result, nil
}

func (r *RestCaller) processResponsesSSE(reader io.Reader, writer io.Writer) ([]byte, error) {
	var (
		result   []byte
		curEvent string
		done     bool
		sugar    = zap.S()
	)

	sugar.Debugln("\nResponse\n")

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if zap.L().Core().Enabled(zap.DebugLevel) {
			sugar.Debugln(line)
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}

		switch {
		case strings.HasPrefix(line, "event:"):
			curEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue

		case strings.HasPrefix(line, "data:"):
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			if curEvent == "" {
				if payload == "[DONE]" {
					_, _ = writer.Write([]byte("\n"))
					result = append(result, '\n')
					done = true
					break
				}
				var legacy struct {
					Choices []struct {
						Delta map[string]any `json:"delta"`
					} `json:"choices"`
				}
				if err := json.Unmarshal([]byte(payload), &legacy); err != nil {
					_, _ = fmt.Fprintf(writer, "Error: %s\n", err.Error())
					continue
				}
				for _, ch := range legacy.Choices {
					if s, ok := ch.Delta["content"].(string); ok && s != "" {
						_, _ = writer.Write([]byte(s))
						result = append(result, s...)
					}
				}
				continue
			}

			var env struct {
				Type     string `json:"type"`
				Delta    string `json:"delta"` // response.output_text.delta
				Text     string `json:"text"`  // response.output_text.done/content_part.done (optional)
				Response struct {
					Status string `json:"status"`
				} `json:"response"`
			}
			if err := json.Unmarshal([]byte(payload), &env); err != nil {
				_, _ = fmt.Fprintf(writer, "Error: %s\n", err.Error())
				continue
			}

			switch env.Type {
			case "response.output_text.delta":
				if env.Delta != "" {
					_, _ = writer.Write([]byte(env.Delta))
					result = append(result, env.Delta...)
				}
			case "response.completed":
				if len(result) == 0 || !bytes.HasSuffix(result, []byte("\n")) {
					_, _ = writer.Write([]byte("\n"))
					result = append(result, '\n')
				}
				done = true
			default:
				// ignore other SSE types
			}
		}

		if done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf(errStreamRead, err)
	}
	return result, nil
}

func (r *RestCaller) doRequest(ctx context.Context, method, url string, body []byte, stream bool) ([]byte, error) {
	maxRetries := r.config.MaxRetries
	if maxRetries == 0 {
		maxRetries = defaultMaxRetries
	}
	if maxRetries < 0 {
		maxRetries = 0
	}

	var response *http.Response
	for attempt := 0; ; attempt++ {
		// Terminal if the context is already cancelled (e.g. Ctrl-C during a
		// prior backoff) — bail before spending another attempt.
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		req, err := r.newRequest(ctx, method, url, body)
		if err != nil {
			return nil, fmt.Errorf(errFailedToCreateRequest, err)
		}

		resp, err := r.client.Do(req)
		if err != nil {
			// A cancelled/expired context is terminal — don't waste retries.
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			// Transient network error — retry with backoff (never mid-stream:
			// we only start reading the body after a successful establishment).
			if attempt < maxRetries {
				r.backoff(ctx, attempt, "")
				continue
			}
			return nil, fmt.Errorf(errFailedToMakeRequest, err)
		}

		if shouldRetryStatus(resp.StatusCode) && attempt < maxRetries {
			retryAfter := resp.Header.Get("Retry-After")
			_ = resp.Body.Close()
			r.backoff(ctx, attempt, retryAfter)
			continue
		}

		response = resp
		break
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
		return r.ProcessResponse(response.Body, os.Stdout, url)
	}

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf(errFailedToRead, err)
	}

	return result, nil
}

// shouldRetryStatus reports whether an HTTP status warrants a retry: rate limits
// (429) and transient server errors (5xx).
func shouldRetryStatus(code int) bool {
	return code == http.StatusTooManyRequests || (code >= 500 && code <= 599)
}

// backoff waits before the next retry attempt. It honors a Retry-After header
// (delta-seconds or HTTP-date) when present, otherwise uses exponential backoff
// with full jitter, capped at maxBackoff.
func (r *RestCaller) backoff(ctx context.Context, attempt int, retryAfter string) {
	var d time.Duration

	// Honor a positive Retry-After (capped at maxBackoff). A zero/past value is
	// unusable and falls through to jittered exponential backoff so a broken
	// "Retry-After: 0" can't turn the retry loop into a hot loop.
	if ra, ok := parseRetryAfter(retryAfter); ok && ra > 0 {
		d = ra
		if d > maxBackoff {
			d = maxBackoff
		}
	} else {
		base := r.config.RetryBaseDelayMs
		if base <= 0 {
			base = defaultRetryBaseDelayMs
		}
		// Cap the shift so a large max_retries can't overflow the duration.
		shift := attempt
		if shift > 30 {
			shift = 30
		}
		full := time.Duration(base) * time.Millisecond << uint(shift)
		if full <= 0 || full > maxBackoff {
			full = maxBackoff
		}
		d = rand.N(full) // full jitter: [0, full)
	}

	// Interruptible sleep — a cancelled context (Ctrl-C) returns immediately
	// instead of waiting out a multi-second backoff.
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}
}

func parseRetryAfter(s string) (time.Duration, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(s); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(s); err == nil {
		if d := time.Until(t); d > 0 {
			return d, true
		}
		return 0, true
	}
	return 0, false
}

func (r *RestCaller) newRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	if r.config.APIKey != "" {
		req.Header.Set(r.config.AuthHeader, r.config.AuthTokenPrefix+r.config.APIKey)
	}
	req.Header.Set(internal.HeaderContentTypeKey, internal.HeaderContentTypeValue)
	req.Header.Set(internal.HeaderUserAgentKey, r.config.UserAgent)

	for key, value := range r.config.CustomHeaders {
		req.Header.Set(key, value)
	}

	return req, nil
}
