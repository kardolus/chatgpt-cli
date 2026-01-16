package agent

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type Agent interface {
	RunAgentGoal(ctx context.Context, goal string) error
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

func New(mode Mode, deps Deps, baseOpts ...func(*BaseAgent)) (Agent, error) {
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

type BaseAgent struct {
	clock  Clock
	config Config
	out    *zap.SugaredLogger
	debug  *zap.SugaredLogger
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
	b.out.Infof("Total duration: %s", b.clock.Now().Sub(start))
	b.debug.Infof("Total duration: %s", b.clock.Now().Sub(start))
}
