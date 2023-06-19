package types

type Config struct {
	Name            string `yaml:"name"`
	APIKey          string `yaml:"api_key"`
	Model           string `yaml:"model"`
	MaxTokens       int    `yaml:"max_tokens"`
	URL             string `yaml:"url"`
	CompletionsPath string `yaml:"completions_path"`
	ModelsPath      string `yaml:"models_path"`
	OmitHistory     bool   `yaml:"omit_history"`
}
