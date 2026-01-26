//go:build !windows

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type fileLock struct {
	path string
	f    *os.File
}

func newFileLock(targetPath string) *fileLock {
	lockPath := targetPath + ".lock"
	_ = filepath.Dir(lockPath)
	return &fileLock{path: lockPath}
}

func (l *fileLock) Lock() error {
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}
	l.f = f

	// Exclusive lock (blocks).
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		l.f = nil
		return fmt.Errorf("flock: %w", err)
	}
	return nil
}

func (l *fileLock) Unlock() error {
	if l.f == nil {
		return nil
	}
	// Best-effort unlock + close.
	_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	err := l.f.Close()
	l.f = nil
	return err
}
