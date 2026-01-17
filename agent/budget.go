package agent

import (
	"fmt"
	"time"
)

//go:generate mockgen -destination=budgetmocks_test.go -package=agent_test github.com/kardolus/chatgpt-cli/agent Budget
type Budget interface {
	Start(now time.Time)
	AllowStep(step Step, now time.Time) error
	AllowTool(kind ToolKind, now time.Time) error
	AllowIteration(now time.Time) error // NEW
	ChargeLLMTokens(tokens int, now time.Time)
	Snapshot(now time.Time) BudgetSnapshot
}

const (
	BudgetKindSteps      = "steps"
	BudgetKindShell      = "shell"
	BudgetKindLLM        = "llm"
	BudgetKindFiles      = "files"
	BudgetKindLLMTokens  = "llm_tokens"
	BudgetKindWallTime   = "wall_time"
	BudgetKindIterations = "iterations"
)

type BudgetLimits struct {
	MaxSteps      int
	MaxWallTime   time.Duration
	MaxLLMTokens  int
	MaxShellCalls int
	MaxLLMCalls   int
	MaxFileOps    int
	MaxIterations int
}

type BudgetSnapshot struct {
	StartedAt      time.Time
	Elapsed        time.Duration
	Limits         BudgetLimits
	StepsUsed      int
	ShellUsed      int
	LLMUsed        int
	FileOpsUsed    int
	LLMTokensUsed  int
	IterationsUsed int
}

type DefaultBudget struct {
	limits BudgetLimits

	started   bool
	startedAt time.Time

	stepsUsed      int
	shellUsed      int
	llmUsed        int
	fileOpsUsed    int
	llmTokensUsed  int
	iterationsUsed int
}

func NewDefaultBudget(limits BudgetLimits) *DefaultBudget {
	return &DefaultBudget{limits: limits}
}

func (b *DefaultBudget) Start(now time.Time) {
	b.started = true
	b.startedAt = now
	b.stepsUsed = 0
	b.shellUsed = 0
	b.llmUsed = 0
	b.fileOpsUsed = 0
	b.llmTokensUsed = 0
	b.iterationsUsed = 0
}

func (b *DefaultBudget) Snapshot(now time.Time) BudgetSnapshot {
	b.ensureStarted(now)

	elapsed := now.Sub(b.startedAt)
	if elapsed < 0 {
		elapsed = 0
	}

	return BudgetSnapshot{
		StartedAt:      b.startedAt,
		Elapsed:        elapsed,
		Limits:         b.limits,
		StepsUsed:      b.stepsUsed,
		ShellUsed:      b.shellUsed,
		LLMUsed:        b.llmUsed,
		FileOpsUsed:    b.fileOpsUsed,
		LLMTokensUsed:  b.llmTokensUsed,
		IterationsUsed: b.iterationsUsed,
	}
}

func (b *DefaultBudget) ChargeLLMTokens(tokens int, now time.Time) {
	b.ensureStarted(now)
	if tokens <= 0 {
		return
	}
	b.llmTokensUsed += tokens
}

func (b *DefaultBudget) AllowIteration(now time.Time) error {
	b.ensureStarted(now)

	if err := b.checkWall(now); err != nil {
		return err
	}

	if b.limits.MaxIterations > 0 && b.iterationsUsed+1 > b.limits.MaxIterations {
		return BudgetExceededError{
			Kind:    BudgetKindIterations,
			Limit:   b.limits.MaxIterations,
			Used:    b.iterationsUsed,
			Message: "iteration budget exceeded",
		}
	}

	b.iterationsUsed++
	return nil
}

func (b *DefaultBudget) AllowStep(step Step, now time.Time) error {
	b.ensureStarted(now)

	if err := b.checkWall(now); err != nil {
		return err
	}

	if b.limits.MaxSteps > 0 && b.stepsUsed+1 > b.limits.MaxSteps {
		return BudgetExceededError{
			Kind:    BudgetKindSteps,
			Limit:   b.limits.MaxSteps,
			Used:    b.stepsUsed,
			Message: "step budget exceeded",
		}
	}

	b.stepsUsed++
	return nil
}

func (b *DefaultBudget) AllowTool(kind ToolKind, now time.Time) error {
	b.ensureStarted(now)

	if err := b.checkWall(now); err != nil {
		return err
	}

	switch kind {
	case ToolShell:
		if b.limits.MaxShellCalls > 0 && b.shellUsed+1 > b.limits.MaxShellCalls {
			return BudgetExceededError{
				Kind:    BudgetKindShell,
				Limit:   b.limits.MaxShellCalls,
				Used:    b.shellUsed,
				Message: "shell call budget exceeded",
			}
		}
		b.shellUsed++

	case ToolLLM:
		if b.limits.MaxLLMCalls > 0 && b.llmUsed+1 > b.limits.MaxLLMCalls {
			return BudgetExceededError{
				Kind:    BudgetKindLLM,
				Limit:   b.limits.MaxLLMCalls,
				Used:    b.llmUsed,
				Message: "llm call budget exceeded",
			}
		}
		b.llmUsed++

	case ToolFiles:
		if b.limits.MaxFileOps > 0 && b.fileOpsUsed+1 > b.limits.MaxFileOps {
			return BudgetExceededError{
				Kind:    BudgetKindFiles,
				Limit:   b.limits.MaxFileOps,
				Used:    b.fileOpsUsed,
				Message: "file ops budget exceeded",
			}
		}
		b.fileOpsUsed++

	default:
		return fmt.Errorf("unknown tool kind: %q", kind)
	}

	return nil
}

func (b *DefaultBudget) ensureStarted(now time.Time) {
	if b.started {
		return
	}
	b.Start(now)
}

func (b *DefaultBudget) checkWall(now time.Time) error {
	if b.limits.MaxWallTime <= 0 {
		return nil
	}
	elapsed := now.Sub(b.startedAt)
	if elapsed > b.limits.MaxWallTime {
		return BudgetExceededError{
			Kind:    BudgetKindWallTime,
			LimitD:  b.limits.MaxWallTime,
			UsedD:   elapsed,
			Message: "wall time budget exceeded",
		}
	}
	return nil
}

// BudgetExceededError is a typed error so the Agent/Planner can branch on it.
type BudgetExceededError struct {
	// "steps" | "shell" | "llm" | "files" | "llm_tokens" | "wall_time"
	Kind    string
	Limit   int
	Used    int
	LimitD  time.Duration
	UsedD   time.Duration
	Message string
}

func (e BudgetExceededError) Error() string {
	switch e.Kind {
	case BudgetKindWallTime:
		return fmt.Sprintf("%s: limit=%s used=%s", e.Message, e.LimitD, e.UsedD)
	default:
		return fmt.Sprintf("%s: kind=%s limit=%d used=%d", e.Message, e.Kind, e.Limit, e.Used)
	}
}
