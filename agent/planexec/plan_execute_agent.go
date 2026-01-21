package planexec

import (
	"context"
	"fmt"
	"github.com/kardolus/chatgpt-cli/agent/core"
	"github.com/kardolus/chatgpt-cli/agent/types"
	"strings"
	"unicode"
)

// Agent pipeline: goal → Planner → Steps → Runner → Tools (Shell | LLM | FileOps), governed by Budget/Policy.

// 	Iteration 1: goal → plan → run(plan)
//	Iteration 2: goal → plan → run until failure → Planner(repair) → run repair
//	Iteration 3: (repeat pipeline per chunk/step): goal → Planner(next) → step(s) → Runner → observe → Planner(next) ..

type PlanExecuteAgent struct {
	*core.BaseAgent
	Planner Planner
	Runner  core.Runner
}

func NewPlanExecuteAgent(clk core.Clock, pl Planner, run core.Runner, opts ...core.BaseOption) *PlanExecuteAgent {
	base := core.NewBaseAgent(clk)
	for _, o := range opts {
		o(base)
	}

	return &PlanExecuteAgent{
		BaseAgent: base,
		Planner:   pl,
		Runner:    run,
	}
}

func (a *PlanExecuteAgent) RunAgentGoal(ctx context.Context, goal string) (string, error) {
	start := a.StartTimer()
	defer a.FinishTimer(start)

	a.LogMode(goal, "Plan-Execute")

	out := a.Out
	dbg := a.Debug

	plan, err := a.Planner.Plan(ctx, goal)
	if err != nil {
		dbg.Errorf("Planner error: %v", err)
		return "", err
	}

	dbg.Debugf("plan goal=%q steps=%d", plan.Goal, len(plan.Steps))

	execCtx := types.ExecContext{Goal: goal, Plan: plan, Results: nil}

	out.Infof("Goal: %s", plan.Goal)
	out.Infof("Mode: Plan-and-Execute (plan first, then run tools)\n")

	for i, s := range plan.Steps {
		switch s.Type {
		case types.ToolShell:
			out.Infof("Step %d/%d: %s (shell %s %v)", i+1, len(plan.Steps), s.Description, s.Command, s.Args)
		case types.ToolLLM:
			out.Infof("Step %d/%d: %s (llm prompt_len=%d)", i+1, len(plan.Steps), s.Description, len(s.Prompt))
		case types.ToolFiles:
			out.Infof("Step %d/%d: %s (file op=%q path=%q)", i+1, len(plan.Steps), s.Description, s.Op, s.Path)
		default:
			out.Infof("Step %d/%d: %s (type=%q)", i+1, len(plan.Steps), s.Description, s.Type)
		}
	}
	out.Info("")

	var final string
	for i, step := range plan.Steps {
		rendered, err := ApplyTemplate(step, execCtx)
		if err != nil {
			out.Errorf("Template render failed (step %d): %s: %v", i+1, step.Description, err)
			dbg.Errorf("template render failed step=%d desc=%q err=%v", i+1, step.Description, err)
			return "", err
		}

		// Debug: rendered step (includes resolved templates)
		dbg.Debugf("rendered step %d/%d: %+v", i+1, len(plan.Steps), rendered)

		res, err := a.Runner.RunStep(ctx, a.Config, rendered)
		if err != nil {
			if core.IsBudgetStop(err, out) || core.IsPolicyStop(err, out) {
				dbg.Errorf("stop error step=%d desc=%q err=%v", i+1, rendered.Description, err)
				return "", err
			}
			out.Errorf("Step failed: %s: %v", rendered.Description, err)
			dbg.Errorf("step failed step=%d desc=%q err=%v transcript=%q", i+1, rendered.Description, err, res.Transcript)
			return "", err
		}

		out.Infof("Step %d finished in %s (outcome=%s)", i+1, res.Duration, res.Outcome)

		// Debug: ALWAYS dump transcript
		if strings.TrimSpace(res.Transcript) != "" {
			dbg.Debugf("step %d transcript:\n%s", i+1, res.Transcript)
		}

		if res.Outcome == types.OutcomeError {
			if res.Transcript != "" {
				out.Errorf("Step failed: %s\n%s", rendered.Description, res.Transcript)
			}
			dbg.Errorf("step outcome error step=%d desc=%q", i+1, rendered.Description)
			return "", fmt.Errorf("step failed: %s", rendered.Description)
		}

		if res.Output != "" {
			final = res.Output
		}

		execCtx.Results = append(execCtx.Results, res)
	}

	result := strings.TrimRightFunc(final, unicode.IsSpace)
	out.Infof("\nResult: %s\n", result)
	dbg.Debugf("final (trimmed): %q", strings.TrimRightFunc(final, unicode.IsSpace))
	return result, nil
}
