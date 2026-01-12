package agent_test

import (
	"testing"

	"github.com/kardolus/chatgpt-cli/agent"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitPolicy(t *testing.T) {
	spec.Run(t, "Testing policy", testPolicy, spec.Report(report.Terminal{}))
}

func testPolicy(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("DefaultPolicy.AllowStep()", func() {
		it("denies unsupported step types", func() {
			p := agent.NewDefaultPolicy(agent.PolicyLimits{})

			err := p.AllowStep(agent.Config{WorkDir: "/tmp"}, agent.Step{
				Type:        agent.ToolKind("wat"),
				Description: "unknown",
			})

			Expect(err).To(HaveOccurred())
			var pe agent.PolicyDeniedError
			Expect(err).To(BeAssignableToTypeOf(pe))
			Expect(err.Error()).To(ContainSubstring("policy denied"))
			Expect(err.Error()).To(ContainSubstring("unsupported step type"))
		})

		it("enforces AllowedTools allowlist when set", func() {
			p := agent.NewDefaultPolicy(agent.PolicyLimits{
				AllowedTools: []agent.ToolKind{agent.ToolShell},
			})

			// shell is allowed
			Expect(p.AllowStep(agent.Config{WorkDir: "/tmp"}, agent.Step{
				Type:    agent.ToolShell,
				Command: "echo",
				Args:    []string{"hi"},
			})).To(Succeed())

			// llm is denied
			err := p.AllowStep(agent.Config{WorkDir: "/tmp"}, agent.Step{
				Type:   agent.ToolLLM,
				Prompt: "hello",
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("tool not allowed"))
			Expect(err.Error()).To(ContainSubstring(string(agent.ToolLLM)))
		})

		when("shell steps", func() {
			it("denies missing/blank Command", func() {
				p := agent.NewDefaultPolicy(agent.PolicyLimits{})

				err := p.AllowStep(agent.Config{WorkDir: "/tmp"}, agent.Step{
					Type:    agent.ToolShell,
					Command: "   ",
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("shell step requires Command"))
			})

			it("denies shell commands present in DeniedShellCommands", func() {
				p := agent.NewDefaultPolicy(agent.PolicyLimits{
					DeniedShellCommands: []string{"rm", "sudo"},
				})

				err := p.AllowStep(agent.Config{WorkDir: "/tmp"}, agent.Step{
					Type:    agent.ToolShell,
					Command: "rm",
					Args:    []string{"-rf", "/"},
				})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("shell command denied"))
				Expect(err.Error()).To(ContainSubstring("rm"))
			})

			it("allows shell commands not in denylist", func() {
				p := agent.NewDefaultPolicy(agent.PolicyLimits{
					DeniedShellCommands: []string{"rm"},
				})

				Expect(p.AllowStep(agent.Config{WorkDir: "/tmp"}, agent.Step{
					Type:    agent.ToolShell,
					Command: "echo",
					Args:    []string{"ok"},
				})).To(Succeed())
			})
		})

		when("llm steps", func() {
			it("denies missing/blank Prompt", func() {
				p := agent.NewDefaultPolicy(agent.PolicyLimits{})

				err := p.AllowStep(agent.Config{}, agent.Step{
					Type:   agent.ToolLLM,
					Prompt: " \n\t",
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("llm step requires Prompt"))
			})

			it("allows non-empty Prompt", func() {
				p := agent.NewDefaultPolicy(agent.PolicyLimits{})

				Expect(p.AllowStep(agent.Config{}, agent.Step{
					Type:   agent.ToolLLM,
					Prompt: "say hi",
				})).To(Succeed())
			})
		})

		when("file steps", func() {
			it("denies missing Op", func() {
				p := agent.NewDefaultPolicy(agent.PolicyLimits{})

				err := p.AllowStep(agent.Config{WorkDir: "/tmp"}, agent.Step{
					Type: agent.ToolFiles,
					Op:   "   ",
					Path: "a.txt",
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("file step requires Op"))
			})

			it("denies missing Path", func() {
				p := agent.NewDefaultPolicy(agent.PolicyLimits{})

				err := p.AllowStep(agent.Config{WorkDir: "/tmp"}, agent.Step{
					Type: agent.ToolFiles,
					Op:   "read",
					Path: "   ",
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("file step requires Path"))
			})

			it("enforces AllowedFileOps (case/whitespace-normalized)", func() {
				p := agent.NewDefaultPolicy(agent.PolicyLimits{
					AllowedFileOps: []string{"read"},
				})

				// "ReAd" should be treated as "read"
				Expect(p.AllowStep(agent.Config{WorkDir: "/tmp"}, agent.Step{
					Type: agent.ToolFiles,
					Op:   "  ReAd  ",
					Path: "a.txt",
				})).To(Succeed())

				// write is denied
				err := p.AllowStep(agent.Config{WorkDir: "/tmp"}, agent.Step{
					Type: agent.ToolFiles,
					Op:   "write",
					Path: "a.txt",
					Data: "x",
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("file op not allowed"))
				Expect(err.Error()).To(ContainSubstring("write"))
			})

			it("restricts file paths to WorkDir when RestrictFilesToWorkDir is enabled (relative escape)", func() {
				p := agent.NewDefaultPolicy(agent.PolicyLimits{
					RestrictFilesToWorkDir: true,
				})

				err := p.AllowStep(agent.Config{WorkDir: "/repo"}, agent.Step{
					Type: agent.ToolFiles,
					Op:   "read",
					Path: "../etc/passwd",
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("path escapes workdir"))
			})

			it("restricts file paths to WorkDir when RestrictFilesToWorkDir is enabled (absolute escape)", func() {
				p := agent.NewDefaultPolicy(agent.PolicyLimits{
					RestrictFilesToWorkDir: true,
				})

				err := p.AllowStep(agent.Config{WorkDir: "/repo"}, agent.Step{
					Type: agent.ToolFiles,
					Op:   "read",
					Path: "/etc/passwd",
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("path escapes workdir"))
			})

			it("allows paths inside WorkDir when RestrictFilesToWorkDir is enabled", func() {
				p := agent.NewDefaultPolicy(agent.PolicyLimits{
					RestrictFilesToWorkDir: true,
				})

				// relative inside
				Expect(p.AllowStep(agent.Config{WorkDir: "/repo"}, agent.Step{
					Type: agent.ToolFiles,
					Op:   "read",
					Path: "dir/file.txt",
				})).To(Succeed())

				// absolute inside
				Expect(p.AllowStep(agent.Config{WorkDir: "/repo"}, agent.Step{
					Type: agent.ToolFiles,
					Op:   "read",
					Path: "/repo/dir/file.txt",
				})).To(Succeed())

				// exactly the workdir itself should not count as escape
				Expect(p.AllowStep(agent.Config{WorkDir: "/repo"}, agent.Step{
					Type: agent.ToolFiles,
					Op:   "read",
					Path: "/repo",
				})).To(Succeed())
			})
		})
	})
}
