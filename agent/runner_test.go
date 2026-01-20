package agent_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/kardolus/chatgpt-cli/agent"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitRunner(t *testing.T) {
	spec.Run(t, "Testing the runner", testRunner, spec.Report(report.Terminal{}))
}

func testRunner(t *testing.T, when spec.G, it spec.S) {
	var (
		mockCtrl   *gomock.Controller
		mockClock  *MockClock
		mockShell  *MockShell
		mockLLM    *MockLLM
		mockFiles  *MockFiles
		mockBudget *MockBudget
		mockPolicy *MockPolicy

		tools   agent.Tools
		subject *agent.DefaultRunner
	)

	it.Before(func() {
		RegisterTestingT(t)

		mockCtrl = gomock.NewController(t)
		mockClock = NewMockClock(mockCtrl)
		mockShell = NewMockShell(mockCtrl)
		mockLLM = NewMockLLM(mockCtrl)
		mockFiles = NewMockFiles(mockCtrl)
		mockBudget = NewMockBudget(mockCtrl)
		mockPolicy = NewMockPolicy(mockCtrl)

		tools = agent.Tools{
			Shell: mockShell,
			LLM:   mockLLM,
			Files: mockFiles,
		}

		subject = agent.NewDefaultRunner(tools, mockClock, mockBudget, mockPolicy)
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("RunStep()", func() {
		it("returns dry-run result and does not invoke tools", func() {
			dur := expectDuration(mockClock, 123*time.Millisecond)

			cfg := agent.Config{DryRun: true, WorkDir: "/tmp"}
			step := agent.Step{
				Type:    agent.ToolShell,
				Command: "echo",
				Args:    []string{"hi"},
			}

			// Budget: count attempted step, but no tool call.
			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			mockBudget.EXPECT().AllowTool(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Outcome).To(Equal(agent.OutcomeDryRun))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Step).To(Equal(step))
			Expect(res.Transcript).To(ContainSubstring("[dry-run][shell]"))
			Expect(res.Transcript).To(ContainSubstring(`workdir="/tmp"`))
			Expect(res.Transcript).To(ContainSubstring(`cmd="echo"`))
			Expect(res.Exec).To(BeNil())
		})

		it("runs shell command and returns ok outcome when exit code is 0", func() {
			dur := expectDuration(mockClock, 123*time.Millisecond)

			cfg := agent.Config{DryRun: false, WorkDir: "/tmp"}
			step := agent.Step{
				Type:    agent.ToolShell,
				Command: "echo",
				Args:    []string{"hi"},
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			expectAllowTool(mockBudget, agent.ToolShell)

			exec := agent.Result{ExitCode: 0, Stdout: "hi\n"}
			mockShell.
				EXPECT().
				Run(gomock.Any(), cfg.WorkDir, "echo", "hi").
				Return(exec, nil).
				Times(1)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeOK))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Exec).NotTo(BeNil())
			Expect(*res.Exec).To(Equal(exec))

			expectShellTranscript(res, cfg, step, exec)
		})

		it("returns error outcome when shell exits non-zero", func() {
			dur := expectDuration(mockClock, 10*time.Millisecond)

			cfg := agent.Config{DryRun: false, WorkDir: "/repo"}
			step := agent.Step{
				Type:    agent.ToolShell,
				Command: "git",
				Args:    []string{"status", "--porcelain"},
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			expectAllowTool(mockBudget, agent.ToolShell)

			exec := agent.Result{ExitCode: 17, Stdout: " M file.go\n"}
			mockShell.
				EXPECT().
				Run(gomock.Any(), cfg.WorkDir, "git", "status", "--porcelain").
				Return(exec, nil).
				Times(1)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Exec).NotTo(BeNil())
			Expect(*res.Exec).To(Equal(exec))

			expectShellTranscript(res, cfg, step, exec)
		})

		it("returns OutcomeError (no error) when shell runner errors, and surfaces error in Output", func() {
			dur := expectDuration(mockClock, 5*time.Millisecond)

			cfg := agent.Config{DryRun: false, WorkDir: "/tmp"}
			step := agent.Step{
				Type:    agent.ToolShell,
				Command: "go",
				Args:    []string{"test", "./..."},
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			expectAllowTool(mockBudget, agent.ToolShell)

			runErr := errors.New("shell boom")
			mockShell.
				EXPECT().
				Run(gomock.Any(), cfg.WorkDir, "go", "test", "./...").
				Return(agent.Result{}, runErr).
				Times(1)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Exec).To(BeNil())
			Expect(res.Output).To(ContainSubstring(runErr.Error()))

			Expect(res.Transcript).To(ContainSubstring("[shell:start]"))
			Expect(res.Transcript).To(ContainSubstring(`workdir="/tmp"`))
			Expect(res.Transcript).To(ContainSubstring(`cmd="go"`))
		})

		it("handles shell command with no args (variadic call)", func() {
			dur := expectDuration(mockClock, 1*time.Millisecond)

			cfg := agent.Config{DryRun: false, WorkDir: "/tmp"}
			step := agent.Step{
				Type:    agent.ToolShell,
				Command: "pwd",
				Args:    nil,
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			expectAllowTool(mockBudget, agent.ToolShell)

			exec := agent.Result{ExitCode: 0, Stdout: "/tmp\n"}
			mockShell.
				EXPECT().
				Run(gomock.Any(), cfg.WorkDir, "pwd").
				Return(exec, nil).
				Times(1)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Outcome).To(Equal(agent.OutcomeOK))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Exec).NotTo(BeNil())
			Expect(*res.Exec).To(Equal(exec))
		})

		it("returns OutcomeError (no error) when llm prompt is missing/blank and does not invoke llm tool", func() {
			dur := expectDuration(mockClock, 123*time.Millisecond)

			cfg := agent.Config{DryRun: false, WorkDir: "/tmp"}
			step := agent.Step{
				Type:   agent.ToolLLM,
				Prompt: "   \n\t",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)

			// Guard: tool not called / budget tool not charged
			mockBudget.EXPECT().AllowTool(agent.ToolLLM, gomock.Any()).Times(0)
			mockLLM.EXPECT().Complete(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Output).To(ContainSubstring("llm step requires Prompt"))
			Expect(res.Exec).To(BeNil())

			expectLLMStartTranscript(res, step)
		})

		it("runs llm completion and returns ok outcome + output + transcript", func() {
			dur := expectDuration(mockClock, 123*time.Millisecond)

			cfg := agent.Config{DryRun: false}
			step := agent.Step{
				Type:   agent.ToolLLM,
				Prompt: "say hi",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)

			// NEW: token preflight
			expectLLMSnapshotOK(mockBudget)

			expectAllowTool(mockBudget, agent.ToolLLM)

			mockLLM.
				EXPECT().
				Complete(gomock.Any(), step.Prompt).
				Return("hi there", 12, nil).
				Times(1)

			// NEW: record token usage
			mockBudget.EXPECT().
				ChargeLLMTokens(12, gomock.Any()).
				Times(1)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeOK))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Output).To(Equal("hi there"))
			Expect(res.Exec).To(BeNil())

			expectLLMOKTranscript(res, step, "hi there")
		})

		it("returns OutcomeError (no error) when llm tool errors and surfaces error in Output", func() {
			dur := expectDuration(mockClock, 10*time.Millisecond)

			cfg := agent.Config{DryRun: false}
			step := agent.Step{
				Type:   agent.ToolLLM,
				Prompt: "do the thing",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)

			// token preflight
			expectLLMSnapshotOK(mockBudget)
			expectAllowTool(mockBudget, agent.ToolLLM)

			runErr := errors.New("llm boom")
			mockLLM.
				EXPECT().
				Complete(gomock.Any(), step.Prompt).
				Return("", 0, runErr).
				Times(1)

			// Typically don't charge tokens on error
			mockBudget.EXPECT().
				ChargeLLMTokens(gomock.Any(), gomock.Any()).
				Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Output).To(ContainSubstring(runErr.Error()))
			Expect(res.Exec).To(BeNil())

			expectLLMStartTranscript(res, step)
		})

		it("returns error StepResult when llm tool budget is denied and does not invoke llm tool", func() {
			dur := expectDuration(mockClock, 5*time.Millisecond)

			cfg := agent.Config{DryRun: false}
			step := agent.Step{
				Type:   agent.ToolLLM,
				Prompt: "say hi",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)

			// token preflight must happen before AllowTool
			expectLLMSnapshotOK(mockBudget)

			toolErr := errors.New("tool budget denied")
			mockBudget.EXPECT().AllowTool(agent.ToolLLM, gomock.Any()).Return(toolErr).Times(1)

			// Guard: LLM must not run
			mockLLM.EXPECT().Complete(gomock.Any(), gomock.Any()).Times(0)

			// And no token charging
			mockBudget.EXPECT().ChargeLLMTokens(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).To(MatchError(toolErr))

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))

			Expect(res.Transcript).To(ContainSubstring("[llm:start]"))
			Expect(res.Transcript).To(ContainSubstring("[budget]"))
			Expect(res.Transcript).To(ContainSubstring(toolErr.Error()))
		})

		it("returns OutcomeError (no error) when file path is missing/blank and does not invoke file tool", func() {
			dur := expectDuration(mockClock, 5*time.Millisecond)

			cfg := agent.Config{DryRun: false, WorkDir: "/tmp"}
			step := agent.Step{
				Type: agent.ToolFiles,
				Op:   "read",
				Path: "   ",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			mockBudget.EXPECT().AllowTool(agent.ToolFiles, gomock.Any()).Times(0)

			mockFiles.EXPECT().ReadFile(gomock.Any()).Times(0)
			mockFiles.EXPECT().WriteFile(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Exec).To(BeNil())
			Expect(res.Output).To(ContainSubstring("file step requires Path"))

			expectFileStartTranscript(res, step)
		})

		it("returns OutcomeError (no error) for unsupported file op and does not invoke file tool", func() {
			dur := expectDuration(mockClock, 5*time.Millisecond)

			cfg := agent.Config{DryRun: false, WorkDir: "/tmp"}
			step := agent.Step{
				Type: agent.ToolFiles,
				Op:   "delete",
				Path: "/tmp/a.txt",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)

			// NOTE: current runner charges tool budget before op switch.
			mockBudget.EXPECT().AllowTool(agent.ToolFiles, gomock.Any()).Times(1)

			mockFiles.EXPECT().ReadFile(gomock.Any()).Times(0)
			mockFiles.EXPECT().WriteFile(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Exec).To(BeNil())
			Expect(res.Output).To(ContainSubstring("unsupported file op"))

			expectFileStartTranscript(res, step)
		})

		it("reads file and returns ok outcome + Output", func() {
			dur := expectDuration(mockClock, 123*time.Millisecond)

			cfg := agent.Config{DryRun: false, WorkDir: "/tmp"}
			step := agent.Step{
				Type: agent.ToolFiles,
				Op:   "read",
				Path: "/tmp/a.txt",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			expectAllowTool(mockBudget, agent.ToolFiles)

			mockFiles.
				EXPECT().
				ReadFile(step.Path).
				Return([]byte("hello\n"), nil).
				Times(1)

			mockFiles.EXPECT().WriteFile(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeOK))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Exec).To(BeNil())
			Expect(res.Output).To(Equal("hello\n"))

			expectFileReadTranscript(res, step.Path, "hello\n")
		})

		it("returns OutcomeError (no error) when read errors, and surfaces error in Output (with start transcript)", func() {
			dur := expectDuration(mockClock, 10*time.Millisecond)

			cfg := agent.Config{DryRun: false, WorkDir: "/tmp"}
			step := agent.Step{
				Type: agent.ToolFiles,
				Op:   "read",
				Path: "/tmp/missing.txt",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			expectAllowTool(mockBudget, agent.ToolFiles)

			readErr := errors.New("read boom")
			mockFiles.
				EXPECT().
				ReadFile(step.Path).
				Return(nil, readErr).
				Times(1)

			mockFiles.EXPECT().WriteFile(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Exec).To(BeNil())
			Expect(res.Output).To(ContainSubstring(readErr.Error()))

			expectFileStartTranscript(res, step)
		})

		it("writes file and returns ok outcome", func() {
			dur := expectDuration(mockClock, 50*time.Millisecond)

			cfg := agent.Config{DryRun: false, WorkDir: "/tmp"}
			step := agent.Step{
				Type: agent.ToolFiles,
				Op:   "write",
				Path: "/tmp/out.txt",
				Data: "payload",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			expectAllowTool(mockBudget, agent.ToolFiles)

			mockFiles.
				EXPECT().
				WriteFile(step.Path, []byte(step.Data)).
				Return(nil).
				Times(1)

			mockFiles.EXPECT().ReadFile(gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeOK))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Exec).To(BeNil())
			Expect(res.Output).To(ContainSubstring("/tmp/out.txt"))

			expectFileWriteTranscript(res, step.Path, len(step.Data))
		})

		it("returns OutcomeError (no error) when write errors, and surfaces error in Output (with start transcript)", func() {
			dur := expectDuration(mockClock, 50*time.Millisecond)

			cfg := agent.Config{DryRun: false, WorkDir: "/tmp"}
			step := agent.Step{
				Type: agent.ToolFiles,
				Op:   "write",
				Path: "/tmp/out.txt",
				Data: "payload",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			expectAllowTool(mockBudget, agent.ToolFiles)

			writeErr := errors.New("write boom")
			mockFiles.
				EXPECT().
				WriteFile(step.Path, []byte(step.Data)).
				Return(writeErr).
				Times(1)

			mockFiles.EXPECT().ReadFile(gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Exec).To(BeNil())
			Expect(res.Output).To(ContainSubstring(writeErr.Error()))

			expectFileStartTranscript(res, step)
		})

		it("treats file op case/whitespace-insensitively (READ)", func() {
			dur := expectDuration(mockClock, 20*time.Millisecond)

			cfg := agent.Config{DryRun: false, WorkDir: "/tmp"}
			step := agent.Step{
				Type: agent.ToolFiles,
				Op:   "  ReAd  ",
				Path: "/tmp/a.txt",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			expectAllowTool(mockBudget, agent.ToolFiles)

			mockFiles.
				EXPECT().
				ReadFile(step.Path).
				Return([]byte("ok"), nil).
				Times(1)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Outcome).To(Equal(agent.OutcomeOK))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Output).To(Equal("ok"))
		})

		it("returns OutcomeError (no error) when step type is unsupported (with transcript)", func() {
			dur := expectDuration(mockClock, 7*time.Millisecond)

			cfg := agent.Config{DryRun: false, WorkDir: "/tmp"}
			step := agent.Step{
				Type:        agent.ToolKind("wat"),
				Description: "unknown step",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			// Guard: no tool should be charged
			mockBudget.EXPECT().AllowTool(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Exec).To(BeNil())
			Expect(res.Output).To(ContainSubstring("unsupported step type: wat"))
			Expect(res.Transcript).To(ContainSubstring(`[unsupported]`))
			Expect(res.Transcript).To(ContainSubstring(`step_type="wat"`))
		})

		it("returns error StepResult when step budget is denied (applies to dry-run too) and does not invoke tools", func() {
			dur := expectDuration(mockClock, 5*time.Millisecond)

			cfg := agent.Config{DryRun: true, WorkDir: "/tmp"}
			step := agent.Step{
				Type:    agent.ToolShell,
				Command: "echo",
				Args:    []string{"hi"},
			}

			stepErr := errors.New("step budget denied")
			mockBudget.EXPECT().AllowStep(step, gomock.Any()).Return(stepErr).Times(1)

			// Guard: no tool budget charge and no tool execution
			mockBudget.EXPECT().AllowTool(gomock.Any(), gomock.Any()).Times(0)
			mockShell.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			mockLLM.EXPECT().Complete(gomock.Any(), gomock.Any()).Times(0)
			mockFiles.EXPECT().ReadFile(gomock.Any()).Times(0)
			mockFiles.EXPECT().WriteFile(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).To(MatchError(stepErr))

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))

			// Budget error is appended on top of the dry-run transcript
			Expect(res.Transcript).To(ContainSubstring("[dry-run][shell]"))
			Expect(res.Transcript).To(ContainSubstring("[budget]"))
			Expect(res.Transcript).To(ContainSubstring(stepErr.Error()))
		})

		it("returns error StepResult when shell tool budget is denied and does not invoke shell tool", func() {
			dur := expectDuration(mockClock, 5*time.Millisecond)

			cfg := agent.Config{DryRun: false, WorkDir: "/tmp"}
			step := agent.Step{
				Type:    agent.ToolShell,
				Command: "echo",
				Args:    []string{"hi"},
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)

			toolErr := errors.New("tool budget denied")
			mockBudget.EXPECT().AllowTool(agent.ToolShell, gomock.Any()).Return(toolErr).Times(1)

			// Guard: shell must not run
			mockShell.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).To(MatchError(toolErr))

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))

			Expect(res.Transcript).To(ContainSubstring("[shell:start]"))
			Expect(res.Transcript).To(ContainSubstring("[budget]"))
			Expect(res.Transcript).To(ContainSubstring(toolErr.Error()))
		})

		it("returns error StepResult when llm tool budget is denied and does not invoke llm tool", func() {
			dur := expectDuration(mockClock, 5*time.Millisecond)

			cfg := agent.Config{DryRun: false}
			step := agent.Step{
				Type:   agent.ToolLLM,
				Prompt: "say hi",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)

			mockBudget.
				EXPECT().
				Snapshot(gomock.Any()).
				Return(agent.BudgetSnapshot{
					Limits:        agent.BudgetLimits{MaxLLMTokens: 0},
					LLMTokensUsed: 0,
				}).
				Times(1)

			toolErr := errors.New("tool budget denied")
			mockBudget.EXPECT().AllowTool(agent.ToolLLM, gomock.Any()).Return(toolErr).Times(1)

			mockLLM.EXPECT().Complete(gomock.Any(), gomock.Any()).Times(0)
			mockBudget.EXPECT().ChargeLLMTokens(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).To(MatchError(toolErr))

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Transcript).To(ContainSubstring("[llm:start]"))
			Expect(res.Transcript).To(ContainSubstring("[budget]"))
			Expect(res.Transcript).To(ContainSubstring(toolErr.Error()))
		})

		it("returns error StepResult when llm token budget preflight fails and does not invoke llm tool or charge tool budget", func() {
			dur := expectDuration(mockClock, 5*time.Millisecond)

			cfg := agent.Config{DryRun: false}
			step := agent.Step{
				Type:   agent.ToolLLM,
				Prompt: "say hi",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)

			// Preflight says we're already out of tokens
			mockBudget.
				EXPECT().
				Snapshot(gomock.Any()).
				Return(agent.BudgetSnapshot{
					Limits:        agent.BudgetLimits{MaxLLMTokens: 100},
					LLMTokensUsed: 100,
				}).
				Times(1)

			// Guard: should bail before tool budget is charged
			mockBudget.EXPECT().AllowTool(agent.ToolLLM, gomock.Any()).Times(0)

			// Guard: LLM must not run
			mockLLM.EXPECT().Complete(gomock.Any(), gomock.Any()).Times(0)

			// Guard: no token charging
			mockBudget.EXPECT().ChargeLLMTokens(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("llm token budget exceeded"))

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Transcript).To(ContainSubstring("[llm:start]"))
			Expect(res.Transcript).To(ContainSubstring("[budget]"))
			Expect(res.Transcript).To(ContainSubstring("token"))
		})

		it("returns OutcomeError when policy denies a dry-run step (no tools invoked)", func() {
			dur := expectDuration(mockClock, 5*time.Millisecond)
			cfg := agent.Config{DryRun: true, WorkDir: "/tmp"}
			step := agent.Step{Type: agent.ToolShell, Command: "echo", Args: []string{"hi"}}

			expectAllowStep(mockBudget, step)

			polErr := errors.New("policy denied")
			mockPolicy.EXPECT().AllowStep(cfg, step).Return(polErr).Times(1)

			mockBudget.EXPECT().AllowTool(gomock.Any(), gomock.Any()).Times(0)
			mockShell.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).To(MatchError(polErr))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Transcript).To(ContainSubstring("[dry-run][shell]"))
			Expect(res.Transcript).To(ContainSubstring("[policy]"))
			Expect(res.Transcript).To(ContainSubstring(polErr.Error()))
		})

		it("returns OutcomeError when policy denies a shell step and does not charge tool budget or run shell", func() {
			dur := expectDuration(mockClock, 5*time.Millisecond)
			cfg := agent.Config{DryRun: false, WorkDir: "/tmp"}
			step := agent.Step{Type: agent.ToolShell, Command: "echo", Args: []string{"hi"}}

			expectAllowStep(mockBudget, step)

			polErr := errors.New("policy denied")
			mockPolicy.EXPECT().AllowStep(cfg, step).Return(polErr).Times(1)

			mockBudget.EXPECT().AllowTool(agent.ToolShell, gomock.Any()).Times(0)
			mockShell.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).To(MatchError(polErr))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Transcript).To(ContainSubstring("[shell:start]"))
			Expect(res.Transcript).To(ContainSubstring("[policy]"))
		})

		it("policy denial short-circuits llm: no token snapshot, no tool budget, no llm call", func() {
			dur := expectDuration(mockClock, 5*time.Millisecond)
			cfg := agent.Config{DryRun: false}
			step := agent.Step{Type: agent.ToolLLM, Prompt: "say hi"}

			expectAllowStep(mockBudget, step)

			polErr := errors.New("policy denied")
			mockPolicy.EXPECT().AllowStep(cfg, step).Return(polErr).Times(1)

			mockBudget.EXPECT().Snapshot(gomock.Any()).Times(0)
			mockBudget.EXPECT().AllowTool(agent.ToolLLM, gomock.Any()).Times(0)
			mockLLM.EXPECT().Complete(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).To(MatchError(polErr))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Transcript).To(ContainSubstring("[llm:start]"))
			Expect(res.Transcript).To(ContainSubstring("[policy]"))
		})

		it("returns OutcomeError when policy denies a file step and does not charge tool budget or touch filesystem", func() {
			dur := expectDuration(mockClock, 5*time.Millisecond)
			cfg := agent.Config{DryRun: false}
			step := agent.Step{Type: agent.ToolFiles, Op: "read", Path: "/tmp/a.txt"}

			expectAllowStep(mockBudget, step)

			polErr := errors.New("policy denied")
			mockPolicy.EXPECT().AllowStep(cfg, step).Return(polErr).Times(1)

			mockBudget.EXPECT().AllowTool(agent.ToolFiles, gomock.Any()).Times(0)
			mockFiles.EXPECT().ReadFile(gomock.Any()).Times(0)
			mockFiles.EXPECT().WriteFile(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).To(MatchError(polErr))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Transcript).To(ContainSubstring("[file:start]"))
			Expect(res.Transcript).To(ContainSubstring("[policy]"))
		})

		it("patches file and returns ok outcome (calls PatchFile)", func() {
			dur := expectDuration(mockClock, 50*time.Millisecond)

			cfg := agent.Config{DryRun: false}
			step := agent.Step{
				Type: agent.ToolFiles,
				Op:   "patch",
				Path: "/tmp/a.txt",
				Data: "@@ -1,1 +1,1 @@\n-a\n+b\n",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			expectAllowTool(mockBudget, agent.ToolFiles)

			mockFiles.
				EXPECT().
				PatchFile(step.Path, []byte(step.Data)).
				Return(agent.PatchResult{Hunks: 2}, nil).
				Times(1)

			// Guard: no other ops
			mockFiles.EXPECT().ReadFile(gomock.Any()).Times(0)
			mockFiles.EXPECT().WriteFile(gomock.Any(), gomock.Any()).Times(0)
			mockFiles.EXPECT().ReplaceBytesInFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeOK))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Exec).To(BeNil())

			Expect(res.Output).NotTo(BeEmpty())
			Expect(res.Transcript).To(ContainSubstring(`op="patch"`))
			Expect(res.Transcript).To(ContainSubstring(step.Path))
		})

		it("returns OutcomeError (no error) when patch errors (still includes patch transcript + error in Output)", func() {
			dur := expectDuration(mockClock, 50*time.Millisecond)

			cfg := agent.Config{DryRun: false}
			step := agent.Step{
				Type: agent.ToolFiles,
				Op:   "patch",
				Path: "/tmp/a.txt",
				Data: "@@ -1,1 +1,1 @@\n-a\n+b\n",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			expectAllowTool(mockBudget, agent.ToolFiles)

			patchErr := errors.New("apply patch /tmp/a.txt: first mismatch at line 7")
			mockFiles.
				EXPECT().
				PatchFile(step.Path, []byte(step.Data)).
				Return(agent.PatchResult{Hunks: 1}, patchErr).
				Times(1)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Exec).To(BeNil())
			Expect(res.Output).To(ContainSubstring(patchErr.Error()))

			Expect(res.Transcript).To(ContainSubstring(`op="patch"`))
			Expect(res.Transcript).To(ContainSubstring(step.Path))
			Expect(res.Transcript).To(ContainSubstring("error"))
		})

		it("replaces bytes in file and returns ok outcome (calls ReplaceBytesInFile)", func() {
			dur := expectDuration(mockClock, 50*time.Millisecond)

			cfg := agent.Config{DryRun: false}
			step := agent.Step{
				Type: agent.ToolFiles,
				Op:   "replace",
				Path: "/tmp/a.txt",
				Old:  "aa",
				New:  "XX",
				N:    2,
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			expectAllowTool(mockBudget, agent.ToolFiles)

			mockFiles.
				EXPECT().
				ReplaceBytesInFile(step.Path, []byte(step.Old), []byte(step.New), step.N).
				Return(agent.ReplaceResult{OccurrencesFound: 5, Replaced: 2}, nil).
				Times(1)

			// Guard: no other ops
			mockFiles.EXPECT().ReadFile(gomock.Any()).Times(0)
			mockFiles.EXPECT().WriteFile(gomock.Any(), gomock.Any()).Times(0)
			mockFiles.EXPECT().PatchFile(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeOK))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Exec).To(BeNil())

			Expect(res.Output).NotTo(BeEmpty())
			Expect(res.Transcript).To(ContainSubstring(`op="replace"`))
			Expect(res.Transcript).To(ContainSubstring(step.Path))
		})

		it("returns OutcomeError (no error) when replace errors and surfaces error in Output", func() {
			dur := expectDuration(mockClock, 50*time.Millisecond)

			cfg := agent.Config{DryRun: false}
			step := agent.Step{
				Type: agent.ToolFiles,
				Op:   "replace",
				Path: "/tmp/a.txt",
				Old:  "nope",
				New:  "x",
				N:    -1,
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			expectAllowTool(mockBudget, agent.ToolFiles)

			replErr := errors.New("replace /tmp/a.txt: pattern not found")
			mockFiles.
				EXPECT().
				ReplaceBytesInFile(step.Path, []byte(step.Old), []byte(step.New), step.N).
				Return(agent.ReplaceResult{OccurrencesFound: 0, Replaced: 0}, replErr).
				Times(1)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Step).To(Equal(step))
			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Exec).To(BeNil())
			Expect(res.Output).To(ContainSubstring(replErr.Error()))

			Expect(res.Transcript).To(ContainSubstring(`op="replace"`))
			Expect(res.Transcript).To(ContainSubstring(step.Path))
			Expect(res.Transcript).To(ContainSubstring("error"))
		})

		it("dry-run patch does not invoke PatchFile", func() {
			dur := expectDuration(mockClock, 10*time.Millisecond)

			cfg := agent.Config{DryRun: true}
			step := agent.Step{
				Type: agent.ToolFiles,
				Op:   "patch",
				Path: "/tmp/a.txt",
				Data: "diff-content",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			mockBudget.EXPECT().AllowTool(agent.ToolFiles, gomock.Any()).Times(0)

			mockFiles.EXPECT().PatchFile(gomock.Any(), gomock.Any()).Times(0)
			mockFiles.EXPECT().ReplaceBytesInFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			mockFiles.EXPECT().ReadFile(gomock.Any()).Times(0)
			mockFiles.EXPECT().WriteFile(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Outcome).To(Equal(agent.OutcomeDryRun))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Transcript).To(ContainSubstring("dry-run"))
			Expect(res.Transcript).To(ContainSubstring(`op="patch"`))
		})

		it("dry-run replace does not invoke ReplaceBytesInFile", func() {
			dur := expectDuration(mockClock, 10*time.Millisecond)

			cfg := agent.Config{DryRun: true}
			step := agent.Step{
				Type: agent.ToolFiles,
				Op:   "replace",
				Path: "/tmp/a.txt",
				Old:  "aa",
				New:  "XX",
				N:    0,
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)
			mockBudget.EXPECT().AllowTool(agent.ToolFiles, gomock.Any()).Times(0)

			mockFiles.EXPECT().PatchFile(gomock.Any(), gomock.Any()).Times(0)
			mockFiles.EXPECT().ReplaceBytesInFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			mockFiles.EXPECT().ReadFile(gomock.Any()).Times(0)
			mockFiles.EXPECT().WriteFile(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Outcome).To(Equal(agent.OutcomeDryRun))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Transcript).To(ContainSubstring("dry-run"))
			Expect(res.Transcript).To(ContainSubstring(`op="replace"`))
		})

		it("returns error StepResult when files tool budget is denied for patch and does not invoke PatchFile", func() {
			dur := expectDuration(mockClock, 10*time.Millisecond)

			cfg := agent.Config{DryRun: false}
			step := agent.Step{
				Type: agent.ToolFiles,
				Op:   "patch",
				Path: "/tmp/a.txt",
				Data: "diff",
			}

			expectAllowStep(mockBudget, step)
			expectAllowPolicy(mockPolicy, cfg, step)

			toolErr := errors.New("tool budget denied")
			mockBudget.EXPECT().AllowTool(agent.ToolFiles, gomock.Any()).Return(toolErr).Times(1)

			mockFiles.EXPECT().PatchFile(gomock.Any(), gomock.Any()).Times(0)

			res, err := subject.RunStep(context.Background(), cfg, step)
			Expect(err).To(MatchError(toolErr))

			Expect(res.Outcome).To(Equal(agent.OutcomeError))
			Expect(res.Duration).To(Equal(dur))
			Expect(res.Transcript).To(ContainSubstring("[budget]"))
		})
	})
}

func expectDuration(mockClock *MockClock, d time.Duration) time.Duration {
	t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)
	t1 := t0.Add(d)

	// Robust to extra clock.Now() calls:
	// first call = t0, all subsequent calls = t1.
	gomock.InOrder(
		mockClock.EXPECT().Now().Return(t0).Times(1),
		mockClock.EXPECT().Now().Return(t1).AnyTimes(),
	)

	return d
}

func expectAllowStep(mockBudget *MockBudget, step agent.Step) {
	mockBudget.
		EXPECT().
		AllowStep(step, gomock.Any()).
		Return(nil).
		Times(1)
}

func expectAllowTool(mockBudget *MockBudget, kind agent.ToolKind) {
	mockBudget.
		EXPECT().
		AllowTool(kind, gomock.Any()).
		Return(nil).
		Times(1)
}

func expectAllowPolicy(mockPolicy *MockPolicy, cfg agent.Config, step agent.Step) {
	mockPolicy.
		EXPECT().
		AllowStep(cfg, step).
		Return(nil).
		Times(1)
}

func expectShellTranscript(res agent.StepResult, cfg agent.Config, step agent.Step, exec agent.Result) {
	Expect(res.Transcript).To(ContainSubstring(`[shell]`))
	Expect(res.Transcript).To(ContainSubstring(fmt.Sprintf(`workdir=%q`, cfg.WorkDir)))
	Expect(res.Transcript).To(ContainSubstring(fmt.Sprintf(`cmd=%q`, step.Command)))
	Expect(res.Transcript).To(ContainSubstring(fmt.Sprintf("exit=%d", exec.ExitCode)))

	if exec.Stdout != "" {
		Expect(res.Transcript).To(ContainSubstring("stdout:\n"))
		Expect(res.Transcript).To(ContainSubstring(exec.Stdout))
	}
	if exec.Stderr != "" {
		Expect(res.Transcript).To(ContainSubstring("stderr:\n"))
		Expect(res.Transcript).To(ContainSubstring(exec.Stderr))
	}
}

func expectLLMStartTranscript(res agent.StepResult, step agent.Step) {
	Expect(res.Transcript).To(ContainSubstring("[llm:start]"))
	Expect(res.Transcript).To(ContainSubstring("prompt:\n"))
	Expect(res.Transcript).To(ContainSubstring(step.Prompt))
}

func expectLLMOKTranscript(res agent.StepResult, step agent.Step, out string) {
	Expect(res.Transcript).To(ContainSubstring("[llm]"))
	Expect(res.Transcript).To(ContainSubstring("prompt:\n"))
	Expect(res.Transcript).To(ContainSubstring(step.Prompt))
	Expect(res.Transcript).To(ContainSubstring("output:\n"))
	Expect(res.Transcript).To(ContainSubstring(out))
}

func expectFileStartTranscript(res agent.StepResult, step agent.Step) {
	Expect(res.Transcript).To(ContainSubstring(`[file:start]`))
	Expect(res.Transcript).To(ContainSubstring(fmt.Sprintf(`op=%q`, step.Op)))
	Expect(res.Transcript).To(ContainSubstring(fmt.Sprintf(`path=%q`, step.Path)))
	Expect(res.Transcript).To(ContainSubstring(fmt.Sprintf(`data_len=%d`, len(step.Data))))
}

func expectFileReadTranscript(res agent.StepResult, path, content string) {
	Expect(res.Transcript).To(ContainSubstring(`[file] op="read"`))
	Expect(res.Transcript).To(ContainSubstring(fmt.Sprintf(`path=%q`, path)))
	Expect(res.Transcript).To(ContainSubstring("content:\n"))
	// content can be large; just check it appears (or a prefix)
	if content != "" {
		Expect(res.Transcript).To(ContainSubstring(content))
	}
}

func expectFileWriteTranscript(res agent.StepResult, path string, dataLen int) {
	Expect(res.Transcript).To(ContainSubstring(`[file] op="write"`))
	Expect(res.Transcript).To(ContainSubstring(fmt.Sprintf(`path=%q`, path)))
	Expect(res.Transcript).To(ContainSubstring(fmt.Sprintf("data_len=%d", dataLen)))
}

func expectLLMSnapshotOK(mockBudget *MockBudget) {
	mockBudget.
		EXPECT().
		Snapshot(gomock.Any()).
		Return(agent.BudgetSnapshot{
			Limits:        agent.BudgetLimits{MaxLLMTokens: 0}, // 0 = unlimited => preflight passes
			LLMTokensUsed: 0,
		}).
		Times(1)
}
