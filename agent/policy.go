package agent

import (
	"fmt"
	"path/filepath"
	"strings"
)

//go:generate mockgen -destination=policymocks_test.go -package=agent_test github.com/kardolus/chatgpt-cli/agent Policy
type Policy interface {
	AllowStep(cfg Config, step Step) error
}

const (
	PolicyKindStepType   = "step_type"
	PolicyKindShell      = "shell"
	PolicyKindLLM        = "llm"
	PolicyKindFiles      = "files"
	PolicyKindPathEscape = "path_escape"
)

type DefaultPolicy struct {
	limits PolicyLimits
}

type PolicyLimits struct {
	AllowedTools           []ToolKind
	RestrictFilesToWorkDir bool
	DeniedShellCommands    []string
	AllowedFileOps         []string
}

func NewDefaultPolicy(limits PolicyLimits) *DefaultPolicy {
	return &DefaultPolicy{limits: limits}
}

func (p *DefaultPolicy) AllowStep(cfg Config, step Step) error {
	switch step.Type {
	case ToolShell, ToolLLM, ToolFiles:
		// ok
	default:
		return PolicyDeniedError{
			Kind:   PolicyKindStepType,
			Reason: fmt.Sprintf("unsupported step type: %s", step.Type),
		}
	}

	if len(p.limits.AllowedTools) > 0 && !containsTool(p.limits.AllowedTools, step.Type) {
		return PolicyDeniedError{
			Kind:   PolicyKindStepType,
			Reason: fmt.Sprintf("tool not allowed: %s", step.Type),
		}
	}

	switch step.Type {
	case ToolShell:
		cmd := strings.TrimSpace(step.Command)
		if cmd == "" {
			return PolicyDeniedError{Kind: PolicyKindShell, Reason: "shell step requires Command"}
		}
		if len(p.limits.DeniedShellCommands) > 0 && containsString(p.limits.DeniedShellCommands, cmd) {
			return PolicyDeniedError{Kind: PolicyKindShell, Reason: fmt.Sprintf("shell command denied: %s", cmd)}
		}

		if p.limits.RestrictFilesToWorkDir && cfg.WorkDir != "" {
			if err := denyShellArgsOutsideWorkDir(cfg.WorkDir, step.Args); err != nil {
				return err
			}
		}

	case ToolLLM:
		if strings.TrimSpace(step.Prompt) == "" {
			return PolicyDeniedError{Kind: PolicyKindLLM, Reason: "llm step requires Prompt"}
		}

	case ToolFiles:
		op := strings.ToLower(strings.TrimSpace(step.Op))
		if op == "" {
			return PolicyDeniedError{Kind: PolicyKindFiles, Reason: "file step requires Op"}
		}
		if strings.TrimSpace(step.Path) == "" {
			return PolicyDeniedError{Kind: PolicyKindFiles, Reason: "file step requires Path"}
		}

		switch op {
		case "patch":
			if strings.TrimSpace(step.Data) == "" {
				return PolicyDeniedError{
					Kind:   PolicyKindFiles,
					Reason: "patch requires Data (unified diff)",
				}
			}
		case "replace":
			if len(step.Old) == 0 {
				return PolicyDeniedError{
					Kind:   PolicyKindFiles,
					Reason: "replace requires Old pattern",
				}
			}
		}

		if !fileOpAllowed(p.limits.AllowedFileOps, op) {
			return PolicyDeniedError{Kind: PolicyKindFiles, Reason: fmt.Sprintf("file op not allowed: %s", op)}
		}

		if p.limits.RestrictFilesToWorkDir && cfg.WorkDir != "" {
			if escapesWorkDir(cfg.WorkDir, step.Path) {
				return PolicyDeniedError{
					Kind:   PolicyKindPathEscape,
					Reason: fmt.Sprintf("path escapes workdir: workdir=%q path=%q", cfg.WorkDir, step.Path),
				}
			}
		}
	}

	return nil
}

// denyShellArgsOutsideWorkDir blocks absolute paths, ~, .., and any arg that would escape workdir.
func denyShellArgsOutsideWorkDir(workdir string, args []string) error {
	for _, raw := range args {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}

		// Hard block obvious escapes
		if strings.HasPrefix(s, "~") {
			return PolicyDeniedError{
				Kind:   PolicyKindPathEscape,
				Reason: fmt.Sprintf("shell arg escapes workdir: workdir=%q arg=%q", workdir, s),
			}
		}

		// If it contains path separators or looks like a path, validate it.
		// (This is intentionally conservative; better to block than allow rm /tmp.)
		if strings.Contains(s, "/") || strings.Contains(s, `\`) || strings.Contains(s, "..") || filepath.IsAbs(s) {
			if escapesWorkDir(workdir, s) {
				return PolicyDeniedError{
					Kind:   PolicyKindPathEscape,
					Reason: fmt.Sprintf("shell arg escapes workdir: workdir=%q arg=%q", workdir, s),
				}
			}
		}
	}
	return nil
}

// PolicyDeniedError is a typed error so Agent/Planner can branch on it.
type PolicyDeniedError struct {
	Kind   string
	Reason string
}

func (e PolicyDeniedError) Error() string {
	return fmt.Sprintf("policy denied: kind=%s reason=%s", e.Kind, e.Reason)
}

func containsTool(xs []ToolKind, k ToolKind) bool {
	for _, x := range xs {
		if x == k {
			return true
		}
	}
	return false
}

func containsString(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// escapesWorkDir returns true if path, when resolved relative to workdir, is outside workdir.
func escapesWorkDir(workdir, path string) bool {
	// Resolve workdir to absolute path
	wd, err := filepath.Abs(workdir)
	if err != nil {
		return true // fail closed
	}
	wd = filepath.Clean(wd)

	// Resolve target path
	var full string
	if filepath.IsAbs(path) {
		full = path
	} else {
		full = filepath.Join(wd, path)
	}
	full, err = filepath.Abs(full)
	if err != nil {
		return true // fail closed
	}
	full = filepath.Clean(full)

	// Allow exactly wd or anything under wd
	if full == wd {
		return false
	}
	prefix := wd + string(filepath.Separator)
	return !strings.HasPrefix(full, prefix)
}

func fileOpAllowed(allowed []string, op string) bool {
	if len(allowed) == 0 {
		return true
	}
	if containsString(allowed, op) {
		return true
	}
	// If you can write arbitrary bytes, you can also patch/replace.
	if (op == "patch" || op == "replace") && containsString(allowed, "write") {
		return true
	}
	return false
}
