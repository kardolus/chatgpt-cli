package types

type Config struct {
	Name             string   `yaml:"name"`
	Provider         Provider `yaml:"provider"`
	APIKey           string   `yaml:"api_key"`
	Model            string   `yaml:"model"`
	MaxTokens        int      `yaml:"max_tokens"`
	ContextWindow    int      `yaml:"context_window"`
	Role             string   `yaml:"role"`
	Temperature      float64  `yaml:"temperature"`
	TopP             float64  `yaml:"top_p"`
	FrequencyPenalty float64  `yaml:"frequency_penalty"`
	PresencePenalty  float64  `yaml:"presence_penalty"`
	Thread           string   `yaml:"thread"`
	OmitHistory      bool     `yaml:"omit_history"`
	URL              string   `yaml:"url"`
	CompletionsPath  string   `yaml:"completions_path"`
	ModelsPath       string   `yaml:"models_path"`
	AuthHeader       string   `yaml:"auth_header"`
	AuthTokenPrefix  string   `yaml:"auth_token_prefix"`
}

type Provider string

const (
	OpenAI Provider = "OpenAI"
	Cohere          = "Cohere"
)
