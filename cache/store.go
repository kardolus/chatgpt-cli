package cache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

//go:generate mockgen -destination=storemocks_test.go -package=cache_test github.com/kardolus/chatgpt-cli/cache Store
type Store interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
}

func NewFileStore(baseDir string) *FileStore {
	return &FileStore{
		baseDir: baseDir,
	}
}

// Ensure FileStore implements the Store interface
var _ Store = &FileStore{}

type FileStore struct {
	baseDir string
}

func (f *FileStore) Get(key string) ([]byte, error) {
	path := f.pathForKey(key)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (f *FileStore) Set(key string, value []byte) error {
	if err := f.ensureBaseDir(); err != nil {
		return err
	}

	dst := f.pathForKey(key)

	// Write to a temp file in the same directory so rename is atomic.
	tmp, err := os.CreateTemp(f.baseDir, fmt.Sprintf(".%s.*.tmp", key))
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	// Clean up temp file on failure.
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(value); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	// Atomic replace on POSIX; on Windows, Rename may fail if dst exists.
	// Best effort: remove dst first if needed.
	if err := os.Rename(tmpName, dst); err != nil {
		if errors.Is(err, os.ErrExist) || errors.Is(err, os.ErrPermission) {
			_ = os.Remove(dst)
			return os.Rename(tmpName, dst)
		}
		return err
	}

	return nil
}

func (f *FileStore) Delete(key string) error {
	path := f.pathForKey(key)
	err := os.Remove(path)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (f *FileStore) ensureBaseDir() error {
	// 0700: single-user CLI cache
	return os.MkdirAll(f.baseDir, 0o700)
}

func (f *FileStore) pathForKey(key string) string {
	// key should already be a safe filename (yours is sha256 hex), so no sanitization needed.
	return filepath.Join(f.baseDir, key+".json")
}
