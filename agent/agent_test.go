package agent_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/kardolus/chatgpt-cli/agent"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitAgent(t *testing.T) {
	spec.Run(t, "Testing the agent orchestration", testAgent, spec.Report(report.Terminal{}))
}

func testAgent(t *testing.T, when spec.G, it spec.S) {
	var (
		mockCtrl    *gomock.Controller
		mockClock   *MockClock
		mockRunner  *MockRunner
		mockPlanner *MockPlanner
	)

	it.Before(func() {
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)
		mockClock = NewMockClock(mockCtrl)
		mockPlanner = NewMockPlanner(mockCtrl)
		mockRunner = NewMockRunner(mockCtrl)
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("RunAgentGoal()", func() {
		const goal = "test goal"

		it.Before(func() {
			expectAgentDuration(mockClock, 123*time.Millisecond)
		})

		it("should bubble up planner errors and not run any steps", func() {
			planErr := fmt.Errorf("planner boom")

			mockPlanner.
				EXPECT().
				Plan(gomock.Any(), goal).
				Return(agent.Plan{}, planErr).
				Times(1)

			// Runner must not be invoked if planning fails
			mockRunner.
				EXPECT().
				RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				Times(0)

			subject := agent.NewAgent(
				mockClock,
				mockPlanner,
				mockRunner,
			)

			err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).To(MatchError(planErr))
		})

		it("should bubble up runner errors and stop executing further steps", func() {
			runErr := fmt.Errorf("runner boom")

			plan := agent.Plan{
				Goal: goal,
				Steps: []agent.Step{
					{Type: agent.ToolShell, Description: "step 1", Command: "echo", Args: []string{"one"}},
					{Type: agent.ToolShell, Description: "step 2", Command: "echo", Args: []string{"two"}},
				},
			}

			mockPlanner.
				EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			mockRunner.
				EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				Return(agent.StepResult{}, runErr).
				Times(1)

			// Guard: step 2 must not run
			mockRunner.
				EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[1]).
				Times(0)

			subject := agent.NewAgent(
				mockClock,
				mockPlanner,
				mockRunner,
			)

			err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).To(MatchError(runErr))
		})

		it("should return an error when runner returns OutcomeError (even if err == nil)", func() {
			plan := agent.Plan{
				Goal: goal,
				Steps: []agent.Step{
					{Type: agent.ToolShell, Description: "step 1", Command: "false", Args: nil},
				},
			}

			mockPlanner.
				EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			mockRunner.
				EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				Return(agent.StepResult{
					Step:    plan.Steps[0],
					Outcome: agent.OutcomeError,
					Exec:    &agent.Result{ExitCode: 42}, // optional; Agent no longer inspects this
				}, nil).
				Times(1)

			subject := agent.NewAgent(
				mockClock,
				mockPlanner,
				mockRunner,
			)

			err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).To(MatchError(`step failed: step 1`))
		})

		it("should succeed when planner returns an empty plan and not run any steps", func() {
			plan := agent.Plan{
				Goal:  goal,
				Steps: nil,
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				Times(0)

			subject := agent.NewAgent(
				mockClock,
				mockPlanner,
				mockRunner,
			)

			Expect(subject.RunAgentGoal(context.Background(), goal)).To(Succeed())
		})

		it("should stop executing further steps when runner returns OutcomeError", func() {
			plan := agent.Plan{
				Goal: goal,
				Steps: []agent.Step{
					{Type: agent.ToolShell, Description: "step 1", Command: "false", Args: nil},
					{Type: agent.ToolShell, Description: "step 2", Command: "echo", Args: []string{"should-not-run"}},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				Return(agent.StepResult{
					Step:    plan.Steps[0],
					Outcome: agent.OutcomeError,
					Exec:    &agent.Result{ExitCode: 7},
				}, nil).
				Times(1)

			// Guard: step 2 must not run
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[1]).
				Times(0)

			subject := agent.NewAgent(
				mockClock,
				mockPlanner,
				mockRunner,
			)

			err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).To(MatchError(`step failed: step 1`))
		})

		it("should treat Exec == nil as success and continue to next step", func() {
			plan := agent.Plan{
				Goal: goal,
				Steps: []agent.Step{
					{Type: agent.ToolLLM, Description: "llm step (no exec)", Prompt: "do something"},
					{Type: agent.ToolShell, Description: "shell step", Command: "echo", Args: []string{"ok"}},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			// First step: Exec is nil, no error, OutcomeOK => success.
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				Return(agent.StepResult{
					Step:    plan.Steps[0],
					Outcome: agent.OutcomeOK,
					Exec:    nil,
				}, nil).
				Times(1)

			// Second step should still run.
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[1]).
				Return(agent.StepResult{
					Step:    plan.Steps[1],
					Outcome: agent.OutcomeOK,
					Exec:    &agent.Result{ExitCode: 0},
				}, nil).
				Times(1)

			subject := agent.NewAgent(
				mockClock,
				mockPlanner,
				mockRunner,
			)

			Expect(subject.RunAgentGoal(context.Background(), goal)).To(Succeed())
		})

		it("WithWorkDir should pass cfg.WorkDir into the runner", func() {
			plan := agent.Plan{
				Goal: goal,
				Steps: []agent.Step{
					{Type: agent.ToolShell, Description: "step 1", Command: "echo", Args: []string{"one"}},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				DoAndReturn(func(_ context.Context, cfg agent.Config, _ agent.Step) (agent.StepResult, error) {
					Expect(cfg.WorkDir).To(Equal("/tmp/my-workdir"))
					return agent.StepResult{Step: plan.Steps[0], Outcome: agent.OutcomeOK}, nil
				}).
				Times(1)

			subject := agent.NewAgent(
				mockClock,
				mockPlanner,
				mockRunner,
				agent.WithWorkDir("/tmp/my-workdir"),
			)

			Expect(subject.RunAgentGoal(context.Background(), goal)).To(Succeed())
		})

		it("WithDryRun should pass cfg.DryRun into the runner", func() {
			plan := agent.Plan{
				Goal: goal,
				Steps: []agent.Step{
					{Type: agent.ToolShell, Description: "step 1", Command: "echo", Args: []string{"one"}},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				DoAndReturn(func(_ context.Context, cfg agent.Config, _ agent.Step) (agent.StepResult, error) {
					Expect(cfg.DryRun).To(BeTrue())
					return agent.StepResult{Step: plan.Steps[0], Outcome: agent.OutcomeDryRun}, nil
				}).
				Times(1)

			subject := agent.NewAgent(
				mockClock,
				mockPlanner,
				mockRunner,
				agent.WithDryRun(true),
			)

			Expect(subject.RunAgentGoal(context.Background(), goal)).To(Succeed())
		})

		it("should run all planned steps (Agent no longer enforces MaxSteps)", func() {
			plan := agent.Plan{
				Goal: goal,
				Steps: []agent.Step{
					{Type: agent.ToolShell, Description: "step 1", Command: "echo", Args: []string{"one"}},
					{Type: agent.ToolShell, Description: "step 2", Command: "echo", Args: []string{"two"}},
					{Type: agent.ToolShell, Description: "step 3", Command: "echo", Args: []string{"three"}},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			gomock.InOrder(
				mockRunner.EXPECT().
					RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
					Return(agent.StepResult{Step: plan.Steps[0], Outcome: agent.OutcomeOK}, nil),
				mockRunner.EXPECT().
					RunStep(gomock.Any(), gomock.Any(), plan.Steps[1]).
					Return(agent.StepResult{Step: plan.Steps[1], Outcome: agent.OutcomeOK}, nil),
				mockRunner.EXPECT().
					RunStep(gomock.Any(), gomock.Any(), plan.Steps[2]).
					Return(agent.StepResult{Step: plan.Steps[2], Outcome: agent.OutcomeOK}, nil),
			)

			subject := agent.NewAgent(
				mockClock,
				mockPlanner,
				mockRunner,
			)

			Expect(subject.RunAgentGoal(context.Background(), goal)).To(Succeed())
		})

		it("should accumulate results and render templates for later steps", func() {
			plan := agent.Plan{
				Goal: goal,
				Steps: []agent.Step{
					{Type: agent.ToolLLM, Description: "step 1", Prompt: "first"},
					{Type: agent.ToolLLM, Description: "step 2", Prompt: "use {{ (index .Results 0).Output }}"},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			// Step 1 runs normally.
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ agent.Config, s agent.Step) (agent.StepResult, error) {
					Expect(s).To(Equal(plan.Steps[0])) // no template here
					return agent.StepResult{Step: s, Outcome: agent.OutcomeOK, Output: "A"}, nil
				}).
				Times(1)

			// Step 2 should arrive rendered.
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ agent.Config, s agent.Step) (agent.StepResult, error) {
					Expect(s.Type).To(Equal(agent.ToolLLM))
					Expect(s.Prompt).To(Equal("use A"))
					return agent.StepResult{Step: s, Outcome: agent.OutcomeOK, Output: "B"}, nil
				}).
				Times(1)

			subject := agent.NewAgent(mockClock, mockPlanner, mockRunner)
			Expect(subject.RunAgentGoal(context.Background(), goal)).To(Succeed())
		})

		it("should error if template rendering fails and not call runner for that step", func() {
			plan := agent.Plan{
				Goal: goal,
				Steps: []agent.Step{
					{Type: agent.ToolLLM, Description: "step 1", Prompt: "ok"},
					{Type: agent.ToolLLM, Description: "step 2", Prompt: "bad {{ .MissingKey }}"},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			// Step 1 runs.
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				Return(agent.StepResult{Step: plan.Steps[0], Outcome: agent.OutcomeOK, Output: "A"}, nil).
				Times(1)

			// Step 2 must NOT run (render should fail before runner call).
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[1]).
				Times(0)

			subject := agent.NewAgent(mockClock, mockPlanner, mockRunner)

			err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).To(HaveOccurred())
		})

		it("should bubble up policy violations (typed) as a stop reason", func() {
			plan := agent.Plan{
				Goal: goal,
				Steps: []agent.Step{
					{Type: agent.ToolShell, Description: "step 1", Command: "echo", Args: []string{"one"}},
					{Type: agent.ToolShell, Description: "step 2", Command: "echo", Args: []string{"two"}},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			polErr := agent.PolicyDeniedError{
				Kind:   "workdir",
				Reason: "workdir not allowed",
			}

			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				Return(agent.StepResult{}, polErr).
				Times(1)

			// Guard: step 2 must not run
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[1]).
				Times(0)

			subject := agent.NewAgent(mockClock, mockPlanner, mockRunner)

			err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).To(MatchError(polErr))
		})

		it("should bubble up budget exceeded errors as a stop reason", func() {
			plan := agent.Plan{
				Goal: goal,
				Steps: []agent.Step{
					{Type: agent.ToolShell, Description: "step 1", Command: "echo", Args: []string{"one"}},
					{Type: agent.ToolShell, Description: "step 2", Command: "echo", Args: []string{"two"}},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			budgetErr := agent.BudgetExceededError{
				Kind:    agent.BudgetKindSteps,
				Limit:   10,
				Used:    10,
				Message: "step budget exceeded",
			}

			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				Return(agent.StepResult{}, budgetErr).
				Times(1)

			// Guard: step 2 must not run
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[1]).
				Times(0)

			subject := agent.NewAgent(mockClock, mockPlanner, mockRunner)

			err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).To(MatchError(budgetErr))
		})
	})
}

func expectAgentDuration(mockClock *MockClock, d time.Duration) {
	t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)
	t1 := t0.Add(d)

	// Robust: first call is t0, all subsequent calls are t1.
	gomock.InOrder(
		mockClock.EXPECT().Now().Return(t0).Times(1),
		mockClock.EXPECT().Now().Return(t1).AnyTimes(),
	)
}
