package configmanager

import (
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/types"
	"gopkg.in/yaml.v3"
)

type ConfigManager struct {
	configStore config.ConfigStore
	Config      types.Config
}

func New(cs config.ConfigStore) *ConfigManager {
	c := cs.ReadDefaults()

	configured, err := cs.Read()
	if err == nil {
		if configured.Model != "" {
			c.Model = configured.Model
		}
		if configured.MaxTokens != 0 {
			c.MaxTokens = configured.MaxTokens
		}
		if configured.URL != "" {
			c.URL = configured.URL
		}
		if configured.CompletionsPath != "" {
			c.CompletionsPath = configured.CompletionsPath
		}
		if configured.ModelsPath != "" {
			c.ModelsPath = configured.ModelsPath
		}
	}

	return &ConfigManager{configStore: cs, Config: c}
}

func (c *ConfigManager) ShowConfig() (string, error) {
	data, err := yaml.Marshal(c.Config)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (c *ConfigManager) WriteModel(model string) error {
	c.Config.Model = model

	return c.configStore.Write(c.Config)
}
