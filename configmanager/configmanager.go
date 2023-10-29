package configmanager

import (
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/types"
	"gopkg.in/yaml.v3"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type ConfigManager struct {
	configStore config.ConfigStore
	Config      types.Config
}

func New(cs config.ConfigStore) *ConfigManager {
	configuration := cs.ReadDefaults()

	userConfig, err := cs.Read()
	if err == nil {
		configuration = replaceByConfigFile(configuration, userConfig)
	}

	return &ConfigManager{configStore: cs, Config: configuration}
}

func (c *ConfigManager) WithEnvironment() *ConfigManager {
	c.Config = replaceByEnvironment(c.Config)
	return c
}

func (c *ConfigManager) APIKeyEnvVarName() string {
	return strings.ToUpper(c.Config.Name) + "_" + "API_KEY"
}

func (c *ConfigManager) ShowConfig() (string, error) {
	data, err := yaml.Marshal(c.Config)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (c *ConfigManager) WriteMaxTokens(tokens int) error {
	c.Config.MaxTokens = tokens

	return c.configStore.Write(c.Config)
}

func (c *ConfigManager) WriteModel(model string) error {
	c.Config.Model = model

	return c.configStore.Write(c.Config)
}

func replaceByConfigFile(defaultConfig, userConfig types.Config) types.Config {
	t := reflect.TypeOf(defaultConfig)
	vDefault := reflect.ValueOf(&defaultConfig).Elem()
	vUser := reflect.ValueOf(userConfig)

	for i := 0; i < t.NumField(); i++ {
		defaultField := vDefault.Field(i)
		userField := vUser.Field(i)

		switch defaultField.Kind() {
		case reflect.String:
			if userStr := userField.String(); userStr != "" {
				defaultField.SetString(userStr)
			}
		case reflect.Int:
			if userInt := int(userField.Int()); userInt != 0 {
				defaultField.SetInt(int64(userInt))
			}
		case reflect.Bool:
			if userBool := userField.Bool(); &userBool != nil {
				defaultField.SetBool(userBool)
			}
		case reflect.Float64:
			if userFloat := userField.Float(); userFloat != 0.0 {
				defaultField.SetFloat(userFloat)
			}
		}
	}

	return defaultConfig
}

func replaceByEnvironment(configuration types.Config) types.Config {
	t := reflect.TypeOf(configuration)
	v := reflect.ValueOf(&configuration).Elem()

	prefix := strings.ToUpper(configuration.Name) + "_"
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("yaml")
		if tag == "name" {
			continue
		}

		if value := os.Getenv(prefix + strings.ToUpper(tag)); value != "" {
			field := v.Field(i)

			switch field.Kind() {
			case reflect.String:
				field.SetString(value)
			case reflect.Int:
				intValue, _ := strconv.Atoi(value)
				field.SetInt(int64(intValue))
			case reflect.Bool:
				boolValue, _ := strconv.ParseBool(value)
				field.SetBool(boolValue)
			case reflect.Float64:
				floatValue, _ := strconv.ParseFloat(value, 64)
				field.SetFloat(floatValue)
			}
		}
	}

	return configuration
}
