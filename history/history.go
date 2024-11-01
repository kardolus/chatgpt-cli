package history

import (
	"fmt"
	"github.com/kardolus/chatgpt-cli/types"
	"strings"
)

const (
	assistantRole = "assistant"
	systemRole    = "system"
	userRole      = "user"
)

type History struct {
	store HistoryStore
}

func NewHistory(store HistoryStore) *History {
	return &History{store: store}
}

func (h *History) Print(thread string) (string, error) {
	var result string

	historyEntries, err := h.store.ReadThread(thread)
	if err != nil {
		return "", err
	}

	var (
		lastRole            string
		concatenatedMessage string
	)

	for _, entry := range historyEntries {
		if entry.Role == userRole && lastRole == userRole {
			concatenatedMessage += entry.Content
		} else {
			if lastRole == userRole && concatenatedMessage != "" {
				result += formatHistory(types.History{
					Message:   types.Message{Role: userRole, Content: concatenatedMessage},
					Timestamp: entry.Timestamp,
				})
				concatenatedMessage = ""
			}

			if entry.Role == userRole {
				concatenatedMessage = entry.Content
			} else {
				result += formatHistory(types.History{
					Message:   entry.Message,
					Timestamp: entry.Timestamp,
				})
			}
		}

		lastRole = entry.Role
	}

	// Handle the case where the last entry is a user entry and was concatenated
	if lastRole == userRole && concatenatedMessage != "" {
		result += formatHistory(types.History{
			Message: types.Message{Role: userRole, Content: concatenatedMessage},
		})
	}

	return result, nil
}

func formatHistory(entry types.History) string {
	var (
		emoji     string
		prefix    string
		timestamp string
	)

	switch entry.Role {
	case systemRole:
		emoji = "💻"
		prefix = "\n"
	case userRole:
		emoji = "👤"
		prefix = "---\n"
		if !entry.Timestamp.IsZero() {
			timestamp = fmt.Sprintf(" [%s]", entry.Timestamp.Format("2006-01-02 15:04:05"))
		}
	case assistantRole:
		emoji = "🤖"
		prefix = "\n"
	}

	return fmt.Sprintf("%s**%s** %s%s:\n%s\n", prefix, strings.ToUpper(entry.Role), emoji, timestamp, entry.Content)
}
