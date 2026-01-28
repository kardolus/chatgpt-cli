package core

import (
	"fmt"
	"strings"
	"sync"
)

const defaultTruncationBanner = "\n…(truncated)\n"

type TranscriptBuffer struct {
	mu     sync.Mutex
	max    int
	b      []byte
	banner []byte
}

func NewTranscriptBuffer(maxBytes int) *TranscriptBuffer {
	if maxBytes < 0 {
		maxBytes = 0
	}
	return &TranscriptBuffer{
		max:    maxBytes,
		b:      make([]byte, 0, minInt(maxBytes, 4096)),
		banner: []byte(defaultTruncationBanner),
	}
}

func (t *TranscriptBuffer) AppendString(s string) {
	if t == nil || s == "" {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.max <= 0 {
		t.b = t.b[:0]
		return
	}

	// Normalize: keep entries line-oriented
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}

	t.b = append(t.b, s...)
	t.enforceCapLocked()
}

func (t *TranscriptBuffer) Appendf(format string, args ...any) {
	t.AppendString(fmt.Sprintf(format, args...))
}

func (t *TranscriptBuffer) String() string {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.b)
}

func (t *TranscriptBuffer) Len() int {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.b)
}

func (t *TranscriptBuffer) Reset() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.b = t.b[:0]
}

func (t *TranscriptBuffer) enforceCapLocked() {
	if len(t.b) <= t.max {
		return
	}

	// Keep most recent bytes.
	keep := t.b[len(t.b)-t.max:]
	t.b = append(t.b[:0], keep...)

	// Prepend banner if we can and it's not already there.
	if len(t.banner) < t.max && !hasPrefixBytes(t.b, t.banner) {
		need := len(t.banner)
		if len(t.b)+need > t.max {
			extra := (len(t.b) + need) - t.max
			if extra < len(t.b) {
				t.b = t.b[extra:]
			} else {
				t.b = t.b[:0]
			}
		}
		tmp := make([]byte, 0, t.max)
		tmp = append(tmp, t.banner...)
		tmp = append(tmp, t.b...)
		t.b = tmp
	}
}

func hasPrefixBytes(b, prefix []byte) bool {
	if len(prefix) == 0 {
		return true
	}
	if len(b) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if b[i] != prefix[i] {
			return false
		}
	}
	return true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
