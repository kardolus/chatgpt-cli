package tools

import (
	"bytes"
	"context"
	"errors"
	"github.com/kardolus/chatgpt-cli/agent/types"
	"os/exec"
	"time"
)

type Shell interface {
	Run(
		ctx context.Context,
		workDir string,
		name string,
		args ...string,
	) (types.Result, error)
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
) (types.Result, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir

	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err := cmd.Run()

	exit := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exit = ee.ExitCode()
		}
	}

	return types.Result{
		Stdout:   outb.String(),
		Stderr:   errb.String(),
		ExitCode: exit,
		Duration: time.Since(start),
	}, nil
}
