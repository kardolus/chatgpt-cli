package api

import "encoding/json"

// Float64 is a custom type that wraps float64 and implements a custom YAML marshaller.
type Float64 float64

// MarshalJSON omits the field if the value is 0.0.
func (f Float64) MarshalJSON() ([]byte, error) {
	if f == 0.0 {
		return []byte("null"), nil // Returning null to omit the field
	}
	return json.Marshal(float64(f))
}

type CompletionsRequest struct {
	Model            string          `json:"model"`
	Temperature      float64         `json:"temperature,omitempty"`
	TopP             float64         `json:"top_p,omitempty"`
	FrequencyPenalty float64         `json:"frequency_penalty,omitempty"`
	MaxTokens        int             `json:"max_completion_tokens"`
	PresencePenalty  float64         `json:"presence_penalty,omitempty"`
	Messages         []Message       `json:"messages"`
	Stream           bool            `json:"stream"`
	Seed             int             `json:"seed,omitempty"`
	Tools            []FunctionTool  `json:"tools,omitempty"`
	ToolChoice       interface{}     `json:"tool_choice,omitempty"`
	ResponseFormat   *ResponseFormat `json:"response_format,omitempty"`
}

type Message struct {
	Role    string      `json:"role"`
	Name    string      `json:"name,omitempty"`
	Content interface{} `json:"content"`
	// ToolCalls is set on an assistant message when the model wants to invoke
	// functions; ToolCallID links a role:"tool" result back to its call.
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// FunctionTool advertises a callable function to the model (the OpenAI
// tools=[{type:"function", function:{...}}] shape). Distinct from the
// web-search Tool used by the Responses API.
type FunctionTool struct {
	Type     string      `json:"type"` // always "function"
	Function FunctionDef `json:"function"`
}

type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"` // JSON Schema
}

// ToolCall is a function invocation the model emitted in its response.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON-encoded arguments
}

// ResponseFormat drives JSON mode / structured output. Type is typically
// "json_object" or "json_schema"; JSONSchema carries the schema for the latter.
type ResponseFormat struct {
	Type       string          `json:"type"`
	JSONSchema json.RawMessage `json:"json_schema,omitempty"`
}

type AudioContent struct {
	Type       string     `json:"type"`
	Text       string     `json:"text,omitempty"`
	InputAudio InputAudio `json:"input_audio,omitempty"`
}

type InputAudio struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

type ImageContent struct {
	Type     string `json:"type"`
	ImageURL struct {
		URL string `json:"url"`
	} `json:"image_url"`
}

type CompletionsResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int      `json:"created"`
	Model   string   `json:"model"`
	Usage   Usage    `json:"usage"`
	Choices []Choice `json:"choices"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
	Index        int     `json:"index"`
}

type Data struct {
	ID               string  `json:"id"`
	Object           string  `json:"object"`
	Created          int     `json:"created"`
	Model            string  `json:"model"`
	Temperature      float64 `json:"temperature"`
	TopP             float64 `json:"top_p"`
	FrequencyPenalty float64 `json:"frequency_penalty"`
	PresencePenalty  float64 `json:"presence_penalty"`
	Choices          []struct {
		Delta        map[string]interface{} `json:"delta"`
		Index        int                    `json:"index"`
		FinishReason string                 `json:"finish_reason"`
	} `json:"choices"`
}

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}
