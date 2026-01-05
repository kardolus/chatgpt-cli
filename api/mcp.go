package api

import (
	"encoding/json"
	"fmt"
	"strings"
)

type MCPError struct {
	// JSON-RPC spec: code is an integer.
	// We normalize it into a string to keep the rest of your code stable.
	Code    string                 `json:"-"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

func (e *MCPError) UnmarshalJSON(b []byte) error {
	// accept code as number or string
	var aux struct {
		Code    json.RawMessage        `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data,omitempty"`
	}

	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	e.Message = aux.Message
	e.Data = aux.Data

	// Try int first (correct JSON-RPC)
	var n int
	if err := json.Unmarshal(aux.Code, &n); err == nil {
		e.Code = fmt.Sprintf("%d", n)
		return nil
	}

	// Then try string (some servers may do this)
	var s string
	if err := json.Unmarshal(aux.Code, &s); err == nil {
		e.Code = s
		return nil
	}

	// Last resort: keep raw
	e.Code = string(aux.Code)
	return nil
}

func (e *MCPError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code == "" {
		return e.Message
	}
	return fmt.Sprintf("mcp error %s: %s", e.Code, e.Message)
}

type MCPMessage struct {
	// JSON-RPC request/response fields
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      string          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`

	// JSON-RPC response fields
	Result json.RawMessage `json:"result,omitempty"`
	Error  *MCPError       `json:"error,omitempty"`
}

type MCPResponse struct {
	Message MCPMessage        `json:"-"`
	Headers map[string]string `json:"-"`
	Status  int               `json:"-"`
}

func (r MCPResponse) Header(key string) (string, bool) {
	for k, v := range r.Headers {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return "", false
}

type MCPRequest struct {
	Endpoint string
	Headers  map[string]string
	Tool     string
	Params   map[string]interface{}
}

type HTTPTransport struct {
	Headers map[string]string
}
