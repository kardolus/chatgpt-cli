package types

type Config struct {
	Model           string `yaml:"model"`
	MaxTokens       int    `yaml:"max_tokens"`
	URL             string `yaml:"url"`
	CompletionsPath string `yaml:"completions_path"`
	ModelsPath      string `yaml:"models_path"`
}
