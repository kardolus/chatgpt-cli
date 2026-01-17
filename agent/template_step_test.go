package agent_test

import (
	"testing"

	"github.com/kardolus/chatgpt-cli/agent"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitTemplateStep(t *testing.T) {
	spec.Run(t, "Testing step templating", testTemplateStep, spec.Report(report.Terminal{}))
}

func testTemplateStep(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("ApplyTemplate()", func() {
		it("leaves fields unchanged when there are no template markers", func() {
			step := agent.Step{
				Type:        agent.ToolShell,
				Description: "plain desc",
				Command:     "echo",
				Args:        []string{"hi"},
			}

			ctx := agent.ExecContext{Goal: "g", Plan: agent.Plan{}, Results: nil}

			out, err := agent.ApplyTemplate(step, ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal(step))
		})

		it("renders Description for any step type", func() {
			step := agent.Step{
				Type:        agent.ToolKind("wat"),
				Description: "goal={{.Goal}}",
			}

			ctx := agent.ExecContext{Goal: "ship it"}
			out, err := agent.ApplyTemplate(step, ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(out.Description).To(Equal("goal=ship it"))
			Expect(out.Type).To(Equal(agent.ToolKind("wat")))
		})

		it("renders shell Command and Args", func() {
			step := agent.Step{
				Type:        agent.ToolShell,
				Description: "do {{.Goal}}",
				Command:     "echo {{.Goal}}",
				Args:        []string{"a={{.Goal}}", "b"},
			}

			ctx := agent.ExecContext{Goal: "hello"}
			out, err := agent.ApplyTemplate(step, ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(out.Description).To(Equal("do hello"))
			Expect(out.Command).To(Equal("echo hello"))
			Expect(out.Args).To(Equal([]string{"a=hello", "b"}))
		})

		it("renders llm Prompt", func() {
			step := agent.Step{
				Type:        agent.ToolLLM,
				Description: "ask",
				Prompt:      "say {{.Goal}}",
			}

			ctx := agent.ExecContext{Goal: "hola"}
			out, err := agent.ApplyTemplate(step, ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(out.Prompt).To(Equal("say hola"))
		})

		it("renders file Op/Path/Data", func() {
			step := agent.Step{
				Type:        agent.ToolFiles,
				Description: "write",
				Op:          "write",
				Path:        "/tmp/{{.Goal}}.txt",
				Data:        "payload={{.Goal}}",
			}

			ctx := agent.ExecContext{Goal: "x"}
			out, err := agent.ApplyTemplate(step, ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(out.Path).To(Equal("/tmp/x.txt"))
			Expect(out.Data).To(Equal("payload=x"))
		})

		it("errors when a referenced key is missing (missingkey=error)", func() {
			step := agent.Step{
				Type:        agent.ToolLLM,
				Description: "desc {{.Nope}}",
				Prompt:      "hi",
			}

			ctx := agent.ExecContext{Goal: "g"}
			_, err := agent.ApplyTemplate(step, ctx)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("render Description"))
		})

		it("errors on invalid template syntax and wraps field name", func() {
			step := agent.Step{
				Type:        agent.ToolShell,
				Description: "ok",
				Command:     "{{ .Goal", // missing closing braces
				Args:        []string{"x"},
			}

			ctx := agent.ExecContext{Goal: "g"}
			_, err := agent.ApplyTemplate(step, ctx)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("render Command"))
		})

		it("wraps args index in error message (Args[i])", func() {
			step := agent.Step{
				Type:        agent.ToolShell,
				Description: "ok",
				Command:     "echo",
				Args:        []string{"{{ .Goal", "ok"}, // first arg invalid template
			}

			ctx := agent.ExecContext{Goal: "g"}
			_, err := agent.ApplyTemplate(step, ctx)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("render Args[0]"))
		})
	})
}
