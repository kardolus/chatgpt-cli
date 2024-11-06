package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type Manager struct {
	configStore ConfigStore
	Config      Config
}

func NewManager(cs ConfigStore) *Manager {
	configuration := cs.ReadDefaults()

	userConfig, err := cs.Read()
	if err == nil {
		configuration = replaceByConfigFile(configuration, userConfig)
	}

	return &Manager{configStore: cs, Config: configuration}
}

func (c *Manager) WithEnvironment() *Manager {
	c.Config = replaceByEnvironment(c.Config)
	return c
}

func (c *Manager) APIKeyEnvVarName() string {
	return strings.ToUpper(c.Config.Name) + "_" + "API_KEY"
}

// DeleteThread removes the specified thread from the configuration store.
// This operation is idempotent; non-existent threads do not cause errors.
func (c *Manager) DeleteThread(thread string) error {
	return c.configStore.Delete(thread)
}

// ListThreads retrieves a list of all threads stored in the configuration.
// It marks the current thread with an asterisk (*) and returns the list sorted alphabetically.
// If an error occurs while retrieving the threads from the config store, it returns the error.
func (c *Manager) ListThreads() ([]string, error) {
	var result []string

	threads, err := c.configStore.List()
	if err != nil {
		return nil, err
	}

	for _, thread := range threads {
		thread = strings.ReplaceAll(thread, ".json", "")
		if thread != c.Config.Thread {
			result = append(result, fmt.Sprintf("- %s", thread))
			continue
		}
		result = append(result, fmt.Sprintf("* %s (current)", thread))
	}

	return result, nil
}

// ShowConfig serializes the current configuration to a YAML string.
// It returns the serialized string or an error if the serialization fails.
func (c *Manager) ShowConfig() (string, error) {
	data, err := yaml.Marshal(c.Config)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func replaceByConfigFile(defaultConfig, userConfig Config) Config {
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
			defaultField.SetBool(userField.Bool())
		case reflect.Float64:
			if userFloat := userField.Float(); userFloat != 0.0 {
				defaultField.SetFloat(userFloat)
			}
		}
	}

	return defaultConfig
}

func replaceByEnvironment(configuration Config) Config {
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
