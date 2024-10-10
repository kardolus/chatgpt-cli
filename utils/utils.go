package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ConfigHomeEnv    = "OPENAI_CONFIG_HOME"
	DataHomeEnv      = "OPENAI_DATA_HOME"
	DefaultConfigDir = ".chatgpt-cli"
	DefaultDataDir   = "history"
)

func FormatPrompt(str string, counter, usage int, now time.Time) string {
	variables := map[string]string{
		"%datetime": now.Format("2006-01-02 15:04:05"),
		"%date":     now.Format("2006-01-02"),
		"%time":     now.Format("15:04:05"),
		"%counter":  fmt.Sprintf("%d", counter),
		"%usage":    fmt.Sprintf("%d", usage),
	}

	// Replace placeholders in the order of longest to shortest
	for _, key := range []string{"%datetime", "%date", "%time", "%counter", "%usage"} {
		str = strings.ReplaceAll(str, key, variables[key])
	}

	// Ensure the last character is a space
	if str != "" && !strings.HasSuffix(str, " ") {
		str += " "
	}

	str = strings.ReplaceAll(str, "\\n", "\n")

	return str
}

func GetConfigHome() (string, error) {
	var result string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	result = filepath.Join(homeDir, DefaultConfigDir)

	if tmp := os.Getenv(ConfigHomeEnv); tmp != "" {
		result = tmp
	}

	return result, nil
}

func GetDataHome() (string, error) {
	var result string

	configHome, err := GetConfigHome()
	if err != nil {
		return "", err
	}

	result = filepath.Join(configHome, DefaultDataDir)

	if tmp := os.Getenv(DataHomeEnv); tmp != "" {
		result = tmp
	}

	return result, nil
}

func FileToString(fileName string) (string, error) {
	bytes, err := os.ReadFile(fileName)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
