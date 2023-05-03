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
}

func New() *FileIO {
	return &FileIO{}
}

func (f *FileIO) Delete() error {
	historyFilePath, err := getPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(historyFilePath); err == nil {
		return os.Remove(historyFilePath)
	}

	return nil
}

func (f *FileIO) Read() ([]types.Message, error) {
	historyFilePath, err := getPath()
	if err != nil {
		return nil, err
	}

	return parseFile(historyFilePath)
}

func (f *FileIO) Write(messages []types.Message) error {
	historyFilePath, err := getPath()
	if err != nil {
		return err
	}

	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(historyFilePath, data, 0644)
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
