package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

type Agent interface {
	RunAgentGoal(ctx context.Context, goal string) (string, error)
}

type Mode string

const (
	ModePlanExecute Mode = "plan_execute"
	ModeReAct       Mode = "react"
)

type Deps struct {
	Clock   Clock
	Planner Planner
	LLM     LLM
	Runner  Runner
	Budget  Budget
}

type BaseAgent struct {
	clock  Clock
	config Config

	out   *zap.SugaredLogger
	debug *zap.SugaredLogger

	syncOut   func()
	syncDebug func()
}

type BaseOption func(*BaseAgent)

func WithDryRun(v bool) BaseOption {
	return func(b *BaseAgent) { b.config.DryRun = v }
}

func WithWorkDir(d string) BaseOption {
	return func(b *BaseAgent) {
		d = strings.TrimSpace(d)
		if d != "" {
			b.config.WorkDir = d
		}
	}
}

func WithHumanLogger(l *zap.SugaredLogger, sync func()) BaseOption {
	return func(b *BaseAgent) {
		if l != nil {
			b.out = l
		}
		if sync != nil {
			b.syncOut = sync
		}
	}
}

func WithDebugLogger(l *zap.SugaredLogger, sync func()) BaseOption {
	return func(b *BaseAgent) {
		if l != nil {
			b.debug = l
		}
		if sync != nil {
			b.syncDebug = sync
		}
	}
}

func validateDepsForMode(mode Mode, deps Deps) error {
	if deps.Clock == nil {
		return fmt.Errorf("agent deps: Clock is required")
	}

	switch mode {
	case ModePlanExecute:
		if deps.Planner == nil {
			return fmt.Errorf("agent deps: Planner is required for mode %q", mode)
		}
		if deps.Runner == nil {
			return fmt.Errorf("agent deps: Runner is required for mode %q", mode)
		}
		return nil

	case ModeReAct:
		if deps.LLM == nil {
			return fmt.Errorf("agent deps: LLM is required for mode %q", mode)
		}
		if deps.Runner == nil {
			return fmt.Errorf("agent deps: Runner is required for mode %q", mode)
		}
		if deps.Budget == nil {
			return fmt.Errorf("agent deps: Budget is required for mode %q", mode)
		}
		return nil

	default:
		return fmt.Errorf("unknown agent mode: %q", mode)
	}
}

func New(mode Mode, deps Deps, baseOpts ...BaseOption) (Agent, error) {
	if err := validateDepsForMode(mode, deps); err != nil {
		return nil, err
	}

	base := NewBaseAgent(deps.Clock)
	for _, o := range baseOpts {
		o(base)
	}

	switch mode {
	case ModePlanExecute:
		return &PlanExecuteAgent{BaseAgent: base, planner: deps.Planner, runner: deps.Runner}, nil
	case ModeReAct:
		return &ReActAgent{BaseAgent: base, llm: deps.LLM, runner: deps.Runner, budget: deps.Budget}, nil
	default:
		return nil, fmt.Errorf("unknown agent mode: %q", mode)
	}
}

func NewBaseAgent(clock Clock) *BaseAgent {
	return &BaseAgent{
		clock:  clock,
		config: Config{DryRun: false, WorkDir: "."},
		out:    zap.NewNop().Sugar(),
		debug:  zap.NewNop().Sugar(),
	}
}

func (b *BaseAgent) logMode(goal, mode string) {
	b.out.Infof("Goal: %s", goal)
	if mode != "" {
		b.out.Infof("Mode: %s\n", mode)
	} else {
		b.out.Info("")
	}
}

func (b *BaseAgent) startTimer() time.Time {
	return b.clock.Now()
}

func (b *BaseAgent) finishTimer(start time.Time) {
	dur := b.clock.Now().Sub(start)
	b.out.Infof("Total duration: %s", dur)
	b.debug.Infof("Total duration: %s", dur)

	if b.syncOut != nil {
		b.syncOut()
	}
	if b.syncDebug != nil {
		b.syncDebug()
	}
}
