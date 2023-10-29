package types

type CompletionsRequest struct {
	Model            string    `json:"model"`
	Temperature      float64   `json:"temperature"`
	TopP             float64   `json:"top_p"`
	FrequencyPenalty float64   `json:"frequency_penalty"`
	PresencePenalty  float64   `json:"presence_penalty"`
	Messages         []Message `json:"messages"`
	Stream           bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionsResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []Choice `json:"choices"`
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
		Delta        map[string]string `json:"delta"`
		Index        int               `json:"index"`
		FinishReason string            `json:"finish_reason"`
	} `json:"choices"`
}

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}
