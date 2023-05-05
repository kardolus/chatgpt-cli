package history

import (
	"encoding/json"
	"github.com/kardolus/chatgpt-cli/types"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Store interface {
	Delete() error
	Read() ([]types.Message, error)
	Write([]types.Message) error
}

// Ensure RestCaller implements Caller interface
var _ Store = &FileIO{}

type FileIO struct {
	historyFilePath string
}

func New() *FileIO {
	path, _ := getPath()
	return &FileIO{
		historyFilePath: path,
	}
}

func (f *FileIO) WithHistory(historyFilePath string) *FileIO {
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

	return ioutil.WriteFile(f.historyFilePath, data, 0644)
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

	buf, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(buf, &result); err != nil {
		return nil, err
	}

	return result, nil
}
