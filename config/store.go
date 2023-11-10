package config

import (
	"github.com/kardolus/chatgpt-cli/types"
	"github.com/kardolus/chatgpt-cli/utils"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

const (
	openAIName             = "openai"
	openAIModel            = "gpt-3.5-turbo"
	openAIModelMaxTokens   = 4096
	openAIURL              = "https://api.openai.com"
	openAICompletionsPath  = "/v1/chat/completions"
	openAIModelsPath       = "/v1/models"
	openAIAuthHeader       = "Authorization"
	openAIAuthTokenPrefix  = "Bearer "
	openAIRole             = "You are a helpful assistant."
	openAIThread           = "default"
	openAITemperature      = 1.0
	openAITopP             = 1.0
	openAIFrequencyPenalty = 0.0
	openAIPresencePenalty  = 0.0
)

type ConfigStore interface {
	Read() (types.Config, error)
	ReadDefaults() types.Config
	Write(types.Config) error
}

// Ensure FileIO implements ConfigStore interface
var _ ConfigStore = &FileIO{}

type FileIO struct {
	configFilePath string
}

func New() *FileIO {
	path, _ := getPath()
	return &FileIO{
		configFilePath: path,
	}
}

func (f *FileIO) WithFilePath(configFilePath string) *FileIO {
	f.configFilePath = configFilePath
	return f
}

func (f *FileIO) Read() (types.Config, error) {
	return parseFile(f.configFilePath)
}

func (f *FileIO) ReadDefaults() types.Config {
	return types.Config{
		Name:             openAIName,
		Model:            openAIModel,
		Role:             openAIRole,
		MaxTokens:        openAIModelMaxTokens,
		URL:              openAIURL,
		CompletionsPath:  openAICompletionsPath,
		ModelsPath:       openAIModelsPath,
		AuthHeader:       openAIAuthHeader,
		AuthTokenPrefix:  openAIAuthTokenPrefix,
		Thread:           openAIThread,
		Temperature:      openAITemperature,
		TopP:             openAITopP,
		FrequencyPenalty: openAIFrequencyPenalty,
		PresencePenalty:  openAIPresencePenalty,
	}
}

func (f *FileIO) Write(config types.Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(f.configFilePath, data, 0644)
}

func getPath() (string, error) {
	homeDir, err := utils.GetChatGPTDirectory()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, "config.yaml"), nil
}

func parseFile(fileName string) (types.Config, error) {
	var result types.Config

	buf, err := os.ReadFile(fileName)
	if err != nil {
		return types.Config{}, err
	}

	if err := yaml.Unmarshal(buf, &result); err != nil {
		return types.Config{}, err
	}

	return result, nil
}
