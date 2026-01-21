package core

import (
	"github.com/kardolus/chatgpt-cli/agent/types"
	"go.uber.org/zap"
	"strings"
	"time"
)

type BaseAgent struct {
	Clock  Clock
	Config types.Config

	Out   *zap.SugaredLogger
	Debug *zap.SugaredLogger

	SyncOut   func()
	SyncDebug func()
}

type BaseOption func(*BaseAgent)

func WithDryRun(v bool) BaseOption {
	return func(b *BaseAgent) { b.Config.DryRun = v }
}

func WithWorkDir(d string) BaseOption {
	return func(b *BaseAgent) {
		d = strings.TrimSpace(d)
		if d != "" {
			b.Config.WorkDir = d
		}
	}
}

func WithHumanLogger(l *zap.SugaredLogger, sync func()) BaseOption {
	return func(b *BaseAgent) {
		if l != nil {
			b.Out = l
		}
		if sync != nil {
			b.SyncOut = sync
		}
	}
}

func WithDebugLogger(l *zap.SugaredLogger, sync func()) BaseOption {
	return func(b *BaseAgent) {
		if l != nil {
			b.Debug = l
		}
		if sync != nil {
			b.SyncDebug = sync
		}
	}
}

func NewBaseAgent(clock Clock) *BaseAgent {
	return &BaseAgent{
		Clock:  clock,
		Config: types.Config{DryRun: false, WorkDir: "."},
		Out:    zap.NewNop().Sugar(),
		Debug:  zap.NewNop().Sugar(),
	}
}

func (b *BaseAgent) LogMode(goal, mode string) {
	b.Out.Infof("Goal: %s", goal)
	if mode != "" {
		b.Out.Infof("Mode: %s\n", mode)
	} else {
		b.Out.Info("")
	}
}

func (b *BaseAgent) StartTimer() time.Time {
	return b.Clock.Now()
}

func (b *BaseAgent) FinishTimer(start time.Time) {
	dur := b.Clock.Now().Sub(start)
	b.Out.Infof("Total duration: %s", dur)
	b.Debug.Infof("Total duration: %s", dur)

	if b.SyncOut != nil {
		b.SyncOut()
	}
	if b.SyncDebug != nil {
		b.SyncDebug()
	}
}
