package agent_test

import (
	"context"
	"errors"
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
	})

	it.After(func() {
		ctrl.Finish()
	})

	when("LLM returns final answer immediately", func() {
		it("returns the answer without tool calls", func() {
			clock.EXPECT().Now().Return(now) // start

			clock.EXPECT().Now().Return(now)                             // iteration 1
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

			clock.EXPECT().Now().Return(now) // end (defer - out.Infof)
			clock.EXPECT().Now().Return(now) // end (defer - dbg.Infof)

			_, err := reactAgent.RunAgentGoal(ctx, "What is the answer?")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	when("LLM uses a shell tool then answers", func() {
		it("executes the tool and returns the final answer", func() {
			clock.EXPECT().Now().Return(now) // start

			// Iteration 1: tool
			clock.EXPECT().Now().Return(now)
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
			clock.EXPECT().Now().Return(now)
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

			clock.EXPECT().Now().Return(now) // end
			clock.EXPECT().Now().Return(now) // end

			_, err := reactAgent.RunAgentGoal(ctx, "How many files?")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	when("budget is exceeded", func() {
		it("returns budget error", func() {
			clock.EXPECT().Now().Return(now) // start

			clock.EXPECT().Now().Return(now)                // iteration
			budget.EXPECT().AllowIteration(now).Return(nil) // allow iteration
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(agent.BudgetExceededError{
				Kind:    agent.BudgetKindLLM,
				Limit:   5,
				Used:    5,
				Message: "llm call budget exceeded",
			})

			clock.EXPECT().Now().Return(now) // end
			clock.EXPECT().Now().Return(now) // end

			_, err := reactAgent.RunAgentGoal(ctx, "Do something")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("llm call budget exceeded"))
		})
	})

	when("LLM returns invalid JSON", func() {
		it("returns parse error", func() {
			clock.EXPECT().Now().Return(now) // start

			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return("not valid json", 5, nil)

			budget.EXPECT().ChargeLLMTokens(5, now)

			clock.EXPECT().Now().Return(now) // end
			clock.EXPECT().Now().Return(now) // end

			_, err := reactAgent.RunAgentGoal(ctx, "Do something")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to locate JSON"))
		})
	})

	when("LLM returns JSON with missing action_type", func() {
		it("returns validation error", func() {
			clock.EXPECT().Now().Return(now) // start

			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return(`{"thought": "thinking"}`, 5, nil)

			budget.EXPECT().ChargeLLMTokens(5, now)

			clock.EXPECT().Now().Return(now) // end
			clock.EXPECT().Now().Return(now) // end

			_, err := reactAgent.RunAgentGoal(ctx, "Do something")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing action_type"))
		})
	})

	when("tool execution fails", func() {
		it("returns the execution error", func() {
			clock.EXPECT().Now().Return(now) // start

			clock.EXPECT().Now().Return(now)
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

			clock.EXPECT().Now().Return(now) // end
			clock.EXPECT().Now().Return(now) // end

			_, err := reactAgent.RunAgentGoal(ctx, "Run false")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exit 1"))
		})
	})

	when("iteration budget is exceeded", func() {
		it("returns iteration budget exceeded error", func() {
			clock.EXPECT().Now().Return(now) // start

			// 10 successful iterations, then fail on 11th AllowIteration
			for i := 0; i < 10; i++ {
				clock.EXPECT().Now().Return(now)
				budget.EXPECT().AllowIteration(now).Return(nil)
				budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
				budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

				llm.EXPECT().
					Complete(gomock.Any(), gomock.Any()).
					Return(`{
						"thought": "still working",
						"action_type": "tool",
						"tool": "shell",
						"command": "echo",
						"args": ["test"]
					}`, 10, nil)

				budget.EXPECT().ChargeLLMTokens(10, now)

				runner.EXPECT().
					RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(agent.StepResult{
						Outcome:  agent.OutcomeOK,
						Output:   "test",
						Duration: 10 * time.Millisecond,
					}, nil)
			}

			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowIteration(now).Return(agent.BudgetExceededError{
				Kind:    agent.BudgetKindIterations,
				Limit:   10,
				Used:    10,
				Message: "iteration budget exceeded",
			})

			clock.EXPECT().Now().Return(now) // end
			clock.EXPECT().Now().Return(now) // end

			_, err := reactAgent.RunAgentGoal(ctx, "Keep looping")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("iteration budget exceeded"))
		})
	})

	when("LLM output has markdown code fences", func() {
		it("strips the fences and parses correctly", func() {
			clock.EXPECT().Now().Return(now) // start

			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowIteration(now).Return(nil)
			budget.EXPECT().Snapshot(now).Return(agent.BudgetSnapshot{})
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return("```json\n{\"thought\": \"done\", \"action_type\": \"answer\", \"final_answer\": \"Success\"}\n```", 10, nil)

			budget.EXPECT().ChargeLLMTokens(10, now)

			clock.EXPECT().Now().Return(now) // end
			clock.EXPECT().Now().Return(now) // end

			_, err := reactAgent.RunAgentGoal(ctx, "Test markdown")
			Expect(err).NotTo(HaveOccurred())
		})

		when("shell tool missing command", func() {
			it("returns validation error", func() {
				clock.EXPECT().Now().Return(now) // start

				clock.EXPECT().Now().Return(now) // iteration
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

				clock.EXPECT().Now().Return(now) // end (defer - out.Infof)
				clock.EXPECT().Now().Return(now) // end (defer - dbg.Infof)

				_, err := reactAgent.RunAgentGoal(ctx, "Test")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("shell tool requires command"))
			})
		})
	})
}
