//go:build windows

package config

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

type fileLock struct {
	path string
	f    *os.File
}

func newFileLock(targetPath string) *fileLock {
	return &fileLock{path: targetPath + ".lock"}
}

// Minimal OVERLAPPED compatible with Windows API.
type overlapped struct {
	Internal     uintptr
	InternalHigh uintptr
	Offset       uint32
	OffsetHigh   uint32
	HEvent       syscall.Handle
}

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx    = kernel32.NewProc("LockFileEx")
	procUnlockFileEx  = kernel32.NewProc("UnlockFileEx")
	lockfileExclusive = uintptr(0x00000002) // LOCKFILE_EXCLUSIVE_LOCK
)

func (l *fileLock) Lock() error {
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}
	l.f = f

	h := syscall.Handle(f.Fd())
	var ov overlapped

	// Lock 1 byte at offset 0. This is a common pattern for advisory locks.
	r1, _, e1 := procLockFileEx.Call(
		uintptr(h),
		lockfileExclusive, // exclusive lock
		0,                 // reserved
		1,                 // nNumberOfBytesToLockLow
		0,                 // nNumberOfBytesToLockHigh
		uintptr(unsafe.Pointer(&ov)),
	)
	if r1 == 0 {
		_ = f.Close()
		l.f = nil
		// e1 is syscall.Errno
		if e1 != nil && e1 != syscall.Errno(0) {
			return fmt.Errorf("LockFileEx: %w", e1)
		}
		return fmt.Errorf("LockFileEx: failed")
	}

	return nil
}

func (l *fileLock) Unlock() error {
	if l.f == nil {
		return nil
	}

	h := syscall.Handle(l.f.Fd())
	var ov overlapped

	r1, _, e1 := procUnlockFileEx.Call(
		uintptr(h),
		0, // reserved
		1, // nNumberOfBytesToUnlockLow
		0, // nNumberOfBytesToUnlockHigh
		uintptr(unsafe.Pointer(&ov)),
	)

	// Always close even if unlock fails.
	errClose := l.f.Close()
	l.f = nil

	if r1 == 0 {
		if e1 != nil && e1 != syscall.Errno(0) {
			return fmt.Errorf("UnlockFileEx: %w", e1)
		}
		return fmt.Errorf("UnlockFileEx: failed")
	}
	return errClose
}
