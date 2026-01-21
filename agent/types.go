package agent

import (
	"time"
)

type ToolKind string

const (
	ToolShell ToolKind = "shell"
	ToolLLM   ToolKind = "llm"
	ToolFiles ToolKind = "file"
)

type Config struct {
	MaxSteps int
	DryRun   bool
	WorkDir  string
}

type Plan struct {
	Goal  string
	Steps []Step
}

type Step struct {
	Type ToolKind

	Description string

	// Shell
	Command string
	Args    []string

	// LLM
	Prompt string

	// Files
	Path string
	Op   string

	// For write/patch: Data is the full content or unified diff
	Data string

	// For replace:
	Old string
	New string
	N   int
}

type ExecContext struct {
	Goal    string
	Plan    Plan
	Results []StepResult
}

type StepResult struct {
	Step       Step
	Outcome    OutcomeKind
	Transcript string
	Duration   time.Duration
	Exec       *Result
	Output     string
	Effects    Effects
}

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}
