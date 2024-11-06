package history

import (
	"github.com/kardolus/chatgpt-cli/api"
	"time"
)

type History struct {
	api.Message
	Timestamp time.Time `json:"timestamp,omitempty"`
}
