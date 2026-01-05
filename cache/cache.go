package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

type Cache struct {
	store Store
}

func New(store Store) *Cache {
	return &Cache{
		store: store,
	}
}

func (c *Cache) GetSessionID(endpoint string) (string, error) {
	key := hash(endpoint)
	raw, err := c.store.Get(key)
	if err != nil {
		return "", err
	}

	var entry Entry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return "", err
	}

	return entry.SessionID, nil
}

func (c *Cache) SetSessionID(endpoint string, sessionId string) error {
	key := hash(endpoint)

	entry := Entry{
		Endpoint:  endpoint,
		SessionID: sessionId,
		UpdatedAt: time.Now(),
	}

	bytes, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	return c.store.Set(key, bytes)
}

func (c *Cache) DeleteSessionID(endpoint string) error {
	key := hash(endpoint)
	return c.store.Delete(key)
}

func hash(endpoint string) string {
	sum := sha256.Sum256([]byte(endpoint))
	return hex.EncodeToString(sum[:])
}
