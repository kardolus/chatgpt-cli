package agent

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

//go:generate mockgen -destination=shellmocks_test.go -package=agent_test github.com/kardolus/chatgpt-cli/agent Shell
type Shell interface {
	Run(
		ctx context.Context,
		workDir string,
		name string,
		args ...string,
	) (Result, error)
}

type ExecShellRunner struct{}

func NewExecShellRunner() *ExecShellRunner {
	return &ExecShellRunner{}
}

func (r *ExecShellRunner) Run(
	ctx context.Context,
	workDir string,
	name string,
	args ...string,
) (Result, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir

	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err := cmd.Run()

	exit := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exit = ee.ExitCode()
		} else {
			return Result{}, err
		}
	}

	return Result{
		Stdout:   outb.String(),
		Stderr:   errb.String(),
		ExitCode: exit,
		Duration: time.Since(start),
	}, nil
}
