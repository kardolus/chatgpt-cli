package configmanager

import (
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/types"
)

type ConfigManager struct {
	configStore config.ConfigStore
}

func New(cs config.ConfigStore) *ConfigManager {
	return &ConfigManager{configStore: cs}
}

func (c *ConfigManager) ReadModel() (string, error) {
	conf, err := c.configStore.Read()
	if err != nil {
		return "", err
	}

	return conf.Model, nil
}

func (c *ConfigManager) WriteModel(model string) error {
	return c.configStore.Write(types.Config{Model: model})
}
