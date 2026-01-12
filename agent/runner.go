package agent

import (
	"context"
	"fmt"
	"strings"
)

type OutcomeKind string

const (
	OutcomeOK     OutcomeKind = "ok"
	OutcomeDryRun OutcomeKind = "dry-run"
	OutcomeError  OutcomeKind = "error"
)

const transcriptMaxBytes = 64_000

type Tools struct {
	Shell Shell
	LLM   LLM
	Files Files
}

//go:generate mockgen -destination=runnermocks_test.go -package=agent_test github.com/kardolus/chatgpt-cli/agent Runner
type Runner interface {
	RunStep(ctx context.Context, cfg Config, step Step) (StepResult, error)
}

type DefaultRunner struct {
	tools  Tools
	clock  Clock
	budget Budget
	policy Policy
}

func NewDefaultRunner(t Tools, c Clock, b Budget, p Policy) *DefaultRunner {
	return &DefaultRunner{tools: t, clock: c, budget: b, policy: p}
}

func (r *DefaultRunner) RunStep(ctx context.Context, cfg Config, step Step) (StepResult, error) {
	start := r.clock.Now()

	if err := r.budget.AllowStep(step, start); err != nil {
		tr := appendBudgetError(buildDryRunTranscript(cfg, step), err)
		return StepResult{
			Step:       step,
			Outcome:    OutcomeError,
			Transcript: limitTranscript(tr, transcriptMaxBytes),
			Duration:   r.clock.Now().Sub(start),
		}, err
	}

	if err := r.policy.AllowStep(cfg, step); err != nil {
		var tr string
		if cfg.DryRun {
			tr = appendPolicyError(buildDryRunTranscript(cfg, step), err)
		} else {
			switch step.Type {
			case ToolShell:
				tr = appendPolicyError(buildShellStartTranscript(cfg, step), err)
			case ToolLLM:
				tr = appendPolicyError(buildLLMStartTranscript(step.Prompt), err)
			case ToolFiles:
				tr = appendPolicyError(buildFileStartTranscript(step), err)
			default:
				tr = appendPolicyError(buildUnsupportedStepTranscript(step), err)
			}
		}

		return StepResult{
			Step:       step,
			Outcome:    OutcomeError,
			Transcript: limitTranscript(tr, transcriptMaxBytes),
			Duration:   r.clock.Now().Sub(start),
		}, err
	}

	if cfg.DryRun {
		tr := buildDryRunTranscript(cfg, step)
		return StepResult{
			Step:       step,
			Outcome:    OutcomeDryRun,
			Transcript: limitTranscript(tr, transcriptMaxBytes),
			Duration:   r.clock.Now().Sub(start),
		}, nil
	}

	switch step.Type {
	case ToolShell:
		if err := r.budget.AllowTool(ToolShell, start); err != nil {
			tr := appendBudgetError(buildShellStartTranscript(cfg, step), err)
			return StepResult{
				Step:       step,
				Outcome:    OutcomeError,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
			}, err
		}

		res, err := r.tools.Shell.Run(ctx, cfg.WorkDir, step.Command, step.Args...)
		if err != nil {
			tr := buildShellStartTranscript(cfg, step)
			return StepResult{
				Step:       step,
				Outcome:    OutcomeError,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
			}, err
		}

		outcome := OutcomeOK
		if res.ExitCode != 0 {
			outcome = OutcomeError
		}

		tr := buildShellTranscript(cfg, step, res)

		out := res.Stdout
		if strings.TrimSpace(out) == "" {
			out = res.Stderr
		}

		return StepResult{
			Step:       step,
			Outcome:    outcome,
			Exec:       &res,
			Output:     out,
			Transcript: limitTranscript(tr, transcriptMaxBytes),
			Duration:   r.clock.Now().Sub(start),
		}, nil

	case ToolLLM:
		if strings.TrimSpace(step.Prompt) == "" {
			err := fmt.Errorf("llm step requires Prompt")
			tr := buildLLMStartTranscript(step.Prompt)
			return StepResult{
				Step:       step,
				Outcome:    OutcomeError,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
			}, err
		}

		// Preflight token budget: block if already out of tokens.
		snap := r.budget.Snapshot(start)
		if snap.Limits.MaxLLMTokens > 0 && snap.LLMTokensUsed >= snap.Limits.MaxLLMTokens {
			err := BudgetExceededError{
				Kind:    BudgetKindLLMTokens,
				Limit:   snap.Limits.MaxLLMTokens,
				Used:    snap.LLMTokensUsed,
				Message: "llm token budget exceeded",
			}
			tr := appendBudgetError(buildLLMStartTranscript(step.Prompt), err)
			return StepResult{
				Step:       step,
				Outcome:    OutcomeError,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
			}, err
		}

		// Charge LLM call (count + wall-time)
		if err := r.budget.AllowTool(ToolLLM, start); err != nil {
			tr := appendBudgetError(buildLLMStartTranscript(step.Prompt), err)
			return StepResult{
				Step:       step,
				Outcome:    OutcomeError,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
			}, err
		}

		out, tokens, err := r.tools.LLM.Complete(ctx, step.Prompt)
		if err != nil {
			tr := buildLLMStartTranscript(step.Prompt)
			return StepResult{
				Step:       step,
				Outcome:    OutcomeError,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
			}, err
		}

		r.budget.ChargeLLMTokens(tokens, start)

		tr := buildLLMTranscript(step.Prompt, out)
		return StepResult{
			Step:       step,
			Outcome:    OutcomeOK,
			Output:     out,
			Transcript: limitTranscript(tr, transcriptMaxBytes),
			Duration:   r.clock.Now().Sub(start),
		}, nil
	case ToolFiles:
		if strings.TrimSpace(step.Op) == "" {
			err := fmt.Errorf("file step requires Op")
			tr := buildFileStartTranscript(step)
			return StepResult{
				Step:       step,
				Outcome:    OutcomeError,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
			}, err
		}
		if strings.TrimSpace(step.Path) == "" {
			err := fmt.Errorf("file step requires Path")
			tr := buildFileStartTranscript(step)
			return StepResult{
				Step:       step,
				Outcome:    OutcomeError,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
			}, err
		}

		if err := r.budget.AllowTool(ToolFiles, start); err != nil {
			tr := appendBudgetError(buildFileStartTranscript(step), err)
			return StepResult{
				Step:       step,
				Outcome:    OutcomeError,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
			}, err
		}

		switch strings.ToLower(strings.TrimSpace(step.Op)) {
		case "read":
			b, err := r.tools.Files.ReadFile(step.Path)
			if err != nil {
				tr := buildFileStartTranscript(step)
				return StepResult{
					Step:       step,
					Outcome:    OutcomeError,
					Transcript: limitTranscript(tr, transcriptMaxBytes),
					Duration:   r.clock.Now().Sub(start),
				}, err
			}

			out := string(b)
			tr := buildFileReadTranscript(step.Path, out)
			return StepResult{
				Step:       step,
				Outcome:    OutcomeOK,
				Output:     out,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
			}, nil

		case "write":
			if err := r.tools.Files.WriteFile(step.Path, []byte(step.Data)); err != nil {
				tr := buildFileStartTranscript(step)
				return StepResult{
					Step:       step,
					Outcome:    OutcomeError,
					Transcript: limitTranscript(tr, transcriptMaxBytes),
					Duration:   r.clock.Now().Sub(start),
				}, err
			}

			tr := buildFileWriteTranscript(step.Path, step.Data)
			return StepResult{
				Step:       step,
				Outcome:    OutcomeOK,
				Output:     fmt.Sprintf("wrote %d bytes to %s", len(step.Data), step.Path),
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
			}, nil

		default:
			err := fmt.Errorf("unsupported file op: %s", step.Op)
			tr := buildFileStartTranscript(step)
			return StepResult{
				Step:       step,
				Outcome:    OutcomeError,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
			}, err
		}

	default:
		tr := buildUnsupportedStepTranscript(step)
		return StepResult{
			Step:       step,
			Outcome:    OutcomeError,
			Transcript: limitTranscript(tr, transcriptMaxBytes),
			Duration:   r.clock.Now().Sub(start),
		}, fmt.Errorf("unsupported step type: %s", step.Type)
	}
}

func appendBudgetError(tr string, err error) string {
	if tr != "" && !strings.HasSuffix(tr, "\n") {
		tr += "\n"
	}
	return tr + fmt.Sprintf("[budget] %v\n", err)
}

func appendPolicyError(tr string, err error) string {
	if tr != "" && !strings.HasSuffix(tr, "\n") {
		tr += "\n"
	}
	return tr + fmt.Sprintf("[policy] %v\n", err)
}

func buildDryRunTranscript(cfg Config, step Step) string {
	switch step.Type {
	case ToolShell:
		return fmt.Sprintf("[dry-run][shell] workdir=%q cmd=%q args=%v\n", cfg.WorkDir, step.Command, step.Args)
	case ToolLLM:
		return fmt.Sprintf("[dry-run][llm]\n%s\n", step.Prompt)
	case ToolFiles:
		return fmt.Sprintf("[dry-run][file] op=%q path=%q data_len=%d\n", step.Op, step.Path, len(step.Data))
	default:
		return fmt.Sprintf("[dry-run] step_type=%q\n", step.Type)
	}
}

func buildShellStartTranscript(cfg Config, step Step) string {
	return fmt.Sprintf(
		"[shell:start] workdir=%q cmd=%q args=%v\n",
		cfg.WorkDir,
		step.Command,
		step.Args,
	)
}

func buildShellTranscript(cfg Config, step Step, res Result) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "[shell] workdir=%q cmd=%q args=%v\n", cfg.WorkDir, step.Command, step.Args)
	_, _ = fmt.Fprintf(&b, "exit=%d\n", res.ExitCode)

	if res.Stdout != "" {
		b.WriteString("stdout:\n")
		b.WriteString(res.Stdout)
		if !strings.HasSuffix(res.Stdout, "\n") {
			b.WriteString("\n")
		}
	}
	if res.Stderr != "" {
		b.WriteString("stderr:\n")
		b.WriteString(res.Stderr)
		if !strings.HasSuffix(res.Stderr, "\n") {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func buildLLMStartTranscript(prompt string) string {
	var b strings.Builder
	b.WriteString("[llm:start]\n")
	b.WriteString("prompt:\n")
	b.WriteString(prompt)
	if !strings.HasSuffix(prompt, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

func buildLLMTranscript(prompt, output string) string {
	var b strings.Builder
	b.WriteString("[llm]\n")
	b.WriteString("prompt:\n")
	b.WriteString(prompt)
	if !strings.HasSuffix(prompt, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("output:\n")
	b.WriteString(output)
	if output != "" && !strings.HasSuffix(output, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

func buildFileStartTranscript(step Step) string {
	return fmt.Sprintf(
		"[file:start] op=%q path=%q data_len=%d\n",
		step.Op,
		step.Path,
		len(step.Data),
	)
}

func buildFileReadTranscript(path, content string) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "[file] op=%q path=%q\n", "read", path)
	b.WriteString("content:\n")
	b.WriteString(content)
	if content != "" && !strings.HasSuffix(content, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

func buildFileWriteTranscript(path, data string) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "[file] op=%q path=%q\n", "write", path)
	_, _ = fmt.Fprintf(&b, "data_len=%d\n", len(data))

	return b.String()
}

func buildUnsupportedStepTranscript(step Step) string {
	return fmt.Sprintf("[unsupported] step_type=%q\n", step.Type)
}

func limitTranscript(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "\n…(truncated)\n"
}
