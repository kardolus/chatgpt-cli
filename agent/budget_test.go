package agent_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kardolus/chatgpt-cli/agent"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitBudget(t *testing.T) {
	spec.Run(t, "Testing the budget", testBudget, spec.Report(report.Terminal{}))
}

func testBudget(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("DefaultBudget", func() {
		it("auto-starts on AllowStep (ensureStarted) and increments steps", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := agent.NewDefaultBudget(agent.BudgetLimits{
				MaxSteps: 10,
			})

			step := agent.Step{Type: agent.ToolShell}

			err := b.AllowStep(step, t0)
			Expect(err).NotTo(HaveOccurred())

			s := b.Snapshot(t0)
			Expect(s.StartedAt).To(Equal(t0))
			Expect(s.Elapsed).To(Equal(time.Duration(0)))
			Expect(s.StepsUsed).To(Equal(1))
			Expect(s.IterationsUsed).To(Equal(0))
		})

		it("enforces MaxSteps", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := agent.NewDefaultBudget(agent.BudgetLimits{
				MaxSteps: 2,
			})

			step := agent.Step{Type: agent.ToolShell}

			Expect(b.AllowStep(step, t0)).To(Succeed())
			Expect(b.AllowStep(step, t0)).To(Succeed())

			err := b.AllowStep(step, t0)
			Expect(err).To(HaveOccurred())

			var be agent.BudgetExceededError
			Expect(err).To(MatchError(ContainSubstring("step budget exceeded")))
			Expect(err).To(BeAssignableToTypeOf(be))

			var typed agent.BudgetExceededError
			errors.As(err, &typed)

			Expect(typed.Kind).To(Equal(agent.BudgetKindSteps))
			Expect(typed.Limit).To(Equal(2))
			Expect(typed.Used).To(Equal(2)) // already used before the rejected increment

			s := b.Snapshot(t0)
			Expect(s.StepsUsed).To(Equal(2)) // should not have incremented on failure
			Expect(s.IterationsUsed).To(Equal(0))
		})

		it("auto-starts on AllowTool (ensureStarted) and increments tool counters", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := agent.NewDefaultBudget(agent.BudgetLimits{
				MaxShellCalls: 10,
			})

			Expect(b.AllowTool(agent.ToolShell, t0)).To(Succeed())

			s := b.Snapshot(t0)
			Expect(s.StartedAt).To(Equal(t0))
			Expect(s.ShellUsed).To(Equal(1))
			Expect(s.LLMUsed).To(Equal(0))
			Expect(s.FileOpsUsed).To(Equal(0))
			Expect(s.IterationsUsed).To(Equal(0))
		})

		it("enforces MaxShellCalls / MaxLLMCalls / MaxFileOps independently", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := agent.NewDefaultBudget(agent.BudgetLimits{
				MaxShellCalls: 1,
				MaxLLMCalls:   2,
				MaxFileOps:    1,
			})

			Expect(b.AllowTool(agent.ToolShell, t0)).To(Succeed())
			err := b.AllowTool(agent.ToolShell, t0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("shell call budget exceeded"))

			Expect(b.AllowTool(agent.ToolLLM, t0)).To(Succeed())
			Expect(b.AllowTool(agent.ToolLLM, t0)).To(Succeed())
			err = b.AllowTool(agent.ToolLLM, t0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("llm call budget exceeded"))

			Expect(b.AllowTool(agent.ToolFiles, t0)).To(Succeed())
			err = b.AllowTool(agent.ToolFiles, t0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("file ops budget exceeded"))

			s := b.Snapshot(t0)
			Expect(s.ShellUsed).To(Equal(1))
			Expect(s.LLMUsed).To(Equal(2))
			Expect(s.FileOpsUsed).To(Equal(1))
			Expect(s.IterationsUsed).To(Equal(0))
		})

		it("returns error for unknown tool kind", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := agent.NewDefaultBudget(agent.BudgetLimits{})
			err := b.AllowTool("wat", t0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`unknown tool kind`))

			s := b.Snapshot(t0)
			Expect(s.IterationsUsed).To(Equal(0))
		})

		it("enforces MaxWallTime in AllowStep", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)
			tLate := t0.Add(11 * time.Second)

			b := agent.NewDefaultBudget(agent.BudgetLimits{
				MaxWallTime: 10 * time.Second,
				MaxSteps:    100,
			})

			step := agent.Step{Type: agent.ToolShell}

			Expect(b.AllowStep(step, t0)).To(Succeed())

			err := b.AllowStep(step, tLate)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("wall time budget exceeded"))

			var typed agent.BudgetExceededError
			ok := errors.As(err, &typed)

			Expect(ok).To(BeTrue())
			Expect(typed.Kind).To(Equal(agent.BudgetKindWallTime))
			Expect(typed.LimitD).To(Equal(10 * time.Second))
			Expect(typed.UsedD).To(Equal(11 * time.Second))

			// step count should NOT increment on wall-time failure
			s := b.Snapshot(tLate)
			Expect(s.StepsUsed).To(Equal(1))
			Expect(s.IterationsUsed).To(Equal(0))
		})

		it("enforces MaxWallTime in AllowTool", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)
			tLate := t0.Add(500 * time.Millisecond)

			b := agent.NewDefaultBudget(agent.BudgetLimits{
				MaxWallTime:   200 * time.Millisecond,
				MaxShellCalls: 100,
			})

			Expect(b.AllowTool(agent.ToolShell, t0)).To(Succeed())

			err := b.AllowTool(agent.ToolShell, tLate)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("wall time budget exceeded"))

			// tool counter should NOT increment on wall-time failure
			s := b.Snapshot(tLate)
			Expect(s.ShellUsed).To(Equal(1))
			Expect(s.IterationsUsed).To(Equal(0))
		})

		it("Snapshot returns elapsed and clamps negative elapsed to 0", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)
			tBefore := t0.Add(-5 * time.Second)
			tAfter := t0.Add(250 * time.Millisecond)

			b := agent.NewDefaultBudget(agent.BudgetLimits{})
			b.Start(t0)

			s0 := b.Snapshot(tBefore)
			Expect(s0.Elapsed).To(Equal(time.Duration(0)))

			s1 := b.Snapshot(tAfter)
			Expect(s1.Elapsed).To(Equal(250 * time.Millisecond))
		})

		it("unknown tool kind starts budget but does not increment counters", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := agent.NewDefaultBudget(agent.BudgetLimits{})

			err := b.AllowTool("wat", t0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`unknown tool kind`))

			s := b.Snapshot(t0)
			Expect(s.StartedAt).To(Equal(t0))
			Expect(s.ShellUsed).To(Equal(0))
			Expect(s.LLMUsed).To(Equal(0))
			Expect(s.FileOpsUsed).To(Equal(0))
			Expect(s.StepsUsed).To(Equal(0))
		})

		it("does not increment tool counter on max tool-call failure", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := agent.NewDefaultBudget(agent.BudgetLimits{MaxShellCalls: 1})

			Expect(b.AllowTool(agent.ToolShell, t0)).To(Succeed())

			err := b.AllowTool(agent.ToolShell, t0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("shell call budget exceeded"))

			s := b.Snapshot(t0)
			Expect(s.ShellUsed).To(Equal(1)) // not 2
			Expect(s.IterationsUsed).To(Equal(0))
		})

		it("ChargeLLMTokens auto-starts and increments LLMTokensUsed", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := agent.NewDefaultBudget(agent.BudgetLimits{
				MaxLLMTokens: 1000,
			})

			b.ChargeLLMTokens(12, t0)

			s := b.Snapshot(t0)
			Expect(s.StartedAt).To(Equal(t0))
			Expect(s.LLMTokensUsed).To(Equal(12))
			Expect(s.StepsUsed).To(Equal(0))
			Expect(s.ShellUsed).To(Equal(0))
			Expect(s.LLMUsed).To(Equal(0))
			Expect(s.FileOpsUsed).To(Equal(0))
			Expect(s.IterationsUsed).To(Equal(0))
		})

		it("ChargeLLMTokens ignores non-positive token charges", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := agent.NewDefaultBudget(agent.BudgetLimits{
				MaxLLMTokens: 1000,
			})

			b.ChargeLLMTokens(0, t0)
			b.ChargeLLMTokens(-5, t0)

			s := b.Snapshot(t0)
			Expect(s.LLMTokensUsed).To(Equal(0))
			Expect(s.IterationsUsed).To(Equal(0))
		})

		it("ChargeLLMTokens accumulates across multiple calls", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := agent.NewDefaultBudget(agent.BudgetLimits{
				MaxLLMTokens: 1000,
			})

			b.ChargeLLMTokens(10, t0)
			b.ChargeLLMTokens(15, t0)

			s := b.Snapshot(t0)
			Expect(s.LLMTokensUsed).To(Equal(25))
			Expect(s.IterationsUsed).To(Equal(0))
		})

		it("auto-starts on AllowIteration and increments iteration counter", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := agent.NewDefaultBudget(agent.BudgetLimits{
				MaxIterations: 10,
			})

			Expect(b.AllowIteration(t0)).To(Succeed())

			s := b.Snapshot(t0)
			Expect(s.StartedAt).To(Equal(t0))
			Expect(s.IterationsUsed).To(Equal(1))
			Expect(s.StepsUsed).To(Equal(0))
			Expect(s.ShellUsed).To(Equal(0))
		})

		it("enforces MaxIterations", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := agent.NewDefaultBudget(agent.BudgetLimits{
				MaxIterations: 2,
			})

			Expect(b.AllowIteration(t0)).To(Succeed())
			Expect(b.AllowIteration(t0)).To(Succeed())

			err := b.AllowIteration(t0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("iteration budget exceeded"))

			var typed agent.BudgetExceededError
			ok := errors.As(err, &typed)
			Expect(ok).To(BeTrue())
			Expect(typed.Kind).To(Equal(agent.BudgetKindIterations))
			Expect(typed.Limit).To(Equal(2))
			Expect(typed.Used).To(Equal(2))

			s := b.Snapshot(t0)
			Expect(s.IterationsUsed).To(Equal(2)) // should not increment on failure
		})

		it("does not increment iteration counter on wall-time failure", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)
			tLate := t0.Add(2 * time.Second)

			b := agent.NewDefaultBudget(agent.BudgetLimits{
				MaxWallTime:   1 * time.Second,
				MaxIterations: 10,
			})

			Expect(b.AllowIteration(t0)).To(Succeed())

			err := b.AllowIteration(tLate)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("wall time budget exceeded"))

			s := b.Snapshot(tLate)
			Expect(s.IterationsUsed).To(Equal(1)) // not 2
		})

		it("iteration budget is independent of steps and tools", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := agent.NewDefaultBudget(agent.BudgetLimits{
				MaxIterations: 1,
				MaxSteps:      10,
				MaxShellCalls: 10,
			})

			Expect(b.AllowIteration(t0)).To(Succeed())
			Expect(b.AllowStep(agent.Step{Type: agent.ToolShell}, t0)).To(Succeed())
			Expect(b.AllowTool(agent.ToolShell, t0)).To(Succeed())

			err := b.AllowIteration(t0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("iteration budget exceeded"))

			s := b.Snapshot(t0)
			Expect(s.IterationsUsed).To(Equal(1))
			Expect(s.StepsUsed).To(Equal(1))
			Expect(s.ShellUsed).To(Equal(1))
		})
	})
}
