package core

import (
	"github.com/kardolus/chatgpt-cli/agent/types"
	"go.uber.org/zap"
	"strings"
	"time"
)

const (
	defaultTranscriptMaxBytes    = 512 * 1024
	defaultPromptHistoryMaxBytes = 512 * 1024
)

type BaseAgent struct {
	Clock  Clock
	Config types.Config

	Out   *zap.SugaredLogger
	Debug *zap.SugaredLogger

	SyncOut   func()
	SyncDebug func()

	Transcript    *TranscriptBuffer
	PromptHistory *TranscriptBuffer
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

func WithTranscriptMaxBytes(n int) BaseOption {
	return func(b *BaseAgent) {
		if n > 0 {
			b.Transcript = NewTranscriptBuffer(n)
		}
	}
}

func WithPromptHistoryMaxBytes(n int) BaseOption {
	return func(b *BaseAgent) {
		if n > 0 {
			b.PromptHistory = NewTranscriptBuffer(n)
		}
	}
}

func NewBaseAgent(clock Clock) *BaseAgent {
	return &BaseAgent{
		Clock:         clock,
		Config:        types.Config{DryRun: false, WorkDir: "."},
		Out:           zap.NewNop().Sugar(),
		Debug:         zap.NewNop().Sugar(),
		Transcript:    NewTranscriptBuffer(defaultTranscriptMaxBytes),
		PromptHistory: NewTranscriptBuffer(defaultPromptHistoryMaxBytes),
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

func (b *BaseAgent) AddTranscript(s string) {
	if strings.TrimSpace(s) == "" {
		return
	}
	if b.Transcript == nil {
		b.Transcript = NewTranscriptBuffer(defaultTranscriptMaxBytes)
	}
	b.Transcript.AppendString(s)
}

func (b *BaseAgent) AddTranscriptf(format string, args ...any) {
	if b.Transcript == nil {
		b.Transcript = NewTranscriptBuffer(defaultTranscriptMaxBytes)
	}
	b.Transcript.Appendf(format, args...)
}

func (b *BaseAgent) TranscriptString() string {
	if b.Transcript == nil {
		return ""
	}
	return b.Transcript.String()
}

func (b *BaseAgent) AddHistory(s string) {
	if strings.TrimSpace(s) == "" {
		return
	}
	if b.PromptHistory == nil {
		b.PromptHistory = NewTranscriptBuffer(defaultPromptHistoryMaxBytes)
	}
	b.PromptHistory.AppendString(s)
}

func (b *BaseAgent) AddHistoryf(format string, args ...any) {
	if b.PromptHistory == nil {
		b.PromptHistory = NewTranscriptBuffer(defaultPromptHistoryMaxBytes)
	}
	b.PromptHistory.Appendf(format, args...)
}

func (b *BaseAgent) History() string {
	if b.PromptHistory == nil {
		return ""
	}
	return b.PromptHistory.String()
}
