package types

type Config struct {
	Name            string `yaml:"name"`
	APIKey          string `yaml:"api_key"`
	Model           string `yaml:"model"`
	MaxTokens       int    `yaml:"max_tokens"`
	Role            string `yaml:"role"`
	Thread          string `yaml:"thread"`
	OmitHistory     bool   `yaml:"omit_history"`
	URL             string `yaml:"url"`
	CompletionsPath string `yaml:"completions_path"`
	ModelsPath      string `yaml:"models_path"`
}
