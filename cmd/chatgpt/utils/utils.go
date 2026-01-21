package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-cli/agent"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/internal"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	AudioPattern           = "-audio"
	TranscribePattern      = "-transcribe"
	TTSPattern             = "-tts"
	ImagePattern           = "-image"
	O1ProPattern           = "o1-pro"
	GPT5Pattern            = "gpt-5"
	InvalidParams          = "params need to be pairs or a JSON object"
	InteractiveHistoryFile = "interactive_history.txt"
	InteractivePrefix      = "int_"
	CommandPrefix          = "cmd_"
)

func BudgetLimitsFromConfig(cfg config.Config) agent.BudgetLimits {
	return agent.BudgetLimits{
		MaxIterations: cfg.Agent.MaxIterations,
		MaxWallTime:   time.Duration(cfg.Agent.MaxWallTime) * time.Second,
		MaxShellCalls: cfg.Agent.MaxShellCalls,
		MaxLLMCalls:   cfg.Agent.MaxLLMCalls,
		MaxFileOps:    cfg.Agent.MaxFileOps,
		MaxLLMTokens:  cfg.Agent.MaxLLMTokens,
	}
}

func ColorToAnsi(color string) (string, string) {
	if color == "" {
		return "", ""
	}

	color = strings.ToLower(strings.TrimSpace(color))

	reset := "\033[0m"

	switch color {
	case "red":
		return "\033[31m", reset
	case "green":
		return "\033[32m", reset
	case "yellow":
		return "\033[33m", reset
	case "blue":
		return "\033[34m", reset
	case "magenta":
		return "\033[35m", reset
	default:
		return "", ""
	}
}

func CreateHistoryFile(history []string) (string, error) {
	dataHome, err := internal.GetDataHome()
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(dataHome, InteractiveHistoryFile)

	content := strings.Join(history, "\n") + "\n"
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return "", err
	}

	return fullPath, nil
}

func FileToString(fileName string) (string, error) {
	bytes, err := os.ReadFile(fileName)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

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

func GenerateThreadName(cfg config.Config, interactive, newThread bool) (result string, updateConfig bool) {
	if !cfg.AutoCreateNewThread && !newThread {
		return cfg.Thread, false
	}

	if newThread {
		return internal.GenerateUniqueSlug(CommandPrefix), true
	}

	if interactive && cfg.AutoCreateNewThread {
		return internal.GenerateUniqueSlug(InteractivePrefix), false
	}

	return cfg.Thread, false
}

func IsBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// Only check up to 512KB to avoid memory issues with large files
	const maxBytes = 512 * 1024
	checkSize := len(data)
	if checkSize > maxBytes {
		checkSize = maxBytes
	}

	// Check if the sample is valid UTF-8
	if !utf8.Valid(data[:checkSize]) {
		return true
	}

	// Count suspicious bytes in the sample
	binaryCount := 0
	for _, b := range data[:checkSize] {
		if b == 0 {
			return true
		}

		if b < 32 && b != 9 && b != 10 && b != 13 {
			binaryCount++
		}
	}

	threshold := checkSize * 10 / 100
	return binaryCount > threshold
}

func ValidateFlags(model string, flags map[string]bool) error {
	if flags["new-thread"] && (flags["set-thread"] || flags["thread"]) {
		return errors.New("the --new-thread flag cannot be used with the --set-thread or --thread flags")
	}
	if flags["speak"] && !flags["output"] {
		return errors.New("the --speak flag cannot be used without the --output flag")
	}
	if flags["draw"] && !flags["output"] {
		return errors.New("the --draw flag cannot be used without the --output flag")
	}
	if !flags["speak"] && !flags["draw"] && flags["output"] {
		return errors.New("the --output flag cannot be used without the --speak or --draw flag")
	}
	if !flags["mcp"] && flags["mcp-param"] {
		return errors.New("the --mcp-param flag cannot be used without the --mcp flag")
	}
	if !flags["mcp"] && flags["mcp-params"] {
		return errors.New("the --mcp-params flag cannot be used without the --mcp flag")
	}
	if !flags["agent"] && flags["agent-mode"] {
		return errors.New("the --agent-mode flag cannot be used without the --agent flag")
	}
	if flags["audio"] && !strings.Contains(model, AudioPattern) {
		return errors.New("the --audio flag cannot be used without a compatible model, ie gpt-4o-audio-preview (see --list-models)")
	}
	if flags["transcribe"] && !strings.Contains(model, TranscribePattern) {
		return errors.New("the --transcribe flag cannot be used without a compatible model, ie gpt-4o-transcribe (see --list-models)")
	}
	if flags["speak"] && flags["output"] && !strings.Contains(model, TTSPattern) {
		return errors.New("the --speak and --output flags cannot be used without a compatible model, ie gpt-4o-mini-tts (see --list-models)")
	}
	if flags["draw"] && flags["output"] && !strings.Contains(model, ImagePattern) {
		return errors.New("the --draw and --output flags cannot be used without a compatible model, ie gpt-image-1 (see --list-models)")
	}
	if flags["voice"] && !strings.Contains(model, TTSPattern) {
		return errors.New("the --voice flag cannot be used without a compatible model, ie gpt-4o-mini-tts (see --list-models)")
	}
	if flags["effort"] && !(strings.Contains(model, O1ProPattern) || strings.Contains(model, GPT5Pattern)) {
		return errors.New("the --effort flag cannot be used with non o1-pro or gpt-5 models (see --list-models)")
	}

	return nil
}

func ParseMCPHeaders(in []string) (map[string]string, error) {
	out := map[string]string{}
	for _, h := range in {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --mcp-header %q (expected 'Key: Value')", h)
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if k == "" {
			return nil, fmt.Errorf("invalid --mcp-header %q (empty key)", h)
		}
		out[k] = v
	}
	return out, nil
}

func ParseMCPParams(params ...string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	if len(params) == 1 {
		if !isJSONObject(params[0]) && !isValidPair(params[0]) {
			return nil, errors.New(InvalidParams)
		}
		if isValidPair(params[0]) {
			k, v := parseTypedValue(params[0])
			result[k] = v
			return result, nil
		}
		// the input is valid json
		if err := json.Unmarshal([]byte(params[0]), &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	for _, param := range params {
		if !isValidPair(param) {
			return nil, errors.New(InvalidParams)
		}
		k, v := parseTypedValue(param)
		result[k] = v
	}

	return result, nil
}

func parseTypedValue(param string) (string, interface{}) {
	k, raw := parsePair(param)

	// Try to unmarshal the value as JSON
	var parsed interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
		return k, parsed
	}

	// Fallback to treating it as a string
	return k, raw
}

func isJSONObject(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

func isValidPair(s string) bool {
	pairs := strings.Split(s, "=")

	if len(pairs) == 2 && pairs[0] != "" && pairs[1] != "" {
		return true
	}

	return false
}

func parsePair(s string) (string, string) {
	pairs := strings.Split(s, "=")
	return pairs[0], pairs[1]
}
