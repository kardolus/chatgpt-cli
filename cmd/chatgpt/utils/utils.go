package utils

import (
	"fmt"
	"os"
	"strings"
	"time"
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
