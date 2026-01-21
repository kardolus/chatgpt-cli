package planexec_test

import (
	"context"
	"fmt"
	"github.com/kardolus/chatgpt-cli/agent/core"
	"github.com/kardolus/chatgpt-cli/agent/planexec"
	"github.com/kardolus/chatgpt-cli/agent/types"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

//go:generate mockgen -destination=runnermocks_test.go -package=planexec_test github.com/kardolus/chatgpt-cli/agent/core Runner
//go:generate mockgen -destination=clockmocks_test.go -package=planexec_test github.com/kardolus/chatgpt-cli/agent/core Clock
//go:generate mockgen -destination=plannermocks_test.go -package=planexec_test github.com/kardolus/chatgpt-cli/agent/planexec Planner

func TestUnitAgent(t *testing.T) {
	spec.Run(t, "Testing the plan-execute agent", testPlanExecuteAgent, spec.Report(report.Terminal{}))
}

func testPlanExecuteAgent(t *testing.T, when spec.G, it spec.S) {
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

		it("should bubble up Planner errors and not run any steps", func() {
			planErr := fmt.Errorf("Planner boom")

			mockPlanner.
				EXPECT().
				Plan(gomock.Any(), goal).
				Return(types.Plan{}, planErr).
				Times(1)

			// Runner must not be invoked if planning fails
			mockRunner.
				EXPECT().
				RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				Times(0)

			subject := planexec.NewPlanExecuteAgent(
				mockClock,
				mockPlanner,
				mockRunner,
			)

			_, err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).To(MatchError(planErr))
		})

		it("should bubble up Runner errors and stop executing further steps", func() {
			runErr := fmt.Errorf("Runner boom")

			plan := types.Plan{
				Goal: goal,
				Steps: []types.Step{
					{Type: types.ToolShell, Description: "step 1", Command: "echo", Args: []string{"one"}},
					{Type: types.ToolShell, Description: "step 2", Command: "echo", Args: []string{"two"}},
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
				Return(types.StepResult{}, runErr).
				Times(1)

			// Guard: step 2 must not run
			mockRunner.
				EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[1]).
				Times(0)

			subject := planexec.NewPlanExecuteAgent(
				mockClock,
				mockPlanner,
				mockRunner,
			)

			_, err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).To(MatchError(runErr))
		})

		it("should return an error when Runner returns OutcomeError (even if err == nil)", func() {
			plan := types.Plan{
				Goal: goal,
				Steps: []types.Step{
					{Type: types.ToolShell, Description: "step 1", Command: "false", Args: nil},
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
				Return(types.StepResult{
					Step:    plan.Steps[0],
					Outcome: types.OutcomeError,
					Exec:    &types.Result{ExitCode: 42}, // optional; Agent no longer inspects this
				}, nil).
				Times(1)

			subject := planexec.NewPlanExecuteAgent(
				mockClock,
				mockPlanner,
				mockRunner,
			)

			_, err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).To(MatchError(`step failed: step 1`))
		})

		it("should succeed when Planner returns an empty plan and not run any steps", func() {
			plan := types.Plan{
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

			subject := planexec.NewPlanExecuteAgent(
				mockClock,
				mockPlanner,
				mockRunner,
			)

			out, err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(BeEmpty())
		})

		it("should stop executing further steps when Runner returns OutcomeError", func() {
			plan := types.Plan{
				Goal: goal,
				Steps: []types.Step{
					{Type: types.ToolShell, Description: "step 1", Command: "false", Args: nil},
					{Type: types.ToolShell, Description: "step 2", Command: "echo", Args: []string{"should-not-run"}},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				Return(types.StepResult{
					Step:    plan.Steps[0],
					Outcome: types.OutcomeError,
					Exec:    &types.Result{ExitCode: 7},
				}, nil).
				Times(1)

			// Guard: step 2 must not run
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[1]).
				Times(0)

			subject := planexec.NewPlanExecuteAgent(
				mockClock,
				mockPlanner,
				mockRunner,
			)

			_, err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).To(MatchError(`step failed: step 1`))
		})

		it("should treat Exec == nil as success and continue to next step", func() {
			plan := types.Plan{
				Goal: goal,
				Steps: []types.Step{
					{Type: types.ToolLLM, Description: "llm step (no exec)", Prompt: "do something"},
					{Type: types.ToolShell, Description: "shell step", Command: "echo", Args: []string{"ok"}},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			// First step: Exec is nil, no error, OutcomeOK => success.
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				Return(types.StepResult{
					Step:    plan.Steps[0],
					Outcome: types.OutcomeOK,
					Exec:    nil,
				}, nil).
				Times(1)

			// Second step should still run.
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[1]).
				Return(types.StepResult{
					Step:    plan.Steps[1],
					Outcome: types.OutcomeOK,
					Exec:    &types.Result{ExitCode: 0},
				}, nil).
				Times(1)

			subject := planexec.NewPlanExecuteAgent(
				mockClock,
				mockPlanner,
				mockRunner,
			)

			_, err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).NotTo(HaveOccurred())
		})

		it("WithWorkDir should pass cfg.WorkDir into the Runner", func() {
			plan := types.Plan{
				Goal: goal,
				Steps: []types.Step{
					{Type: types.ToolShell, Description: "step 1", Command: "echo", Args: []string{"one"}},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				DoAndReturn(func(_ context.Context, cfg types.Config, _ types.Step) (types.StepResult, error) {
					Expect(cfg.WorkDir).To(Equal("/tmp/my-workdir"))
					return types.StepResult{Step: plan.Steps[0], Outcome: types.OutcomeOK}, nil
				}).
				Times(1)

			subject := planexec.NewPlanExecuteAgent(
				mockClock,
				mockPlanner,
				mockRunner,
				core.WithWorkDir("/tmp/my-workdir"),
			)

			_, err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).NotTo(HaveOccurred())
		})

		it("WithDryRun should pass cfg.DryRun into the Runner", func() {
			plan := types.Plan{
				Goal: goal,
				Steps: []types.Step{
					{Type: types.ToolShell, Description: "step 1", Command: "echo", Args: []string{"one"}},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				DoAndReturn(func(_ context.Context, cfg types.Config, _ types.Step) (types.StepResult, error) {
					Expect(cfg.DryRun).To(BeTrue())
					return types.StepResult{Step: plan.Steps[0], Outcome: types.OutcomeDryRun}, nil
				}).
				Times(1)

			subject := planexec.NewPlanExecuteAgent(
				mockClock,
				mockPlanner,
				mockRunner,
				core.WithDryRun(true),
			)

			_, err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).NotTo(HaveOccurred())
		})

		it("happy path: should return the output of the final step", func() {
			const goal = "do the thing"

			plan := types.Plan{
				Goal: goal,
				Steps: []types.Step{
					{Type: types.ToolLLM, Description: "step 1", Prompt: "first"},
					{Type: types.ToolLLM, Description: "step 2", Prompt: "second"},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				Return(types.StepResult{
					Step:    plan.Steps[0],
					Outcome: types.OutcomeOK,
					Output:  "A",
				}, nil).
				Times(1)

			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[1]).
				Return(types.StepResult{
					Step:    plan.Steps[1],
					Outcome: types.OutcomeOK,
					Output:  "B",
				}, nil).
				Times(1)

			subject := planexec.NewPlanExecuteAgent(
				mockClock,
				mockPlanner,
				mockRunner,
			)

			out, err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal("B")) // last step wins
		})

		it("should run all planned steps (PlanExecuteAgent no longer enforces MaxSteps)", func() {
			plan := types.Plan{
				Goal: goal,
				Steps: []types.Step{
					{Type: types.ToolShell, Description: "step 1", Command: "echo", Args: []string{"one"}},
					{Type: types.ToolShell, Description: "step 2", Command: "echo", Args: []string{"two"}},
					{Type: types.ToolShell, Description: "step 3", Command: "echo", Args: []string{"three"}},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			gomock.InOrder(
				mockRunner.EXPECT().
					RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
					Return(types.StepResult{Step: plan.Steps[0], Outcome: types.OutcomeOK}, nil),
				mockRunner.EXPECT().
					RunStep(gomock.Any(), gomock.Any(), plan.Steps[1]).
					Return(types.StepResult{Step: plan.Steps[1], Outcome: types.OutcomeOK}, nil),
				mockRunner.EXPECT().
					RunStep(gomock.Any(), gomock.Any(), plan.Steps[2]).
					Return(types.StepResult{Step: plan.Steps[2], Outcome: types.OutcomeOK}, nil),
			)

			subject := planexec.NewPlanExecuteAgent(
				mockClock,
				mockPlanner,
				mockRunner,
			)

			_, err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).NotTo(HaveOccurred())
		})

		it("should accumulate results and render templates for later steps", func() {
			plan := types.Plan{
				Goal: goal,
				Steps: []types.Step{
					{Type: types.ToolLLM, Description: "step 1", Prompt: "first"},
					{Type: types.ToolLLM, Description: "step 2", Prompt: "use {{ (index .Results 0).Output }}"},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			// Step 1 runs normally.
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ types.Config, s types.Step) (types.StepResult, error) {
					Expect(s).To(Equal(plan.Steps[0])) // no template here
					return types.StepResult{Step: s, Outcome: types.OutcomeOK, Output: "A"}, nil
				}).
				Times(1)

			// Step 2 should arrive rendered.
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ types.Config, s types.Step) (types.StepResult, error) {
					Expect(s.Type).To(Equal(types.ToolLLM))
					Expect(s.Prompt).To(Equal("use A"))
					return types.StepResult{Step: s, Outcome: types.OutcomeOK, Output: "B"}, nil
				}).
				Times(1)

			subject := planexec.NewPlanExecuteAgent(mockClock, mockPlanner, mockRunner)
			_, err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).NotTo(HaveOccurred())
		})

		it("should error if template rendering fails and not call Runner for that step", func() {
			plan := types.Plan{
				Goal: goal,
				Steps: []types.Step{
					{Type: types.ToolLLM, Description: "step 1", Prompt: "ok"},
					{Type: types.ToolLLM, Description: "step 2", Prompt: "bad {{ .MissingKey }}"},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			// Step 1 runs.
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				Return(types.StepResult{Step: plan.Steps[0], Outcome: types.OutcomeOK, Output: "A"}, nil).
				Times(1)

			// Step 2 must NOT run (render should fail before Runner call).
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[1]).
				Times(0)

			subject := planexec.NewPlanExecuteAgent(mockClock, mockPlanner, mockRunner)

			_, err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).To(HaveOccurred())
		})

		it("should bubble up policy violations (typed) as a stop reason", func() {
			plan := types.Plan{
				Goal: goal,
				Steps: []types.Step{
					{Type: types.ToolShell, Description: "step 1", Command: "echo", Args: []string{"one"}},
					{Type: types.ToolShell, Description: "step 2", Command: "echo", Args: []string{"two"}},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			polErr := core.PolicyDeniedError{
				Kind:   "workdir",
				Reason: "workdir not allowed",
			}

			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				Return(types.StepResult{}, polErr).
				Times(1)

			// Guard: step 2 must not run
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[1]).
				Times(0)

			subject := planexec.NewPlanExecuteAgent(mockClock, mockPlanner, mockRunner)

			_, err := subject.RunAgentGoal(context.Background(), goal)
			Expect(err).To(MatchError(polErr))
		})

		it("should bubble up budget exceeded errors as a stop reason", func() {
			plan := types.Plan{
				Goal: goal,
				Steps: []types.Step{
					{Type: types.ToolShell, Description: "step 1", Command: "echo", Args: []string{"one"}},
					{Type: types.ToolShell, Description: "step 2", Command: "echo", Args: []string{"two"}},
				},
			}

			mockPlanner.EXPECT().
				Plan(gomock.Any(), goal).
				Return(plan, nil).
				Times(1)

			budgetErr := core.BudgetExceededError{
				Kind:    core.BudgetKindSteps,
				Limit:   10,
				Used:    10,
				Message: "step budget exceeded",
			}

			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[0]).
				Return(types.StepResult{}, budgetErr).
				Times(1)

			// Guard: step 2 must not run
			mockRunner.EXPECT().
				RunStep(gomock.Any(), gomock.Any(), plan.Steps[1]).
				Times(0)

			subject := planexec.NewPlanExecuteAgent(mockClock, mockPlanner, mockRunner)

			_, err := subject.RunAgentGoal(context.Background(), goal)
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
