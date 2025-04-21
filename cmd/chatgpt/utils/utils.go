package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	AudioPattern      = "-audio"
	TranscribePattern = "-transcribe"
	TTSPattern        = "-tts"
	O1ProPattern      = "o1-pro"
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
	if !flags["speak"] && flags["output"] {
		return errors.New("the --speak flag cannot be used without the --output flag")
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
	if flags["voice"] && !strings.Contains(model, TTSPattern) {
		return errors.New("the --voice flag cannot be used without a compatible model, ie gpt-4o-mini-tts (see --list-models)")
	}
	if flags["effort"] && !strings.Contains(model, O1ProPattern) {
		return errors.New("the --effort flag cannot be used with non o1-pro models (see --list-models)")
	}

	return nil
}
