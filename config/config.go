package config

import (
	"github.com/kardolus/chatgpt-cli/types"
	"github.com/kardolus/chatgpt-cli/utils"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"runtime"
)

type ConfigStore interface {
	Read() (types.Config, error)
	ReadDefaults() (types.Config, error)
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

func (f *FileIO) ReadDefaults() (types.Config, error) {
	_, thisFile, _, _ := runtime.Caller(0)

	configPath := filepath.Join(thisFile, "..", "..", "resources", "config.yaml")

	return parseFile(configPath)
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
