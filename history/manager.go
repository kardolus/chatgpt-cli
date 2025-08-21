package history

import (
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-cli/api"
	"os"
	"strings"
)

const (
	assistantRole = "assistant"
	systemRole    = "system"
	userRole      = "user"
	functionRole  = "function"
)

type Manager struct {
	store Store
}

func NewHistory(store Store) *Manager {
	return &Manager{store: store}
}

func (h *Manager) ParseUserHistory(thread string) ([]string, error) {
	var result []string

	historyEntries, err := h.store.ReadThread(thread)
	if err != nil {
		// Gracefully handle missing file
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		// Return any other error
		return nil, err
	}

	for _, entry := range historyEntries {
		if entry.Role == userRole {
			if s, ok := entry.Content.(string); ok {
				result = append(result, s)
			}
		}
	}

	return result, nil
}

func (h *Manager) Print(thread string) (string, error) {
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
			concatenatedMessage += entry.Content.(string)
		} else {
			if lastRole == userRole && concatenatedMessage != "" {
				result += formatHistory(History{
					Message:   api.Message{Role: userRole, Content: concatenatedMessage},
					Timestamp: entry.Timestamp,
				})
				concatenatedMessage = ""
			}

			if entry.Role == userRole {
				concatenatedMessage = entry.Content.(string)
			} else {
				result += formatHistory(History{
					Message:   entry.Message,
					Timestamp: entry.Timestamp,
				})
			}
		}

		lastRole = entry.Role
	}

	// Handle the case where the last entry is a user entry and was concatenated
	if lastRole == userRole && concatenatedMessage != "" {
		result += formatHistory(History{
			Message: api.Message{Role: userRole, Content: concatenatedMessage},
		})
	}

	return result, nil
}

func formatHistory(entry History) string {
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
	case functionRole:
		emoji = "🔌"
		prefix = "---\n"
	case assistantRole:
		emoji = "🤖"
		prefix = "\n"
	}

	return fmt.Sprintf("%s**%s** %s%s:\n%s\n", prefix, strings.ToUpper(entry.Role), emoji, timestamp, entry.Content)
}
