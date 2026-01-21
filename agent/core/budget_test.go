package core_test

import (
	"errors"
	"github.com/kardolus/chatgpt-cli/agent/core"
	"github.com/kardolus/chatgpt-cli/agent/types"
	"testing"
	"time"

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

			b := core.NewDefaultBudget(core.BudgetLimits{
				MaxSteps: 10,
			})

			step := types.Step{Type: types.ToolShell}

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

			b := core.NewDefaultBudget(core.BudgetLimits{
				MaxSteps: 2,
			})

			step := types.Step{Type: types.ToolShell}

			Expect(b.AllowStep(step, t0)).To(Succeed())
			Expect(b.AllowStep(step, t0)).To(Succeed())

			err := b.AllowStep(step, t0)
			Expect(err).To(HaveOccurred())

			var be core.BudgetExceededError
			Expect(err).To(MatchError(ContainSubstring("step budget exceeded")))
			Expect(err).To(BeAssignableToTypeOf(be))

			var typed core.BudgetExceededError
			errors.As(err, &typed)

			Expect(typed.Kind).To(Equal(core.BudgetKindSteps))
			Expect(typed.Limit).To(Equal(2))
			Expect(typed.Used).To(Equal(2)) // already used before the rejected increment

			s := b.Snapshot(t0)
			Expect(s.StepsUsed).To(Equal(2)) // should not have incremented on failure
			Expect(s.IterationsUsed).To(Equal(0))
		})

		it("auto-starts on AllowTool (ensureStarted) and increments tool counters", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := core.NewDefaultBudget(core.BudgetLimits{
				MaxShellCalls: 10,
			})

			Expect(b.AllowTool(types.ToolShell, t0)).To(Succeed())

			s := b.Snapshot(t0)
			Expect(s.StartedAt).To(Equal(t0))
			Expect(s.ShellUsed).To(Equal(1))
			Expect(s.LLMUsed).To(Equal(0))
			Expect(s.FileOpsUsed).To(Equal(0))
			Expect(s.IterationsUsed).To(Equal(0))
		})

		it("enforces MaxShellCalls / MaxLLMCalls / MaxFileOps independently", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := core.NewDefaultBudget(core.BudgetLimits{
				MaxShellCalls: 1,
				MaxLLMCalls:   2,
				MaxFileOps:    1,
			})

			Expect(b.AllowTool(types.ToolShell, t0)).To(Succeed())
			err := b.AllowTool(types.ToolShell, t0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("shell call budget exceeded"))

			Expect(b.AllowTool(types.ToolLLM, t0)).To(Succeed())
			Expect(b.AllowTool(types.ToolLLM, t0)).To(Succeed())
			err = b.AllowTool(types.ToolLLM, t0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("llm call budget exceeded"))

			Expect(b.AllowTool(types.ToolFiles, t0)).To(Succeed())
			err = b.AllowTool(types.ToolFiles, t0)
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

			b := core.NewDefaultBudget(core.BudgetLimits{})
			err := b.AllowTool("wat", t0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`unknown tool kind`))

			s := b.Snapshot(t0)
			Expect(s.IterationsUsed).To(Equal(0))
		})

		it("enforces MaxWallTime in AllowStep", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)
			tLate := t0.Add(11 * time.Second)

			b := core.NewDefaultBudget(core.BudgetLimits{
				MaxWallTime: 10 * time.Second,
				MaxSteps:    100,
			})

			step := types.Step{Type: types.ToolShell}

			Expect(b.AllowStep(step, t0)).To(Succeed())

			err := b.AllowStep(step, tLate)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("wall time budget exceeded"))

			var typed core.BudgetExceededError
			ok := errors.As(err, &typed)

			Expect(ok).To(BeTrue())
			Expect(typed.Kind).To(Equal(core.BudgetKindWallTime))
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

			b := core.NewDefaultBudget(core.BudgetLimits{
				MaxWallTime:   200 * time.Millisecond,
				MaxShellCalls: 100,
			})

			Expect(b.AllowTool(types.ToolShell, t0)).To(Succeed())

			err := b.AllowTool(types.ToolShell, tLate)
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

			b := core.NewDefaultBudget(core.BudgetLimits{})
			b.Start(t0)

			s0 := b.Snapshot(tBefore)
			Expect(s0.Elapsed).To(Equal(time.Duration(0)))

			s1 := b.Snapshot(tAfter)
			Expect(s1.Elapsed).To(Equal(250 * time.Millisecond))
		})

		it("unknown tool kind starts budget but does not increment counters", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := core.NewDefaultBudget(core.BudgetLimits{})

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

			b := core.NewDefaultBudget(core.BudgetLimits{MaxShellCalls: 1})

			Expect(b.AllowTool(types.ToolShell, t0)).To(Succeed())

			err := b.AllowTool(types.ToolShell, t0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("shell call budget exceeded"))

			s := b.Snapshot(t0)
			Expect(s.ShellUsed).To(Equal(1)) // not 2
			Expect(s.IterationsUsed).To(Equal(0))
		})

		it("ChargeLLMTokens auto-starts and increments LLMTokensUsed", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)

			b := core.NewDefaultBudget(core.BudgetLimits{
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

			b := core.NewDefaultBudget(core.BudgetLimits{
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

			b := core.NewDefaultBudget(core.BudgetLimits{
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

			b := core.NewDefaultBudget(core.BudgetLimits{
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

			b := core.NewDefaultBudget(core.BudgetLimits{
				MaxIterations: 2,
			})

			Expect(b.AllowIteration(t0)).To(Succeed())
			Expect(b.AllowIteration(t0)).To(Succeed())

			err := b.AllowIteration(t0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("iteration budget exceeded"))

			var typed core.BudgetExceededError
			ok := errors.As(err, &typed)
			Expect(ok).To(BeTrue())
			Expect(typed.Kind).To(Equal(core.BudgetKindIterations))
			Expect(typed.Limit).To(Equal(2))
			Expect(typed.Used).To(Equal(2))

			s := b.Snapshot(t0)
			Expect(s.IterationsUsed).To(Equal(2)) // should not increment on failure
		})

		it("does not increment iteration counter on wall-time failure", func() {
			t0 := time.Date(2026, 1, 13, 9, 0, 0, 0, time.UTC)
			tLate := t0.Add(2 * time.Second)

			b := core.NewDefaultBudget(core.BudgetLimits{
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

			b := core.NewDefaultBudget(core.BudgetLimits{
				MaxIterations: 1,
				MaxSteps:      10,
				MaxShellCalls: 10,
			})

			Expect(b.AllowIteration(t0)).To(Succeed())
			Expect(b.AllowStep(types.Step{Type: types.ToolShell}, t0)).To(Succeed())
			Expect(b.AllowTool(types.ToolShell, t0)).To(Succeed())

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
