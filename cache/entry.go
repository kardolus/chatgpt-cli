package cache

import "time"

type Entry struct {
	Endpoint  string    `json:"endpoint"`
	SessionID string    `json:"session_id"`
	UpdatedAt time.Time `json:"updated_at"`
}
