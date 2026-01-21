package core

import (
	"context"
	"fmt"
	"github.com/kardolus/chatgpt-cli/agent/tools"
	"github.com/kardolus/chatgpt-cli/agent/types"
	"regexp"
	"strings"
	"time"
)

const transcriptMaxBytes = 64_000

type Tools struct {
	Shell tools.Shell
	LLM   tools.LLM
	Files tools.Files
}

type Runner interface {
	RunStep(ctx context.Context, cfg types.Config, step types.Step) (types.StepResult, error)
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

func (r *DefaultRunner) RunStep(ctx context.Context, cfg types.Config, step types.Step) (types.StepResult, error) {
	start := r.clock.Now()

	// HARD STOP: budget step gate
	if err := r.budget.AllowStep(step, start); err != nil {
		tr := appendBudgetError(buildDryRunTranscript(cfg, step), err)
		return types.StepResult{
			Step:       step,
			Outcome:    types.OutcomeError,
			Transcript: limitTranscript(tr, transcriptMaxBytes),
			Duration:   r.clock.Now().Sub(start),
			Output:     err.Error(),
		}, err
	}

	// HARD STOP: policy gate
	if err := r.policy.AllowStep(cfg, step); err != nil {
		var tr string
		if cfg.DryRun {
			tr = appendPolicyError(buildDryRunTranscript(cfg, step), err)
		} else {
			switch step.Type {
			case types.ToolShell:
				tr = appendPolicyError(buildShellStartTranscript(cfg, step), err)
			case types.ToolLLM:
				tr = appendPolicyError(buildLLMStartTranscript(step.Prompt), err)
			case types.ToolFiles:
				tr = appendPolicyError(buildFileStartTranscript(step), err)
			default:
				tr = appendPolicyError(buildUnsupportedStepTranscript(step), err)
			}
		}

		return types.StepResult{
			Step:       step,
			Outcome:    types.OutcomeError,
			Transcript: limitTranscript(tr, transcriptMaxBytes),
			Duration:   r.clock.Now().Sub(start),
			Output:     err.Error(),
		}, err
	}

	if cfg.DryRun {
		tr := buildDryRunTranscript(cfg, step)
		return types.StepResult{
			Step:       step,
			Outcome:    types.OutcomeDryRun,
			Transcript: limitTranscript(tr, transcriptMaxBytes),
			Duration:   r.clock.Now().Sub(start),
		}, nil
	}

	switch step.Type {
	case types.ToolShell:
		// HARD STOP: tool budget gate
		if err := r.budget.AllowTool(types.ToolShell, start); err != nil {
			tr := appendBudgetError(buildShellStartTranscript(cfg, step), err)
			return types.StepResult{
				Step:       step,
				Outcome:    types.OutcomeError,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
				Output:     err.Error(),
			}, err
		}

		res, err := r.tools.Shell.Run(ctx, cfg.WorkDir, step.Command, step.Args...)
		if err != nil {
			// SOFT FAIL: let agent recover
			tr := buildShellStartTranscript(cfg, step)
			return softStepError(r, start, step, tr, err), nil
		}

		outcome := types.OutcomeOK
		if res.ExitCode != 0 {
			outcome = types.OutcomeError // already soft: err is nil
		}

		tr := buildShellTranscript(cfg, step, res)

		out := res.Stdout
		if strings.TrimSpace(out) == "" {
			out = res.Stderr
		}

		return types.StepResult{
			Step:       step,
			Outcome:    outcome,
			Exec:       &res,
			Output:     out,
			Transcript: limitTranscript(tr, transcriptMaxBytes),
			Duration:   r.clock.Now().Sub(start),
			Effects: []types.StepEffect{
				effect("shell.exec", "", 0, map[string]any{
					"cmd":      step.Command,
					"args":     step.Args,
					"workdir":  cfg.WorkDir,
					"exitCode": res.ExitCode,
				}),
			},
		}, nil

	case types.ToolLLM:
		// SOFT FAIL: agent can correct prompt
		if strings.TrimSpace(step.Prompt) == "" {
			err := fmt.Errorf("llm step requires Prompt")
			tr := buildLLMStartTranscript(step.Prompt)
			return softStepError(r, start, step, tr, err), nil
		}

		// HARD STOP: token budget preflight
		snap := r.budget.Snapshot(start)
		if snap.Limits.MaxLLMTokens > 0 && snap.LLMTokensUsed >= snap.Limits.MaxLLMTokens {
			err := BudgetExceededError{
				Kind:    BudgetKindLLMTokens,
				Limit:   snap.Limits.MaxLLMTokens,
				Used:    snap.LLMTokensUsed,
				Message: "llm token budget exceeded",
			}
			tr := appendBudgetError(buildLLMStartTranscript(step.Prompt), err)
			return types.StepResult{
				Step:       step,
				Outcome:    types.OutcomeError,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
				Output:     err.Error(),
			}, err
		}

		// HARD STOP: tool budget gate
		if err := r.budget.AllowTool(types.ToolLLM, start); err != nil {
			tr := appendBudgetError(buildLLMStartTranscript(step.Prompt), err)
			return types.StepResult{
				Step:       step,
				Outcome:    types.OutcomeError,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
				Output:     err.Error(),
			}, err
		}

		out, tokens, err := r.tools.LLM.Complete(ctx, step.Prompt)
		if err != nil {
			// SOFT FAIL: agent can retry / simplify / etc
			tr := buildLLMStartTranscript(step.Prompt)
			return softStepError(r, start, step, tr, err), nil
		}

		r.budget.ChargeLLMTokens(tokens, start)

		tr := buildLLMTranscript(step.Prompt, out)
		return types.StepResult{
			Step:       step,
			Outcome:    types.OutcomeOK,
			Output:     out,
			Transcript: limitTranscript(tr, transcriptMaxBytes),
			Duration:   r.clock.Now().Sub(start),
		}, nil

	case types.ToolFiles:
		// SOFT FAIL: agent can correct op/path/data
		if strings.TrimSpace(step.Op) == "" {
			err := fmt.Errorf("file step requires Op")
			tr := buildFileStartTranscript(step)
			return softStepError(r, start, step, tr, err), nil
		}
		if strings.TrimSpace(step.Path) == "" {
			err := fmt.Errorf("file step requires Path")
			tr := buildFileStartTranscript(step)
			return softStepError(r, start, step, tr, err), nil
		}

		// HARD STOP: tool budget gate
		if err := r.budget.AllowTool(types.ToolFiles, start); err != nil {
			tr := appendBudgetError(buildFileStartTranscript(step), err)
			return types.StepResult{
				Step:       step,
				Outcome:    types.OutcomeError,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
				Output:     err.Error(),
			}, err
		}

		switch strings.ToLower(strings.TrimSpace(step.Op)) {
		case "read":
			b, err := r.tools.Files.ReadFile(step.Path)
			if err != nil {
				// SOFT FAIL
				tr := buildFileStartTranscript(step)
				return softStepError(r, start, step, tr, err), nil
			}

			out := string(b)
			tr := buildFileReadTranscript(step.Path, out)
			return types.StepResult{
				Step:       step,
				Outcome:    types.OutcomeOK,
				Output:     out,
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
			}, nil

		case "write":
			if step.Data == "" {
				err := fmt.Errorf("file write requires Data")
				tr := buildFileStartTranscript(step)
				return softStepError(r, start, step, tr, err), nil
			}

			if err := r.tools.Files.WriteFile(step.Path, []byte(step.Data)); err != nil {
				tr := buildFileStartTranscript(step)
				return softStepError(r, start, step, tr, err), nil
			}

			tr := buildFileWriteTranscript(step.Path, step.Data)
			return types.StepResult{
				Step:       step,
				Outcome:    types.OutcomeOK,
				Output:     fmt.Sprintf("wrote %d bytes to %s", len(step.Data), step.Path),
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
				Effects: []types.StepEffect{
					effect("file.write", step.Path, len(step.Data), nil),
				},
			}, nil

		case "patch":
			if step.Data == "" {
				err := fmt.Errorf("file patch requires Data (unified diff)")
				tr := buildFileStartTranscript(step)
				return softStepError(r, start, step, tr, err), nil
			}

			patchRes, err := r.tools.Files.PatchFile(step.Path, []byte(step.Data))
			if err != nil {
				tr := buildFilePatchTranscript(step.Path, patchRes, err)
				return softStepError(r, start, step, tr, err), nil
			}

			tr := buildFilePatchTranscript(step.Path, patchRes, nil)
			return types.StepResult{
				Step:       step,
				Outcome:    types.OutcomeOK,
				Output:     fmt.Sprintf("patched %s (hunks=%d)", step.Path, patchRes.Hunks),
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
				Effects: []types.StepEffect{
					effect("file.patch", step.Path, 0, map[string]any{
						"hunks": patchRes.Hunks,
					}),
				},
			}, nil

		case "replace":
			if step.Old == "" {
				err := fmt.Errorf("file replace requires Old")
				tr := buildFileStartTranscript(step)
				return softStepError(r, start, step, tr, err), nil
			}

			replRes, err := r.tools.Files.ReplaceBytesInFile(step.Path, []byte(step.Old), []byte(step.New), step.N)
			if err != nil {
				tr := buildFileReplaceTranscript(step.Path, step.N, replRes, err)
				return softStepError(r, start, step, tr, err), nil
			}

			tr := buildFileReplaceTranscript(step.Path, step.N, replRes, nil)
			return types.StepResult{
				Step:    step,
				Outcome: types.OutcomeOK,
				Output: fmt.Sprintf(
					"replaced %d occurrence(s) in %s (found=%d)",
					replRes.Replaced,
					step.Path,
					replRes.OccurrencesFound,
				),
				Transcript: limitTranscript(tr, transcriptMaxBytes),
				Duration:   r.clock.Now().Sub(start),
				Effects: []types.StepEffect{
					effect("file.replace", step.Path, 0, map[string]any{
						"found":    replRes.OccurrencesFound,
						"replaced": replRes.Replaced,
						"n":        step.N,
					}),
				},
			}, nil

		default:
			// SOFT FAIL: agent can correct op
			err := fmt.Errorf("unsupported file op: %s", step.Op)
			tr := buildFileStartTranscript(step)
			return softStepError(r, start, step, tr, err), nil
		}

	default:
		// SOFT FAIL: agent can correct tool type
		err := fmt.Errorf("unsupported step type: %s", step.Type)
		tr := buildUnsupportedStepTranscript(step)
		return softStepError(r, start, step, tr, err), nil
	}
}

func softStepError(r *DefaultRunner, start time.Time, step types.Step, tr string, err error) types.StepResult {
	// Keep transcript readable; include error in Output so agent sees it in OBSERVATION.
	if tr != "" && !strings.HasSuffix(tr, "\n") {
		tr += "\n"
	}
	tr += fmt.Sprintf("[error] %v\n", err)

	return types.StepResult{
		Step:       step,
		Outcome:    types.OutcomeError,
		Output:     err.Error(),
		Transcript: limitTranscript(tr, transcriptMaxBytes),
		Duration:   r.clock.Now().Sub(start),
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

func buildDryRunTranscript(cfg types.Config, step types.Step) string {
	switch step.Type {
	case types.ToolShell:
		return fmt.Sprintf("[dry-run][shell] workdir=%q cmd=%q args=%v\n", cfg.WorkDir, step.Command, step.Args)

	case types.ToolLLM:
		return fmt.Sprintf("[dry-run][llm]\n%s\n", step.Prompt)

	case types.ToolFiles:
		op := strings.ToLower(strings.TrimSpace(step.Op))

		switch op {
		case "replace":
			return fmt.Sprintf(
				"[dry-run][file] op=%q path=%q old_len=%d new_len=%d n=%d\n",
				step.Op, step.Path, len(step.Old), len(step.New), step.N,
			)

		case "patch":
			return fmt.Sprintf(
				"[dry-run][file] op=%q path=%q diff_len=%d\n",
				step.Op, step.Path, len(step.Data),
			)

		case "write":
			return fmt.Sprintf(
				"[dry-run][file] op=%q path=%q data_len=%d\n",
				step.Op, step.Path, len(step.Data),
			)

		case "read":
			return fmt.Sprintf(
				"[dry-run][file] op=%q path=%q\n",
				step.Op, step.Path,
			)

		default:
			// Unknown op, keep current behavior as a safe fallback.
			return fmt.Sprintf("[dry-run][file] op=%q path=%q data_len=%d\n", step.Op, step.Path, len(step.Data))
		}

	default:
		return fmt.Sprintf("[dry-run] step_type=%q\n", step.Type)
	}
}

var firstMismatchLineRe = regexp.MustCompile(`\bline\s+(\d+)\b`)

func buildFilePatchTranscript(path string, res tools.PatchResult, err error) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "[file] op=%q path=%q\n", "patch", path)
	_, _ = fmt.Fprintf(&b, "hunks=%d\n", res.Hunks)

	if err != nil {
		_, _ = fmt.Fprintf(&b, "error=%q\n", err.Error())

		if m := firstMismatchLineRe.FindStringSubmatch(err.Error()); len(m) == 2 {
			_, _ = fmt.Fprintf(&b, "first_mismatch_line=%s\n", m[1])
		}
	}

	return b.String()
}

func buildFileReplaceTranscript(path string, n int, res tools.ReplaceResult, err error) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "[file] op=%q path=%q\n", "replace", path)
	_, _ = fmt.Fprintf(&b, "n=%d\n", n)
	_, _ = fmt.Fprintf(&b, "occurrences_found=%d\n", res.OccurrencesFound)
	_, _ = fmt.Fprintf(&b, "replaced=%d\n", res.Replaced)

	if err != nil {
		_, _ = fmt.Fprintf(&b, "error=%q\n", err.Error())
	}
	return b.String()
}

func buildShellStartTranscript(cfg types.Config, step types.Step) string {
	return fmt.Sprintf(
		"[shell:start] workdir=%q cmd=%q args=%v\n",
		cfg.WorkDir,
		step.Command,
		step.Args,
	)
}

func buildShellTranscript(cfg types.Config, step types.Step, res types.Result) string {
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

func buildFileStartTranscript(step types.Step) string {
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

func buildUnsupportedStepTranscript(step types.Step) string {
	return fmt.Sprintf("[unsupported] step_type=%q\n", step.Type)
}

func limitTranscript(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "\n…(truncated)\n"
}

func effect(kind, path string, bytes int, meta map[string]any) types.StepEffect {
	return types.StepEffect{
		Kind:  kind,
		Path:  path,
		Bytes: bytes,
		Meta:  meta,
	}
}
