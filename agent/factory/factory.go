package factory

import (
	"context"
	"fmt"
	"github.com/kardolus/chatgpt-cli/agent/core"
	"github.com/kardolus/chatgpt-cli/agent/planexec"
	"github.com/kardolus/chatgpt-cli/agent/react"
	"github.com/kardolus/chatgpt-cli/agent/tools"
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
	Clock   core.Clock
	Planner planexec.Planner
	LLM     tools.LLM
	Runner  core.Runner
	Budget  core.Budget
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

func New(mode Mode, deps Deps, baseOpts ...core.BaseOption) (Agent, error) {
	if err := validateDepsForMode(mode, deps); err != nil {
		return nil, err
	}

	base := core.NewBaseAgent(deps.Clock)
	for _, o := range baseOpts {
		o(base)
	}

	switch mode {
	case ModePlanExecute:
		return &planexec.PlanExecuteAgent{BaseAgent: base, Planner: deps.Planner, Runner: deps.Runner}, nil
	case ModeReAct:
		return &react.ReActAgent{BaseAgent: base, LLM: deps.LLM, Runner: deps.Runner, Budget: deps.Budget}, nil
	default:
		return nil, fmt.Errorf("unknown agent mode: %q", mode)
	}
}
