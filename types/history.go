package types

import "time"

type History struct {
	Message
	Timestamp time.Time `json:"timestamp,omitempty"`
}
