package core

import (
	"context"
	"time"
)

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type RealClock struct{}

func NewRealClock() *RealClock { return &RealClock{} }

func (c *RealClock) Now() time.Time { return time.Now() }

func (c *RealClock) Sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
