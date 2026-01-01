package client

import (
	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/history"
	"strings"
	"unicode/utf8"
)

const (
	MaxTokenBufferPercentage = 20
	SystemRole               = "system"
)

// ProvideContext adds custom context to the client's history by converting the
// provided string into a series of messages. This allows the ChatGPT API to have
// prior knowledge of the provided context when generating responses.
//
// The context string should contain the text you want to provide as context,
// and the method will split it into messages, preserving punctuation and special
// characters.
func (c *Client) ProvideContext(context string) {
	c.initHistory()
	historyEntries := c.createHistoryEntriesFromString(context)
	c.History = append(c.History, historyEntries...)
}

func (c *Client) createHistoryEntriesFromString(input string) []history.History {
	var result []history.History

	words := strings.Fields(input)

	for i := 0; i < len(words); i += 100 {
		end := i + 100
		if end > len(words) {
			end = len(words)
		}

		content := strings.Join(words[i:end], " ")

		item := history.History{
			Message: api.Message{
				Role:    UserRole,
				Content: content,
			},
			Timestamp: c.timer.Now(),
		}
		result = append(result, item)
	}

	return result
}

func (c *Client) initHistory() {
	if len(c.History) != 0 {
		return
	}

	if !c.Config.OmitHistory {
		c.History, _ = c.historyStore.Read()
	}

	if len(c.History) == 0 {
		c.History = []history.History{{
			Message: api.Message{
				Role: SystemRole,
			},
			Timestamp: c.timer.Now(),
		}}
	}

	c.History[0].Content = c.Config.Role
}

func (c *Client) truncateHistory() {
	tokens, rolling := countTokens(c.History)
	effectiveTokenSize := calculateEffectiveContextWindow(c.Config.ContextWindow, MaxTokenBufferPercentage)

	if tokens <= effectiveTokenSize {
		return
	}

	var index int
	var total int
	diff := tokens - effectiveTokenSize

	for i := 1; i < len(rolling); i++ {
		total += rolling[i]
		if total > diff {
			index = i
			break
		}
	}

	c.History = append(c.History[:1], c.History[index+1:]...)
}

func (c *Client) updateHistory(response string) {
	c.History = append(c.History, history.History{
		Message: api.Message{
			Role:    AssistantRole,
			Content: response,
		},
		Timestamp: c.timer.Now(),
	})

	if !c.Config.OmitHistory {
		_ = c.historyStore.Write(c.History)
	}
}

func calculateEffectiveContextWindow(window int, bufferPercentage int) int {
	adjustedPercentage := 100 - bufferPercentage
	effectiveContextWindow := (window * adjustedPercentage) / 100
	return effectiveContextWindow
}

func countTokens(entries []history.History) (int, []int) {
	var result int
	var rolling []int

	for _, entry := range entries {
		charCount, wordCount := 0, 0
		words := strings.Fields(entry.Content.(string))
		wordCount += len(words)

		for _, word := range words {
			charCount += utf8.RuneCountInString(word)
		}

		// This is a simple approximation; actual token count may differ.
		// You can adjust this based on your language and the specific tokenizer used by the model.
		tokenCountForMessage := (charCount + wordCount) / 2
		result += tokenCountForMessage
		rolling = append(rolling, tokenCountForMessage)
	}

	return result, rolling
}
