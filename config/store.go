package config

import (
	"github.com/kardolus/chatgpt-cli/types"
	"github.com/kardolus/chatgpt-cli/utils"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
)

const (
	openAIName             = "openai"
	openAIModel            = "gpt-3.5-turbo"
	openAIMaxTokens        = 4096
	openAIContextWindow    = 8192
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
	openAICommandPrompt    = "[%datetime] [Q%counter]"
)

type ConfigStore interface {
	Delete(string) error
	List() ([]string, error)
	Read() (types.Config, error)
	ReadDefaults() types.Config
	Write(types.Config) error
}

// Ensure FileIO implements ConfigStore interface
var _ ConfigStore = &FileIO{}

type FileIO struct {
	configFilePath  string
	historyFilePath string
}

func New() *FileIO {
	configPath, _ := getPath()
	historyPath, _ := utils.GetHistoryDir()

	return &FileIO{
		configFilePath:  configPath,
		historyFilePath: historyPath,
	}
}

func (f *FileIO) WithConfigPath(configFilePath string) *FileIO {
	f.configFilePath = configFilePath
	return f
}

func (f *FileIO) WithHistoryPath(historyPath string) *FileIO {
	f.historyFilePath = historyPath
	return f
}

func (f *FileIO) Delete(name string) error {
	path := filepath.Join(f.historyFilePath, name+".json")

	if _, err := os.Stat(path); err == nil {
		return os.Remove(path)
	}
	return nil
}

func (f *FileIO) List() ([]string, error) {
	var result []string

	files, err := os.ReadDir(f.historyFilePath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		result = append(result, file.Name())
	}

	return result, nil
}

func (f *FileIO) Read() (types.Config, error) {
	result, err := parseFile(f.configFilePath)
	if err != nil {
		return types.Config{}, err
	}

	return migrate(result), nil
}

func (f *FileIO) ReadDefaults() types.Config {
	return types.Config{
		Name:             openAIName,
		Model:            openAIModel,
		Role:             openAIRole,
		MaxTokens:        openAIMaxTokens,
		ContextWindow:    openAIContextWindow,
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
		CommandPrompt:    openAICommandPrompt,
	}
}

func (f *FileIO) Write(config types.Config) error {
	rootNode, err := f.readNode()

	// If readNode returns an error or there was a problem reading the rootNode, initialize a new rootNode.
	if err != nil || rootNode.Kind == 0 {
		rootNode = yaml.Node{Kind: yaml.DocumentNode}
		rootNode.Content = append(rootNode.Content, &yaml.Node{Kind: yaml.MappingNode})
	}

	updateNodeFromConfig(&rootNode, config)

	modifiedContent, err := yaml.Marshal(&rootNode)
	if err != nil {
		return err
	}

	return os.WriteFile(f.configFilePath, modifiedContent, 0644)
}

func (f *FileIO) readNode() (yaml.Node, error) {
	var rootNode yaml.Node

	content, err := os.ReadFile(f.configFilePath)
	if err != nil {
		return rootNode, err
	}

	if err := yaml.Unmarshal(content, &rootNode); err != nil {
		return rootNode, err
	}

	return rootNode, nil
}

func getPath() (string, error) {
	homeDir, err := utils.GetChatGPTDirectory()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, "config.yaml"), nil
}

func migrate(config types.Config) types.Config {
	// the "old" max_tokens became context_window
	if config.ContextWindow == 0 && config.MaxTokens > 0 {
		config.ContextWindow = config.MaxTokens
		// set it to the default in case the value is small
		if config.ContextWindow < openAIContextWindow {
			config.ContextWindow = openAIContextWindow
		}
		config.MaxTokens = openAIMaxTokens
	}
	return config
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

// updateNodeFromConfig updates the specified yaml.Node with values from the Config struct.
// It uses reflection to match struct fields with YAML tags, updating the node accordingly.
func updateNodeFromConfig(node *yaml.Node, config types.Config) {
	t := reflect.TypeOf(config)
	v := reflect.ValueOf(config)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		yamlTag := field.Tag.Get("yaml")

		if yamlTag == "" || yamlTag == "-" {
			continue // Skip fields without yaml tag or marked to be ignored
		}

		// Convert value to string; adjust for different data types as needed
		var strValue string
		switch value.Kind() {
		case reflect.String:
			strValue = value.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			strValue = strconv.FormatInt(value.Int(), 10)
		case reflect.Float32, reflect.Float64:
			strValue = strconv.FormatFloat(value.Float(), 'f', -1, 64)
		case reflect.Bool:
			strValue = strconv.FormatBool(value.Bool())
		default:
			continue // Skip unsupported types for simplicity
		}

		setField(node, yamlTag, strValue)
	}
}

// setField either updates an existing field or adds a new field to the YAML mapping node.
// It assumes the root node is a DocumentNode containing a MappingNode.
func setField(root *yaml.Node, key string, newValue string) {
	found := false

	if root.Kind == yaml.DocumentNode {
		root = root.Content[0] // Move from document node to the actual mapping node.
	}

	if root.Kind != yaml.MappingNode {
		return // If the root is not a mapping node, we can't do anything.
	}

	for i := 0; i < len(root.Content); i += 2 {
		keyNode := root.Content[i]
		if keyNode.Value == key {
			valueNode := root.Content[i+1]
			valueNode.Value = newValue
			found = true
			break
		}
	}

	if !found { // If the key wasn't found, add it.
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, &yaml.Node{Kind: yaml.ScalarNode, Value: newValue})
	}
}
