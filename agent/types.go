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
	Data string
}

type ExecContext struct {
	Goal    string
	Plan    Plan
	Results []StepResult
}

type StepResult struct {
	Step       Step
	Outcome    OutcomeKind
	Transcript string // human-readable narrative/log of what happened
	Duration   time.Duration
	Exec       *Result
	Output     string // semantic result intended for downstream steps
}

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}
