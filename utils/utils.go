package utils

import (
	"os"
	"path/filepath"
)

func GetChatGPTDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".chatgpt-cli"), nil
}
