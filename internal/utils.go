package internal

import (
	"github.com/google/uuid"
	"os"
	"path/filepath"
)

const (
	ConfigHomeEnv     = "OPENAI_CONFIG_HOME"
	DataHomeEnv       = "OPENAI_DATA_HOME"
	CacheHomeEnv      = "OPENAI_CACHE_HOME"
	DefaultConfigDir  = ".chatgpt-cli"
	DefaultDataDir    = "history"
	DefaultCacheDir   = "cache"
	SlugPostfixLength = 4
)

func GenerateUniqueSlug(prefix string) string {
	guid := uuid.New()
	return prefix + guid.String()[:SlugPostfixLength]
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

func GetCacheHome() (string, error) {
	var result string

	configHome, err := GetConfigHome()
	if err != nil {
		return "", err
	}

	result = filepath.Join(configHome, DefaultCacheDir)

	if tmp := os.Getenv(CacheHomeEnv); tmp != "" {
		result = tmp
	}

	return result, nil
}
