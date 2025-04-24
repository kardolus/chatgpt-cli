package config

type Config struct {
	Name                string  `yaml:"name"`
	APIKey              string  `yaml:"api_key"`
	Model               string  `yaml:"model"`
	MaxTokens           int     `yaml:"max_tokens"`
	ContextWindow       int     `yaml:"context_window"`
	Role                string  `yaml:"role"`
	Temperature         float64 `yaml:"temperature"`
	TopP                float64 `yaml:"top_p"`
	FrequencyPenalty    float64 `yaml:"frequency_penalty"`
	PresencePenalty     float64 `yaml:"presence_penalty"`
	Thread              string  `yaml:"thread"`
	OmitHistory         bool    `yaml:"omit_history"`
	URL                 string  `yaml:"url"`
	CompletionsPath     string  `yaml:"completions_path"`
	ModelsPath          string  `yaml:"models_path"`
	ResponsesPath       string  `yaml:"responses_path"`
	SpeechPath          string  `yaml:"speech_path"`
	TranscriptionsPath  string  `yaml:"transcriptions_path"`
	AuthHeader          string  `yaml:"auth_header"`
	AuthTokenPrefix     string  `yaml:"auth_token_prefix"`
	CommandPrompt       string  `yaml:"command_prompt"`
	CommandPromptColor  string  `yaml:"command_prompt_color"`
	OutputPrompt        string  `yaml:"output_prompt"`
	OutputPromptColor   string  `yaml:"output_prompt_color"`
	AutoCreateNewThread bool    `yaml:"auto_create_new_thread"`
	TrackTokenUsage     bool    `yaml:"track_token_usage"`
	SkipTLSVerify       bool    `yaml:"skip_tls_verify"`
	Multiline           bool    `yaml:"multiline"`
	Seed                int     `yaml:"seed"`
	Effort              string  `yaml:"effort"`
	Voice               string  `yaml:"voice"`
	ApifyAPIKey         string  `yaml:"apify_api_key"`
}
