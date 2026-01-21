package utils

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"regexp"
)

var hunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

type diffOp struct {
	kind byte   // ' ', '+', '-'
	line []byte // content WITHOUT the prefix; includes trailing newline if present in diff
}

type diffHunk struct {
	oldStart int // 1-based
	ops      []diffOp
}

func ApplyUnifiedDiff(orig, unified []byte) ([]byte, error) {
	origLines := splitLinesKeepNL(orig)

	hunks, err := ParseUnifiedDiff(unified)
	if err != nil {
		return nil, err
	}
	if len(hunks) == 0 {
		// Treat empty patch as no-op
		return orig, nil
	}

	// Build output by walking orig and applying hunks in order.
	var out [][]byte
	origIdx := 0 // 0-based into origLines

	for _, hk := range hunks {
		// Copy any untouched lines before the hunk.
		targetIdx := hk.oldStart - 1
		if targetIdx < 0 {
			return nil, fmt.Errorf("invalid hunk oldStart=%d", hk.oldStart)
		}
		if targetIdx > len(origLines) {
			return nil, fmt.Errorf("hunk starts past EOF: start=%d len=%d", hk.oldStart, len(origLines))
		}

		// Copy from current position up to hunk start.
		if targetIdx < origIdx {
			return nil, fmt.Errorf("overlapping or out-of-order hunks: oldStart=%d", hk.oldStart)
		}
		out = append(out, origLines[origIdx:targetIdx]...)
		origIdx = targetIdx

		// Apply ops.
		for _, o := range hk.ops {
			switch o.kind {
			case ' ':
				if origIdx >= len(origLines) {
					return nil, errors.New("patch context extends past EOF")
				}
				if !equalLineContext(origLines[origIdx], o.line) {
					return nil, fmt.Errorf("patch context mismatch at line %d", origIdx+1)
				}
				out = append(out, origLines[origIdx])
				origIdx++

			case '-':
				// deletion must match
				if origIdx >= len(origLines) {
					return nil, errors.New("patch deletion extends past EOF")
				}
				isEOF := origIdx == len(origLines)-1
				if !equalLine(origLines[origIdx], o.line, isEOF) {
					return nil, fmt.Errorf("patch deletion mismatch at line %d", origIdx+1)
				}
				origIdx++ // skip

			case '+':
				// insertion
				out = append(out, o.line)

			default:
				return nil, fmt.Errorf("unknown diff op %q", o.kind)
			}
		}
	}

	// Copy remaining lines after last hunk.
	out = append(out, origLines[origIdx:]...)

	// Join
	return bytes.Join(out, nil), nil
}

func ParseUnifiedDiff(unified []byte) ([]diffHunk, error) {
	sc := bufio.NewScanner(bytes.NewReader(unified))

	const max = 10 * 1024 * 1024
	sc.Buffer(make([]byte, 0, 64*1024), max)

	var hunks []diffHunk
	var cur *diffHunk

	// Track last op so we can apply "\ No newline at end of file"
	var lastOp *diffOp

	for sc.Scan() {
		line := sc.Bytes() // no trailing '\n'

		if bytes.HasPrefix(line, []byte("diff ")) ||
			bytes.HasPrefix(line, []byte("index ")) ||
			bytes.HasPrefix(line, []byte("--- ")) ||
			bytes.HasPrefix(line, []byte("+++ ")) {
			continue
		}

		if m := hunkHeaderRe.FindSubmatch(line); m != nil {
			oldStart, err := atoiBytes(m[1])
			if err != nil {
				return nil, fmt.Errorf("invalid hunk header: %w", err)
			}
			hunks = append(hunks, diffHunk{oldStart: oldStart})
			cur = &hunks[len(hunks)-1]
			lastOp = nil
			continue
		}

		if cur == nil {
			if len(bytes.TrimSpace(line)) == 0 {
				continue
			}
			return nil, errors.New("invalid unified diff: missing hunk header")
		}

		// IMPORTANT: this marker refers to the *previous* diff line
		if bytes.HasPrefix(line, []byte(`\ No newline at end of file`)) {
			if lastOp != nil && len(lastOp.line) > 0 && lastOp.line[len(lastOp.line)-1] == '\n' {
				lastOp.line = lastOp.line[:len(lastOp.line)-1]
			}
			continue
		}

		if len(line) == 0 {
			return nil, errors.New("invalid unified diff: empty line without prefix")
		}

		prefix := line[0]
		if prefix != ' ' && prefix != '+' && prefix != '-' {
			return nil, fmt.Errorf("invalid diff line prefix %q", prefix)
		}

		// Add newline back (scanner strips it)
		ln := append([]byte(nil), line...)
		ln = append(ln, '\n')

		content := append([]byte(nil), ln[1:]...) // WITHOUT prefix, WITH newline
		cur.ops = append(cur.ops, diffOp{kind: prefix, line: content})

		// point at the stored op so we can mutate it if marker appears next
		lastOp = &cur.ops[len(cur.ops)-1]
	}

	if err := sc.Err(); err != nil {
		return nil, err
	}

	return hunks, nil
}

func splitLinesKeepNL(b []byte) [][]byte {
	if len(b) == 0 {
		return nil
	}
	var lines [][]byte
	start := 0
	for i := 0; i < len(b); i++ {
		if b[i] == '\n' {
			lines = append(lines, b[start:i+1])
			start = i + 1
		}
	}
	if start < len(b) {
		// last line without newline
		lines = append(lines, b[start:])
	}
	return lines
}

func atoiBytes(b []byte) (int, error) {
	n := 0
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, errors.New("non-digit")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func equalLine(origLine, diffLine []byte, isEOFLine bool) bool {
	if bytes.Equal(origLine, diffLine) {
		return true
	}
	if !isEOFLine {
		return false
	}
	// If original last line has no '\n', allow diff line to include it.
	if len(origLine) > 0 && origLine[len(origLine)-1] != '\n' &&
		len(diffLine) > 0 && diffLine[len(diffLine)-1] == '\n' {
		return bytes.Equal(origLine, diffLine[:len(diffLine)-1])
	}
	return false
}

func equalLineContext(origLine, diffLine []byte) bool {
	if bytes.Equal(origLine, diffLine) {
		return true
	}
	return bytes.Equal(normalizeForContextCompare(origLine), normalizeForContextCompare(diffLine))
}

func normalizeForContextCompare(line []byte) []byte {
	// Work on a copy? We can just compute slices since we only trim.
	b := line

	// Drop trailing newline (context diffLine typically includes '\n'; orig may not at EOF)
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	// Drop trailing CR (CRLF)
	if len(b) > 0 && b[len(b)-1] == '\r' {
		b = b[:len(b)-1]
	}

	// Trim trailing spaces/tabs
	i := len(b)
	for i > 0 {
		c := b[i-1]
		if c == ' ' || c == '\t' {
			i--
			continue
		}
		break
	}
	b = b[:i]

	return b
}
