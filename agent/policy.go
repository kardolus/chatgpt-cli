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

		if len(p.limits.AllowedFileOps) > 0 && !containsString(p.limits.AllowedFileOps, op) {
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
	wd := filepath.Clean(workdir)

	// If path is absolute, clean it. If relative, join to workdir.
	var full string
	if filepath.IsAbs(path) {
		full = filepath.Clean(path)
	} else {
		full = filepath.Clean(filepath.Join(wd, path))
	}

	// Make sure full starts with wd + separator, or equals wd.
	if full == wd {
		return false
	}
	prefix := wd + string(filepath.Separator)
	return !strings.HasPrefix(full, prefix)
}
