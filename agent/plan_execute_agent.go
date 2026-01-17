package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"go.uber.org/zap"
)

// Agent pipeline: goal → Planner → Steps → Runner → Tools (Shell | LLM | FileOps), governed by Budget/Policy.

// 	Iteration 1: goal → plan → run(plan)
//	Iteration 2: goal → plan → run until failure → planner(repair) → run repair
//	Iteration 3: (repeat pipeline per chunk/step): goal → planner(next) → step(s) → runner → observe → planner(next) ..

type PlanExecuteAgent struct {
	*BaseAgent
	planner Planner
	runner  Runner
}

func NewPlanExecuteAgent(clk Clock, pl Planner, run Runner, opts ...BaseOption) *PlanExecuteAgent {
	base := NewBaseAgent(clk)
	for _, o := range opts {
		o(base)
	}

	return &PlanExecuteAgent{
		BaseAgent: base,
		planner:   pl,
		runner:    run,
	}
}

func (a *PlanExecuteAgent) RunAgentGoal(ctx context.Context, goal string) (string, error) {
	start := a.startTimer()
	defer a.finishTimer(start)

	a.logMode(goal, "Plan-Execute")

	out := a.out
	dbg := a.debug

	plan, err := a.planner.Plan(ctx, goal)
	if err != nil {
		dbg.Errorf("planner error: %v", err)
		return "", err
	}

	dbg.Debugf("plan goal=%q steps=%d", plan.Goal, len(plan.Steps))

	execCtx := ExecContext{Goal: goal, Plan: plan, Results: nil}

	out.Infof("Goal: %s", plan.Goal)
	out.Infof("Mode: Plan-and-Execute (plan first, then run tools)\n")

	for i, s := range plan.Steps {
		switch s.Type {
		case ToolShell:
			out.Infof("Step %d/%d: %s (shell %s %v)", i+1, len(plan.Steps), s.Description, s.Command, s.Args)
		case ToolLLM:
			out.Infof("Step %d/%d: %s (llm prompt_len=%d)", i+1, len(plan.Steps), s.Description, len(s.Prompt))
		case ToolFiles:
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

		res, err := a.runner.RunStep(ctx, a.config, rendered)
		if err != nil {
			if isBudgetStop(err, out) || isPolicyStop(err, out) {
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

		if res.Outcome == OutcomeError {
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

func isBudgetStop(err error, log *zap.SugaredLogger) bool {
	var be BudgetExceededError
	if errors.As(err, &be) {
		log.Warnf("Budget exceeded (kind=%s): %v", be.Kind, err)
		return true
	}
	return false
}

func isPolicyStop(err error, log *zap.SugaredLogger) bool {
	var pe PolicyDeniedError
	if errors.As(err, &pe) {
		log.Warnf("Policy denied (kind=%s): %v", pe.Kind, err)
		return true
	}
	return false
}
