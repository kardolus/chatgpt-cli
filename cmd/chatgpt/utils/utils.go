package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-cli/api"
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
	InvalidMCPPatter       = "the MCP pattern has to be of the form <provider>/<plugin>[@<version>]"
	ApifyProvider          = "apify"
	UnsupportedProvider    = "only apify is currently supported"
	LatestVersion          = "latest"
	InvalidParams          = "params need to be pairs or a JSON object"
	InvalidApifyFunction   = "apify functions need to be of the form user~actor"
	InteractiveHistoryFile = "interactive_history.txt"
	InteractivePrefix      = "int_"
	CommandPrefix          = "cmd_"
)

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
	if !flags["mcp"] && flags["param"] {
		return errors.New("the --param flag cannot be used without the --mcp flag")
	}
	if !flags["mcp"] && flags["params"] {
		return errors.New("the --params flag cannot be used without the --mcp flag")
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

// ParseMCPPlugin expects input for the apify provider of the form [provider]/[user]~[actor]@[version]
func ParseMCPPlugin(input string) (api.MCPRequest, error) {
	var result api.MCPRequest

	fields := strings.Split(input, "/")
	if len(fields) != 2 || fields[0] == "" || fields[1] == "" {
		return api.MCPRequest{}, errors.New(InvalidMCPPatter)
	}

	validProviders := map[string]bool{
		ApifyProvider: true,
	}

	if validProviders[strings.ToLower(fields[0])] {
		result.Provider = fields[0]
	} else {
		return api.MCPRequest{}, errors.New(UnsupportedProvider)
	}

	function := strings.Split(fields[1], "@")

	result.Function = function[0]

	if result.Provider == ApifyProvider {
		parts := strings.Split(result.Function, "~")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return api.MCPRequest{}, errors.New(InvalidApifyFunction)
		}
	}

	if len(function) == 1 {
		result.Version = LatestVersion
	} else if len(function) == 2 {
		result.Version = function[1]
	}

	return result, nil
}

func ParseParams(params ...string) (map[string]interface{}, error) {
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
