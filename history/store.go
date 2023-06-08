package history

import (
	"encoding/json"
	"github.com/kardolus/chatgpt-cli/types"
	"os"
	"path/filepath"
)

type HistoryStore interface {
	Delete() error
	Read() ([]types.Message, error)
	Write([]types.Message) error
}

// Ensure FileIO implements the HistoryStore interface
var _ HistoryStore = &FileIO{}

type FileIO struct {
	historyFilePath string
}

func New() *FileIO {
	path, _ := getPath()
	return &FileIO{
		historyFilePath: path,
	}
}

func (f *FileIO) WithFilePath(historyFilePath string) *FileIO {
	f.historyFilePath = historyFilePath
	return f
}

func (f *FileIO) Delete() error {
	if _, err := os.Stat(f.historyFilePath); err == nil {
		return os.Remove(f.historyFilePath)
	}

	return nil
}

func (f *FileIO) Read() ([]types.Message, error) {
	return parseFile(f.historyFilePath)
}

func (f *FileIO) Write(messages []types.Message) error {
	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	return os.WriteFile(f.historyFilePath, data, 0644)
}

func getPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".chatgpt-cli", "history"), nil
}

func parseFile(fileName string) ([]types.Message, error) {
	var result []types.Message

	buf, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(buf, &result); err != nil {
		return nil, err
	}

	return result, nil
}
