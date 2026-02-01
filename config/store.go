package config

import (
	"fmt"
	"github.com/kardolus/chatgpt-cli/internal"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

const (
	openAIName                 = "openai"
	openAIModel                = "gpt-4o"
	openAIMaxTokens            = 4096
	openAIContextWindow        = 8192
	openAIURL                  = "https://api.openai.com"
	openAICompletionsPath      = "/v1/chat/completions"
	openAIResponsesPath        = "/v1/responses"
	openAITranscriptionsPath   = "/v1/audio/transcriptions"
	openAISpeechPath           = "/v1/audio/speech"
	openAIImageGenerationsPath = "/v1/images/generations"
	openAIImageEditsPath       = "/v1/images/edits"
	openAIModelsPath           = "/v1/models"
	openAIAuthHeader           = "Authorization"
	openAIAuthTokenPrefix      = "Bearer "
	openAIRole                 = "You are a helpful assistant."
	openAIThread               = "default"
	openAITemperature          = 1.0
	openAITopP                 = 1.0
	openAIFrequencyPenalty     = 0.0
	openAIPresencePenalty      = 0.0
	openAICommandPrompt        = "[%datetime] [Q%counter]"
	openAIEffort               = "low"
	openAIVoice                = "voice"
)

type Store interface {
	Delete(string) error
	List() ([]string, error)
	Read() (Config, error)
	ReadDefaults() Config
	Write(Config) error
}

// FileNotFoundError is a custom error type for non-existent files
type FileNotFoundError struct {
	Path string
}

func (e *FileNotFoundError) Error() string {
	return fmt.Sprintf("no threads matched the pattern %s", e.Path)
}

// Ensure FileIO implements ConfigStore interface
var _ Store = &FileIO{}

type FileIO struct {
	configFilePath  string
	historyFilePath string
}

func NewStore() *FileIO {
	configPath, _ := getPath()
	historyPath, _ := internal.GetDataHome()

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

func (f *FileIO) Delete(pattern string) error {
	if !strings.HasSuffix(pattern, "*") && !strings.HasSuffix(pattern, ".json") {
		pattern += ".json"
	} else if strings.HasSuffix(pattern, "*") {
		pattern += ".json"
	}

	fullPattern := filepath.Join(f.historyFilePath, pattern)

	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return fmt.Errorf("failed to process pattern %s: %w", fullPattern, err)
	}

	if len(matches) == 0 {
		return &FileNotFoundError{Path: fullPattern}
	}

	for _, path := range matches {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to delete file %s: %w", path, err)
		}
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

func (f *FileIO) Read() (Config, error) {
	result, err := parseFile(f.configFilePath)
	if err != nil {
		return Config{}, err
	}

	return migrate(result), nil
}

func (f *FileIO) ReadDefaults() Config {
	return Config{
		Name:                 openAIName,
		Model:                openAIModel,
		Role:                 openAIRole,
		MaxTokens:            openAIMaxTokens,
		ContextWindow:        openAIContextWindow,
		URL:                  openAIURL,
		CompletionsPath:      openAICompletionsPath,
		ResponsesPath:        openAIResponsesPath,
		TranscriptionsPath:   openAITranscriptionsPath,
		SpeechPath:           openAISpeechPath,
		ImageGenerationsPath: openAIImageGenerationsPath,
		ImageEditsPath:       openAIImageEditsPath,
		ModelsPath:           openAIModelsPath,
		AuthHeader:           openAIAuthHeader,
		AuthTokenPrefix:      openAIAuthTokenPrefix,
		Thread:               openAIThread,
		Temperature:          openAITemperature,
		TopP:                 openAITopP,
		FrequencyPenalty:     openAIFrequencyPenalty,
		PresencePenalty:      openAIPresencePenalty,
		CommandPrompt:        openAICommandPrompt,
		Effort:               openAIEffort,
		Voice:                openAIVoice,
	}
}

func (f *FileIO) Write(config Config) error {
	lock := newFileLock(f.configFilePath)
	if err := lock.Lock(); err != nil {
		return err
	}
	defer func() { _ = lock.Unlock() }()

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

	perm := os.FileMode(0o600)
	if st, err := os.Stat(f.configFilePath); err == nil {
		perm = st.Mode() & 0o777
	}

	return atomicWriteFile(f.configFilePath, modifiedContent, perm)
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

func atomicWriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(filename)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()

	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	// Best-effort: make sure permissions are correct before rename.
	if err := tmp.Chmod(perm); err != nil {
		return fmt.Errorf("chmod temp: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("fsync temp: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}

	if runtime.GOOS == "windows" {
		_ = os.Remove(filename)
	}

	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("rename temp -> target: %w", err)
	}

	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}

	return nil
}

func getPath() (string, error) {
	homeDir, err := internal.GetConfigHome()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, "config.yaml"), nil
}

func migrate(config Config) Config {
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

func parseFile(fileName string) (Config, error) {
	var result Config

	buf, err := os.ReadFile(fileName)
	if err != nil {
		return Config{}, err
	}

	if err := yaml.Unmarshal(buf, &result); err != nil {
		return Config{}, err
	}

	return result, nil
}

// updateNodeFromConfig updates the specified yaml.Node with values from the Config struct.
// It uses reflection to match struct fields with YAML tags, updating the node accordingly.
func updateNodeFromConfig(node *yaml.Node, config Config) {
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
