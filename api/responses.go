package api

type HTTPResponse struct {
	Status  int
	Headers map[string]string
	Body    []byte
}

type ResponsesRequest struct {
	Model           string    `json:"model"`
	Input           []Message `json:"input"`
	MaxOutputTokens int       `json:"max_output_tokens"`
	Reasoning       Reasoning `json:"reasoning"`
	Stream          bool      `json:"stream"`
	Temperature     float64   `json:"temperature,omitempty"`
	TopP            float64   `json:"top_p,omitempty"`
}

type Reasoning struct {
	Effort string `json:"effort"`
}

type ResponsesResponse struct {
	ID                 string   `json:"id"`
	Object             string   `json:"object"`
	CreatedAt          int      `json:"created_at"`
	Status             string   `json:"status"`
	Error              any      `json:"error"`
	IncompleteDetails  any      `json:"incomplete_details"`
	Instructions       any      `json:"instructions"`
	MaxOutputTokens    int      `json:"max_output_tokens"`
	Model              string   `json:"model"`
	Output             []Output `json:"output"`
	ParallelToolCalls  bool     `json:"parallel_tool_calls"`
	PreviousResponseID any      `json:"previous_response_id"`
	Reasoning          struct {
		Effort          string `json:"effort"`
		GenerateSummary any    `json:"generate_summary"`
	} `json:"reasoning"`
	Store       bool    `json:"store"`
	Temperature float64 `json:"temperature"`
	Text        struct {
		Format struct {
			Type string `json:"type"`
		} `json:"format"`
	} `json:"text"`
	ToolChoice string     `json:"tool_choice"`
	Tools      []any      `json:"tools"`
	TopP       float64    `json:"top_p"`
	Truncation string     `json:"truncation"`
	Usage      TokenUsage `json:"usage"`
	User       any        `json:"user"`
	Metadata   struct {
	} `json:"metadata"`
}

type TokenUsage struct {
	InputTokens        int `json:"input_tokens"`
	InputTokensDetails struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"input_tokens_details"`
	OutputTokens        int `json:"output_tokens"`
	OutputTokensDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
	TotalTokens int `json:"total_tokens"`
}

type Output struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Summary []any     `json:"summary,omitempty"`
	Status  string    `json:"status,omitempty"`
	Content []Content `json:"content,omitempty"`
	Role    string    `json:"role,omitempty"`
}

type Content struct {
	Type        string `json:"type"`
	Annotations []any  `json:"annotations"`
	Text        string `json:"text"`
}
