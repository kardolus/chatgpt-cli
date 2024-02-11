package utils

import (
	"os"
	"path/filepath"
)

const (
	cliDirName     = ".chatgpt-cli"
	historyDirName = "history"
)

func GetChatGPTDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, cliDirName), nil
}

func GetHistoryDir() (string, error) {
	homeDir, err := GetChatGPTDirectory()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, historyDirName), nil
}
