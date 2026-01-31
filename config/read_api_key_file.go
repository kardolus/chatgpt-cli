package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxAPIKeyFileBytes int64 = 10 * 1024 // 10KB

func ReadAPIKeyFile(path string) (string, error) {
	clean := filepath.Clean(path)

	f, err := os.Open(clean)
	if err != nil {
		return "", fmt.Errorf("failed to open api key file: %w", err)
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat api key file: %w", err)
	}

	if !st.Mode().IsRegular() {
		return "", errors.New("api key file must be a regular file")
	}

	if st.Size() > maxAPIKeyFileBytes {
		return "", fmt.Errorf("api key file too large (max %d bytes)", maxAPIKeyFileBytes)
	}

	r := io.LimitReader(f, maxAPIKeyFileBytes+1)
	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read api key file: %w", err)
	}
	if int64(len(b)) > maxAPIKeyFileBytes {
		return "", fmt.Errorf("api key file too large (max %d bytes)", maxAPIKeyFileBytes)
	}

	key := strings.TrimSpace(string(b))
	if key == "" {
		return "", errors.New("api key file is empty")
	}
	return key, nil
}

func MaxAPIKeyFileBytesForTest() int64 { return maxAPIKeyFileBytes }
