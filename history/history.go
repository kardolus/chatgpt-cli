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

	messages, err := h.store.ReadThread(thread)
	if err != nil {
		return "", err
	}

	var (
		lastRole            string
		concatenatedMessage string
	)

	for _, message := range messages {
		if message.Role == userRole && lastRole == userRole {
			concatenatedMessage += message.Content
		} else {
			if lastRole == userRole && concatenatedMessage != "" {
				result += formatMessage(types.Message{Role: userRole, Content: concatenatedMessage})
				concatenatedMessage = ""
			}

			if message.Role == userRole {
				concatenatedMessage = message.Content
			} else {
				result += formatMessage(message)
			}
		}

		lastRole = message.Role
	}

	// Handle the case where the last message is a user message and was concatenated
	if lastRole == userRole && concatenatedMessage != "" {
		result += formatMessage(types.Message{Role: userRole, Content: concatenatedMessage})
	}

	return result, nil
}

func formatMessage(msg types.Message) string {
	var (
		emoji  string
		prefix string
	)

	switch msg.Role {
	case systemRole:
		emoji = "💻"
		prefix = "\n"
	case userRole:
		emoji = "👤"
		prefix = "---\n"
	case assistantRole:
		emoji = "🤖"
		prefix = "\n"
	}

	return fmt.Sprintf("%s**%s** %s:\n%s\n", prefix, strings.ToUpper(msg.Role), emoji, msg.Content)
}
