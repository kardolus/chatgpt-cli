package agent_test

import (
	"context"
	"errors"
	"github.com/sclevine/spec/report"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/golang/mock/gomock"
	"github.com/kardolus/chatgpt-cli/agent"
)

func TestUnitPlanner(t *testing.T) {
	spec.Run(t, "Testing the runner", testDefaultPlanner, spec.Report(report.Terminal{}))
}

func testDefaultPlanner(t *testing.T, when spec.G, it spec.S) {
	var (
		ctrl   *gomock.Controller
		llm    *MockLLM
		budget *MockBudget
		clock  *MockClock

		planner *agent.DefaultPlanner
		ctx     context.Context
		now     time.Time
	)

	it.Before(func() {
		RegisterTestingT(t)

		ctrl = gomock.NewController(t)
		llm = NewMockLLM(ctrl)
		budget = NewMockBudget(ctrl)
		clock = NewMockClock(ctrl)

		planner = agent.NewDefaultPlanner(llm, budget, clock)
		ctx = context.Background()
		now = time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	})

	it.After(func() {
		ctrl.Finish()
	})

	when("goal is empty", func() {
		it("returns missing goal and does not call tools", func() {
			_, err := planner.Plan(ctx, "   ")
			Expect(err).To(MatchError("missing goal"))
		})
	})

	when("budget refuses the LLM tool call", func() {
		it("returns the budget error and does not call llm", func() {
			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(agent.BudgetExceededError{
				Kind:    agent.BudgetKindLLM,
				Limit:   1,
				Used:    1,
				Message: "llm call budget exceeded",
			})

			_, err := planner.Plan(ctx, "do something")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("llm call budget exceeded"))
		})
	})

	when("llm returns an error", func() {
		it("returns the llm error", func() {
			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return("", 0, errors.New("boom"))

			_, err := planner.Plan(ctx, "do something")
			Expect(err).To(MatchError("boom"))
		})
	})

	when("llm returns invalid json", func() {
		it("returns parse error", func() {
			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return("not json", 12, nil)

			budget.EXPECT().ChargeLLMTokens(12, now)

			_, err := planner.Plan(ctx, "do something")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse planner JSON"))
		})
	})

	when("llm returns empty string", func() {
		it("returns planner returned empty response", func() {
			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			llm.EXPECT().
				Complete(gomock.Any(), gomock.Any()).
				Return("   \n", 5, nil)

			budget.EXPECT().ChargeLLMTokens(5, now)

			_, err := planner.Plan(ctx, "do something")
			Expect(err).To(MatchError("planner returned empty response"))
		})
	})

	when("json goal is empty", func() {
		it("uses fallback goal passed into parsePlanJSON", func() {
			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			raw := `{
				"goal": "",
				"steps": [
					{
						"type": "shell",
						"description": "List files",
						"command": "ls",
						"args": ["-la"]
					}
				]
			}`

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(raw, 7, nil)
			budget.EXPECT().ChargeLLMTokens(7, now)

			plan, err := planner.Plan(ctx, "fallback-goal")
			Expect(err).NotTo(HaveOccurred())
			Expect(plan.Goal).To(Equal("fallback-goal"))
			Expect(plan.Steps).To(HaveLen(1))
			Expect(plan.Steps[0].Type).To(Equal(agent.ToolShell))
		})
	})

	when("validation fails: missing description", func() {
		it("returns step missing description", func() {
			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			raw := `{
				"goal": "x",
				"steps": [
					{
						"type": "shell",
						"description": "",
						"command": "ls",
						"args": []
					}
				]
			}`

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(raw, 3, nil)
			budget.EXPECT().ChargeLLMTokens(3, now)

			_, err := planner.Plan(ctx, "x")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing description"))
		})
	})

	when("validation fails: unknown type", func() {
		it("returns unknown step type", func() {
			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			raw := `{
				"goal": "x",
				"steps": [
					{
						"type": "wat",
						"description": "???"
					}
				]
			}`

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(raw, 3, nil)
			budget.EXPECT().ChargeLLMTokens(3, now)

			_, err := planner.Plan(ctx, "x")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown step type"))
		})
	})

	when("templates are invalid", func() {
		it("rejects invalid go template syntax", func() {
			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			raw := `{
				"goal": "x",
				"steps": [
					{
						"type": "llm",
						"description": "Bad template",
						"prompt": "hello {{"
					}
				]
			}`

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(raw, 10, nil)
			budget.EXPECT().ChargeLLMTokens(10, now)

			_, err := planner.Plan(ctx, "x")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid template"))
		})

		it("rejects index .Results without a literal index", func() {
			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			// contains "index .Results" but not "(index .Results <number>)"
			raw := `{
				"goal": "x",
				"steps": [
					{
						"type": "llm",
						"description": "Non literal",
						"prompt": "value: {{ index .Results .N }}"
					}
				]
			}`

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(raw, 10, nil)
			budget.EXPECT().ChargeLLMTokens(10, now)

			_, err := planner.Plan(ctx, "x")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("template uses index .Results but not with a literal index"))
		})

		it("rejects reference to future results", func() {
			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			// step 0 references Results[0] -> invalid (must be < stepIndex)
			raw := `{
				"goal": "x",
				"steps": [
					{
						"type": "llm",
						"description": "future",
						"prompt": "Summarize: {{ (index .Results 0).Output }}"
					}
				]
			}`

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(raw, 10, nil)
			budget.EXPECT().ChargeLLMTokens(10, now)

			_, err := planner.Plan(ctx, "x")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("only prior results are available"))
		})
	})

	when("llm returns json wrapped in code fences", func() {
		it("strips code fences and parses successfully", func() {
			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			raw := "```json\n" + `{
          "goal": "x",
          "steps": [
            { "type": "shell", "description": "List", "command": "ls", "args": ["-la"] }
          ]
        }` + "\n```"

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(raw, 7, nil)
			budget.EXPECT().ChargeLLMTokens(7, now)

			plan, err := planner.Plan(ctx, "fallback")
			Expect(err).NotTo(HaveOccurred())
			Expect(plan.Goal).To(Equal("x"))
			Expect(plan.Steps).To(HaveLen(1))
			Expect(plan.Steps[0].Type).To(Equal(agent.ToolShell))
		})
	})

	when("llm returns fenced non-json", func() {
		it("still returns parse error", func() {
			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			raw := "```\nnot json\n```"

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(raw, 3, nil)
			budget.EXPECT().ChargeLLMTokens(3, now)

			_, err := planner.Plan(ctx, "x")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse planner JSON"))
		})
	})

	when("templates are valid", func() {
		it("accepts plans that reference prior results", func() {
			clock.EXPECT().Now().Return(now)
			budget.EXPECT().AllowTool(agent.ToolLLM, now).Return(nil)

			// step 1 references Results[0] -> valid
			raw := `{
				"goal": "x",
				"steps": [
					{
						"type": "shell",
						"description": "Get status",
						"command": "git",
						"args": ["status", "--porcelain"]
					},
					{
						"type": "llm",
						"description": "Summarize",
						"prompt": "Summarize:\n{{ (index .Results 0).Output }}"
					}
				]
			}`

			llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(raw, 10, nil)
			budget.EXPECT().ChargeLLMTokens(10, now)

			plan, err := planner.Plan(ctx, "x")
			Expect(err).NotTo(HaveOccurred())
			Expect(plan.Steps).To(HaveLen(2))
			Expect(plan.Steps[1].Type).To(Equal(agent.ToolLLM))
		})
	})
}
