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

func TestUnitReAct(t *testing.T) {
	spec.Run(t, "Testing ReActAgent", testReActAgent, spec.Report(report.Terminal{}))
}

func testReActAgent(t *testing.T, when spec.G, it spec.S) {
	var (
		ctrl   *gomock.Controller
		llm    *MockLLM
		runner *MockRunner
		budget *MockBudget
		clock  *MockClock

		reactAgent *agent.ReActAgent
		ctx        context.Context
		now        time.Time
	)

	it.Before(func() {
		RegisterTestingT(t)

		ctrl = gomock.NewController(t)
		llm = NewMockLLM(ctrl)
		runner = NewMockRunner(ctrl)
		budget = NewMockBudget(ctrl)
		clock = NewMockClock(ctrl)

		reactAgent = agent.NewReActAgent(llm, runner, budget, clock)
		ctx = context.Background()
		now = time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

		clock.EXPECT().Now().Return(now).AnyTimes()
	})

	it.After(func() {
		ctrl.Finish()
	})

	when("LLM returns final answer immediately", func() {
		it("returns the answer without tool calls", func() {

			budget.EXPECT().AllowIteration(now).Return(nil)              // NEW
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{}) // NEW (no token limit)
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return(`{
					"thought": "The answer is simple",
					"action_type": "answer",
					"final_answer": "42"
				}`, 10, nil)

			budget.EXPECT().ChargeLLMTokens(10, now)

			_, err := reactAgent.RunAgentGoal(ctx, "What is the answer?")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	when("LLM uses a shell tool then answers", func() {
		it("executes the tool and returns the final answer", func() {

			// Iteration 1: tool
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return(`{
					"thought": "I need to list files",
					"action_type": "tool",
					"tool": "shell",
					"command": "ls",
					"args": ["-la"]
				}`, 15, nil)

			budget.EXPECT().ChargeLLMTokens(15, now)

			runner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ agent.Config, step agent.Step) (agent.StepResult, error) {
					Expect(step.Type).To(Equal(agent.ToolShell))
					Expect(step.Command).To(Equal("ls"))
					Expect(step.Args).To(Equal([]string{"-la"}))
					return agent.StepResult{
						Outcome:  agent.OutcomeOK,
						Output:   "file1.txt\nfile2.txt",
						Duration: 100 * time.Millisecond,
					}, nil
				})

			// Iteration 2: answer
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, prompt string) (string, int, error) {
					Expect(prompt).To(ContainSubstring("OBSERVATION: file1.txt"))
					return `{
						"thought": "I have the file list",
						"action_type": "answer",
						"final_answer": "There are 2 files"
					}`, 12, nil
				})

			budget.EXPECT().ChargeLLMTokens(12, now)

			_, err := reactAgent.RunAgentGoal(ctx, "How many files?")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	when("budget is exceeded", func() {
		it("returns budget error", func() {

			budget.EXPECT().AllowIteration(now).Return(nil) // allow iteration
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(agent.BudgetExceededError{
				Kind:    agent.BudgetKindLLM,
				Limit:   5,
				Used:    5,
				Message: "llm call budget exceeded",
			})

			_, err := reactAgent.RunAgentGoal(ctx, "Do something")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("llm call budget exceeded"))
		})
	})

	when("LLM returns invalid JSON", func() {
		it("returns parse error", func() {

			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return("not valid json", 5, nil)

			budget.EXPECT().ChargeLLMTokens(5, now)

			_, err := reactAgent.RunAgentGoal(ctx, "Do something")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to locate JSON"))
		})
	})

	when("LLM returns JSON with missing action_type", func() {
		it("returns validation error", func() {

			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return(`{"thought": "thinking"}`, 5, nil)

			budget.EXPECT().ChargeLLMTokens(5, now)

			_, err := reactAgent.RunAgentGoal(ctx, "Do something")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing action_type"))
		})
	})

	when("tool execution fails", func() {
		it("returns the execution error", func() {

			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return(`{
					"thought": "running command",
					"action_type": "tool",
					"tool": "shell",
					"command": "false"
				}`, 10, nil)

			budget.EXPECT().ChargeLLMTokens(10, now)

			runner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(agent.StepResult{
					Outcome:    agent.OutcomeError,
					Transcript: "command failed",
				}, errors.New("exit 1"))

			_, err := reactAgent.RunAgentGoal(ctx, "Run false")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exit 1"))
		})
	})

	when("iteration budget is exceeded", func() {
		it("returns iteration budget exceeded error", func() {

			// 10 successful iterations, then fail on 11th AllowIteration
			for i := 0; i < 10; i++ {
				budget.EXPECT().AllowIteration(now).Return(nil)
				budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
				budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

				llm.EXPECT().
					Complete(gomock.Any(), gomock.Any()).
					Return(fmt.Sprintf(`{
					"thought": "still working",
					"action_type": "tool",
					"tool": "shell",
					"command": "echo",
					"args": ["test-%d"]
				}`, i), 10, nil)

				budget.EXPECT().ChargeLLMTokens(10, now)
			}

			// We expect 10 tool executions total.
			runner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				Times(10).
				Return(agent.StepResult{
					Outcome:  agent.OutcomeOK,
					Output:   "ok",
					Duration: 10 * time.Millisecond,
				}, nil)

			budget.EXPECT().AllowIteration(now).Return(agent.BudgetExceededError{
				Kind:    agent.BudgetKindIterations,
				Limit:   10,
				Used:    10,
				Message: "iteration budget exceeded",
			})

			_, err := reactAgent.RunAgentGoal(ctx, "Keep looping")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("iteration budget exceeded"))
		})
	})

	when("LLM output has markdown code fences", func() {
		it("strips the fences and parses correctly", func() {

			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return("```json\n{\"thought\": \"done\", \"action_type\": \"answer\", \"final_answer\": \"Success\"}\n```", 10, nil)

			budget.EXPECT().ChargeLLMTokens(10, now)

			_, err := reactAgent.RunAgentGoal(ctx, "Test markdown")
			Expect(err).NotTo(HaveOccurred())
		})

		when("shell tool missing command", func() {
			it("returns validation error", func() {

				budget.EXPECT().AllowIteration(now).Return(nil)
				budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
				budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

				llm.EXPECT().
					Complete(gomock.Any(), gomock.Any()).
					Return(`{
					"thought": "using shell",
					"action_type": "tool",
					"tool": "shell",
					"command": ""
				}`, 10, nil)

				budget.EXPECT().ChargeLLMTokens(10, now)

				_, err := reactAgent.RunAgentGoal(ctx, "Test")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("shell tool requires command"))
			})
		})
	})

	when("LLM uses shorthand action_type like file/shell/llm", func() {
		it("treats action_type=file as a tool call", func() {

			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			// NOTE: action_type is "file" (shorthand). tool is omitted.
			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return(`{
				"thought": "read it",
				"action_type": "file",
				"op": "read",
				"path": "AGENTS.md"
			}`, 10, nil)

			budget.EXPECT().ChargeLLMTokens(10, now)

			runner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ agent.Config, step agent.Step) (agent.StepResult, error) {
					Expect(step.Type).To(Equal(agent.ToolFiles))
					Expect(step.Op).To(Equal("read"))
					Expect(step.Path).To(Equal("AGENTS.md"))
					return agent.StepResult{
						Outcome:  agent.OutcomeOK,
						Output:   "ok",
						Duration: 1 * time.Millisecond,
					}, nil
				})

			// Next iteration: answer
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return(`{
				"thought": "done",
				"action_type": "answer",
				"final_answer": "ok"
			}`, 5, nil)

			budget.EXPECT().ChargeLLMTokens(5, now)

			_, err := reactAgent.RunAgentGoal(ctx, "Read AGENTS")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	when("LLM returns multiple JSON objects back-to-back", func() {
		it("parses only the first JSON object", func() {

			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			// Two JSON objects concatenated. parseReActResponse should take only the first.
			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return(
					`{"thought":"one","action_type":"answer","final_answer":"A"}{"thought":"two","action_type":"answer","final_answer":"B"}`,
					10,
					nil,
				)

			budget.EXPECT().ChargeLLMTokens(10, now)

			res, err := reactAgent.RunAgentGoal(ctx, "Test")
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal("A"))
		})
	})

	when("LLM repeats the same tool call twice in a row", func() {
		it("injects a repetition observation and forces a different next step", func() {

			// Iteration 1: tool
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return(`{
				"thought": "do it",
				"action_type": "tool",
				"tool": "shell",
				"command": "ls",
				"args": ["-la"]
			}`, 10, nil)

			budget.EXPECT().ChargeLLMTokens(10, now)

			// Only ONE tool execution should happen (iteration 1).
			runner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				Times(1).
				Return(agent.StepResult{
					Outcome:  agent.OutcomeOK,
					Output:   "file1\nfile2\n",
					Duration: 1 * time.Millisecond,
				}, nil)

			// Iteration 2: same tool again (should be blocked by repetition guard)
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return(`{
				"thought": "try again",
				"action_type": "tool",
				"tool": "shell",
				"command": "ls",
				"args": ["-la"]
			}`, 10, nil)

			budget.EXPECT().ChargeLLMTokens(10, now)

			// Iteration 3: must see injected repetition message in prompt, then answer
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, prompt string) (string, int, error) {
					Expect(prompt).To(ContainSubstring("OBSERVATION: You are repeating the same tool call"))
					return `{
					"thought": "ok, I'll stop repeating",
					"action_type": "answer",
					"final_answer": "done"
				}`, 5, nil
				})

			budget.EXPECT().ChargeLLMTokens(5, now)

			res, err := reactAgent.RunAgentGoal(ctx, "List files")
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal("done"))
		})
	})

	when("LLM ignores repetition warnings", func() {
		it("hard-stops after too many repeats in the rolling window", func() {

			// We'll do 6 iterations total (1 executes, 2-5 skipped, 6 hard-stops)
			for i := 0; i < 6; i++ {
				budget.EXPECT().AllowIteration(now).Return(nil)
				budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
				budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

				llm.EXPECT().
					Complete(gomock.Any(), gomock.Any()).
					Return(`{
          "thought": "list files again",
          "action_type": "tool",
          "tool": "shell",
          "command": "ls",
          "args": ["-la"]
        }`, 1, nil)

				budget.EXPECT().ChargeLLMTokens(1, now)
			}

			// Only the FIRST iteration should actually execute the tool.
			runner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				Times(1).
				DoAndReturn(func(_ context.Context, _ agent.Config, step agent.Step) (agent.StepResult, error) {
					Expect(step.Type).To(Equal(agent.ToolShell))
					Expect(step.Command).To(Equal("ls"))
					Expect(step.Args).To(Equal([]string{"-la"}))
					return agent.StepResult{
						Outcome:  agent.OutcomeOK,
						Output:   "file1\nfile2\n",
						Duration: 10 * time.Millisecond,
					}, nil
				})

			_, err := reactAgent.RunAgentGoal(ctx, "Loop forever")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("agent appears stuck"))
			Expect(err.Error()).To(ContainSubstring("repeated tool call too many times"))
		})
	})

	when("LLM uses shorthand action_type=file (no tool field)", func() {
		it("treats it as a tool call and executes file op", func() {

			// Iteration 1: shorthand file tool
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return(`{
				"thought": "read README",
				"action_type": "file",
				"op": "read",
				"path": "README.md"
			}`, 10, nil)

			budget.EXPECT().ChargeLLMTokens(10, now)

			runner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ agent.Config, step agent.Step) (agent.StepResult, error) {
					Expect(step.Type).To(Equal(agent.ToolFiles))
					Expect(step.Op).To(Equal("read"))
					Expect(step.Path).To(Equal("README.md"))
					return agent.StepResult{
						Outcome:  agent.OutcomeOK,
						Output:   "README CONTENT",
						Duration: 5 * time.Millisecond,
					}, nil
				})

			// Iteration 2: answer
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, prompt string) (string, int, error) {
					Expect(prompt).To(ContainSubstring("OBSERVATION: README CONTENT"))
					return `{
					"thought": "done",
					"action_type": "answer",
					"final_answer": "ok"
				}`, 1, nil
				})

			budget.EXPECT().ChargeLLMTokens(1, now)

			_, err := reactAgent.RunAgentGoal(ctx, "Read README and answer")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	when("LLM uses shorthand action_type=file AND also sets tool=file", func() {
		it("still treats it as a tool call (compat mode)", func() {

			// Iteration 1: shorthand but with tool field present
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return(`{
				"thought": "read AGENTS",
				"action_type": "file",
				"tool": "file",
				"op": "read",
				"path": "AGENTS.md"
			}`, 10, nil)

			budget.EXPECT().ChargeLLMTokens(10, now)

			runner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(agent.StepResult{
					Outcome:  agent.OutcomeOK,
					Output:   "AGENTS CONTENT",
					Duration: 5 * time.Millisecond,
				}, nil)

			// Iteration 2: answer
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return(`{
				"thought": "done",
				"action_type": "answer",
				"final_answer": "ok"
			}`, 1, nil)

			budget.EXPECT().ChargeLLMTokens(1, now)

			_, err := reactAgent.RunAgentGoal(ctx, "Read AGENTS and answer")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	when("LLM uses shorthand action_type=file but tool mismatches", func() {
		it("fails validation (invalid action_type) rather than running tools", func() {

			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				// tool is present but inconsistent; we should NOT normalize
				Return(`{
				"thought": "oops",
				"action_type": "file",
				"tool": "shell",
				"command": "ls"
			}`, 10, nil)

			budget.EXPECT().ChargeLLMTokens(10, now)

			_, err := reactAgent.RunAgentGoal(ctx, "Bad shorthand")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`invalid action_type: "file"`))
		})
	})

	when("LLM uses file patch", func() {
		it("converts to a ToolFiles step and executes it", func() {
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(`{
      "thought":"apply diff",
      "action_type":"tool",
      "tool":"file",
      "op":"patch",
      "path":"a.txt",
      "data":"--- a/a.txt\n+++ b/a.txt\n@@\n+hi\n"
    }`, 1, nil)
			budget.EXPECT().ChargeLLMTokens(1, now)

			runner.EXPECT().RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ agent.Config, step agent.Step) (agent.StepResult, error) {
					Expect(step.Type).To(Equal(agent.ToolFiles))
					Expect(step.Op).To(Equal("patch"))
					Expect(step.Path).To(Equal("a.txt"))
					Expect(step.Data).To(ContainSubstring("+++ b/a.txt"))
					return agent.StepResult{Outcome: agent.OutcomeOK, Output: "patched"}, nil
				})

			// next iteration: answer
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(`{
      "thought":"done",
      "action_type":"answer",
      "final_answer":"ok"
    }`, 1, nil)
			budget.EXPECT().ChargeLLMTokens(1, now)

			_, err := reactAgent.RunAgentGoal(ctx, "Patch a.txt")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	when("LLM uses file replace", func() {
		it("converts to a ToolFiles step with Old/New/N", func() {
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(`{
      "thought":"swap token",
      "action_type":"tool",
      "tool":"file",
      "op":"replace",
      "path":"a.txt",
      "old":"foo",
      "new":"bar",
      "n":2
    }`, 1, nil)
			budget.EXPECT().ChargeLLMTokens(1, now)

			runner.EXPECT().RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ agent.Config, step agent.Step) (agent.StepResult, error) {
					Expect(step.Type).To(Equal(agent.ToolFiles))
					Expect(step.Op).To(Equal("replace"))
					Expect(step.Path).To(Equal("a.txt"))
					Expect(step.Old).To(Equal("foo"))
					Expect(step.New).To(Equal("bar"))
					Expect(step.N).To(Equal(2))
					return agent.StepResult{Outcome: agent.OutcomeOK, Output: "replaced"}, nil
				})

			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)
			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(`{
      "thought":"done",
      "action_type":"answer",
      "final_answer":"ok"
    }`, 1, nil)
			budget.EXPECT().ChargeLLMTokens(1, now)

			_, err := reactAgent.RunAgentGoal(ctx, "Replace in a.txt")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	when("LLM uses file patch without data", func() {
		it("returns validation error", func() {
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(`{
      "thought":"patch",
      "action_type":"tool",
      "tool":"file",
      "op":"patch",
      "path":"a.txt",
      "data":"   "
    }`, 1, nil)
			budget.EXPECT().ChargeLLMTokens(1, now)

			_, err := reactAgent.RunAgentGoal(ctx, "Patch")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("file patch requires data"))
		})
	})

	when("LLM uses file replace without old", func() {
		it("returns validation error", func() {
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(`{
      "thought":"replace",
      "action_type":"tool",
      "tool":"file",
      "op":"replace",
      "path":"a.txt",
      "new":""
    }`, 1, nil)
			budget.EXPECT().ChargeLLMTokens(1, now)

			_, err := reactAgent.RunAgentGoal(ctx, "Replace")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("file replace requires old pattern"))
		})
	})

	when("patch fails and agent falls back to full write", func() {
		it("continues after patch failure observation and then writes", func() {
			// Iteration 1: patch
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(`{
      "thought":"try patch first",
      "action_type":"tool",
      "tool":"file",
      "op":"patch",
      "path":"a.txt",
      "data":"--- a/a.txt\n+++ b/a.txt\n@@\n+hi\n"
    }`, 1, nil)
			budget.EXPECT().ChargeLLMTokens(1, now)

			runner.EXPECT().RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ agent.Config, step agent.Step) (agent.StepResult, error) {
					Expect(step.Op).To(Equal("patch"))
					// IMPORTANT: err == nil, failure is conveyed in Output/Outcome
					return agent.StepResult{
						Outcome:  agent.OutcomeError,
						Output:   "patch failed: hunk did not apply",
						Duration: 1 * time.Millisecond,
					}, nil
				})

			// Iteration 2: LLM sees patch failed and chooses full write
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, prompt string) (string, int, error) {
					Expect(prompt).To(ContainSubstring("OBSERVATION: ERROR:"))
					Expect(prompt).To(ContainSubstring("patch failed"))
					return `{
          "thought":"fallback to write full file",
          "action_type":"tool",
          "tool":"file",
          "op":"write",
          "path":"a.txt",
          "data":"FULL NEW CONTENT\n"
        }`, 1, nil
				})
			budget.EXPECT().ChargeLLMTokens(1, now)

			runner.EXPECT().RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ agent.Config, step agent.Step) (agent.StepResult, error) {
					Expect(step.Op).To(Equal("write"))
					Expect(step.Path).To(Equal("a.txt"))
					Expect(step.Data).To(Equal("FULL NEW CONTENT\n"))
					return agent.StepResult{Outcome: agent.OutcomeOK, Output: "wrote"}, nil
				})

			// Iteration 3: answer
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(`{
      "thought":"done",
      "action_type":"answer",
      "final_answer":"ok"
    }`, 1, nil)
			budget.EXPECT().ChargeLLMTokens(1, now)

			_, err := reactAgent.RunAgentGoal(ctx, "Modify a.txt")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	when("a step produces side effects", func() {
		it("includes STATE line with cumulative effects in the next prompt", func() {
			// iter 1
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(`{
      "thought":"write a file",
      "action_type":"tool",
      "tool":"file",
      "op":"write",
      "path":"a.txt",
      "data":"hi"
    }`, 1, nil)
			budget.EXPECT().ChargeLLMTokens(1, now)

			runner.EXPECT().RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(agent.StepResult{
					Outcome:  agent.OutcomeOK,
					Output:   "wrote",
					Duration: 1 * time.Millisecond,
					Effects: agent.Effects{
						{Kind: "file.write", Path: "a.txt"},
					},
				}, nil)

			// iter 2
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, prompt string) (string, int, error) {
					Expect(prompt).To(ContainSubstring("State:"))
					Expect(prompt).To(ContainSubstring("file.write x1"))
					return `{
          "thought":"done",
          "action_type":"answer",
          "final_answer":"ok"
        }`, 1, nil
				})
			budget.EXPECT().ChargeLLMTokens(1, now)

			_, err := reactAgent.RunAgentGoal(ctx, "Write")
			Expect(err).NotTo(HaveOccurred())
		})
	})
}
