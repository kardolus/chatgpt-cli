package agent

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type StepEffect struct {
	Kind  string         // "file.write" | "file.patch" | "file.replace" | "shell.exec" | ...
	Path  string         // for file ops
	Bytes int            // for writes (optional)
	Meta  map[string]any // extra stats, like hunks, replaced count, exit code, etc.
}

type Effects []StepEffect

func (e Effects) HasKind(kind string) bool {
	for _, eff := range e {
		if eff.Kind == kind {
			return true
		}
	}
	return false
}

func (e Effects) HasPrefix(prefix string) bool {
	for _, eff := range e {
		if strings.HasPrefix(eff.Kind, prefix) {
			return true
		}
	}
	return false
}

func mergeEffects(dst *Effects, src Effects) {
	if len(src) == 0 {
		return
	}
	*dst = append(*dst, src...)
}

func formatEffectsForConversation(effects Effects) string {
	if len(effects) == 0 {
		return "SIDE_EFFECTS: none"
	}

	var b strings.Builder
	b.WriteString("SIDE_EFFECTS:\n")
	for _, e := range effects {
		b.WriteString("- kind=")
		b.WriteString(e.Kind)

		if e.Path != "" {
			b.WriteString(" path=")
			b.WriteString(strconv.Quote(e.Path))
		}

		if e.Bytes != 0 {
			b.WriteString(" bytes=")
			b.WriteString(strconv.Itoa(e.Bytes))
		}

		if len(e.Meta) > 0 {
			raw, err := json.Marshal(e.Meta)
			if err == nil {
				b.WriteString(" meta=")
				b.WriteString(string(raw))
			}
		}

		b.WriteString("\n")
	}

	return strings.TrimRightFunc(b.String(), unicode.IsSpace)
}

func summarizeEffectsForUI(effects Effects) string {
	if len(effects) == 0 {
		return "no side effects"
	}

	counts := map[string]int{}
	for _, e := range effects {
		counts[e.Kind]++
	}

	var parts []string
	for k, n := range counts {
		parts = append(parts, fmt.Sprintf("%s x%d", k, n))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}
