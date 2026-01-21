package planexec_test

import (
	"github.com/kardolus/chatgpt-cli/agent/planexec"
	"github.com/kardolus/chatgpt-cli/agent/types"
	"testing"

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
			step := types.Step{
				Type:        types.ToolShell,
				Description: "plain desc",
				Command:     "echo",
				Args:        []string{"hi"},
			}

			ctx := types.ExecContext{Goal: "g", Plan: types.Plan{}, Results: nil}

			out, err := planexec.ApplyTemplate(step, ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal(step))
		})

		it("renders Description for any step type", func() {
			step := types.Step{
				Type:        types.ToolKind("wat"),
				Description: "goal={{.Goal}}",
			}

			ctx := types.ExecContext{Goal: "ship it"}
			out, err := planexec.ApplyTemplate(step, ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(out.Description).To(Equal("goal=ship it"))
			Expect(out.Type).To(Equal(types.ToolKind("wat")))
		})

		it("renders shell Command and Args", func() {
			step := types.Step{
				Type:        types.ToolShell,
				Description: "do {{.Goal}}",
				Command:     "echo {{.Goal}}",
				Args:        []string{"a={{.Goal}}", "b"},
			}

			ctx := types.ExecContext{Goal: "hello"}
			out, err := planexec.ApplyTemplate(step, ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(out.Description).To(Equal("do hello"))
			Expect(out.Command).To(Equal("echo hello"))
			Expect(out.Args).To(Equal([]string{"a=hello", "b"}))
		})

		it("renders llm Prompt", func() {
			step := types.Step{
				Type:        types.ToolLLM,
				Description: "ask",
				Prompt:      "say {{.Goal}}",
			}

			ctx := types.ExecContext{Goal: "hola"}
			out, err := planexec.ApplyTemplate(step, ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(out.Prompt).To(Equal("say hola"))
		})

		it("renders file Op/Path/Data", func() {
			step := types.Step{
				Type:        types.ToolFiles,
				Description: "write",
				Op:          "write",
				Path:        "/tmp/{{.Goal}}.txt",
				Data:        "payload={{.Goal}}",
			}

			ctx := types.ExecContext{Goal: "x"}
			out, err := planexec.ApplyTemplate(step, ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(out.Path).To(Equal("/tmp/x.txt"))
			Expect(out.Data).To(Equal("payload=x"))
		})

		it("errors when a referenced key is missing (missingkey=error)", func() {
			step := types.Step{
				Type:        types.ToolLLM,
				Description: "desc {{.Nope}}",
				Prompt:      "hi",
			}

			ctx := types.ExecContext{Goal: "g"}
			_, err := planexec.ApplyTemplate(step, ctx)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("render Description"))
		})

		it("errors on invalid template syntax and wraps field name", func() {
			step := types.Step{
				Type:        types.ToolShell,
				Description: "ok",
				Command:     "{{ .Goal", // missing closing braces
				Args:        []string{"x"},
			}

			ctx := types.ExecContext{Goal: "g"}
			_, err := planexec.ApplyTemplate(step, ctx)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("render Command"))
		})

		it("wraps args index in error message (Args[i])", func() {
			step := types.Step{
				Type:        types.ToolShell,
				Description: "ok",
				Command:     "echo",
				Args:        []string{"{{ .Goal", "ok"}, // first arg invalid template
			}

			ctx := types.ExecContext{Goal: "g"}
			_, err := planexec.ApplyTemplate(step, ctx)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("render Args[0]"))
		})
	})
}
