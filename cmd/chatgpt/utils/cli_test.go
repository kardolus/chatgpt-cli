package utils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kardolus/chatgpt-cli/agent/types"
	"github.com/kardolus/chatgpt-cli/cmd/chatgpt/utils"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"gopkg.in/yaml.v3"
)

func TestUnitCLIHelpers(t *testing.T) {
	spec.Run(t, "Testing the CLI helpers", testCLIHelpers, spec.Report(report.Terminal{}))
}

func testCLIHelpers(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("BoolToOnOff()", func() {
		it("maps true/false to ON/OFF", func() {
			Expect(utils.BoolToOnOff(true)).To(Equal("ON"))
			Expect(utils.BoolToOnOff(false)).To(Equal("OFF"))
		})
	})

	when("ToAliasFlagName()", func() {
		it("converts underscores and dots to dashes", func() {
			for in, want := range map[string]string{
				"context_window": "context-window",
				"agent.mode":     "agent-mode",
				"agent.work_dir": "agent-work-dir",
				"model":          "model",
				"max_tokens":     "max-tokens",
				"already-dashed": "already-dashed",
			} {
				Expect(utils.ToAliasFlagName(in)).To(Equal(want), in)
			}
		})
	})

	when("ResolveAgentMode()", func() {
		it("defaults to react when both flag and config are empty", func() {
			mode, err := utils.ResolveAgentMode("", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(mode).To(Equal("react"))
		})

		it("prefers the flag over the config value", func() {
			mode, err := utils.ResolveAgentMode("plan", "react")
			Expect(err).NotTo(HaveOccurred())
			Expect(mode).To(Equal("plan"))
		})

		it("falls back to the config value when the flag is empty", func() {
			mode, err := utils.ResolveAgentMode("", "PLAN")
			Expect(err).NotTo(HaveOccurred())
			Expect(mode).To(Equal("plan"))
		})

		it("normalizes plan_execute / plan-execute aliases to plan", func() {
			for _, in := range []string{"plan_execute", "plan-execute", " Plan-Execute "} {
				mode, err := utils.ResolveAgentMode(in, "")
				Expect(err).NotTo(HaveOccurred(), in)
				Expect(mode).To(Equal("plan"), in)
			}
		})

		it("treats whitespace-only input as empty and defaults to react", func() {
			mode, err := utils.ResolveAgentMode("   ", "\t")
			Expect(err).NotTo(HaveOccurred())
			Expect(mode).To(Equal("react"))
		})

		it("errors on an unknown mode", func() {
			_, err := utils.ResolveAgentMode("banana", "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown agent mode"))
		})
	})

	when("BuildAgentGoal()", func() {
		it("joins piped context and args", func() {
			goal, err := utils.BuildAgentGoal("some context", []string{"do", "the", "thing"})
			Expect(err).NotTo(HaveOccurred())
			Expect(goal).To(Equal("some context\ndo the thing"))
		})

		it("uses only args when there is no context", func() {
			goal, err := utils.BuildAgentGoal("", []string{"just", "args"})
			Expect(err).NotTo(HaveOccurred())
			Expect(goal).To(Equal("just args"))
		})

		it("uses only context when there are no args", func() {
			goal, err := utils.BuildAgentGoal("  ctx only  ", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(goal).To(Equal("ctx only"))
		})

		it("errors when nothing is provided", func() {
			_, err := utils.BuildAgentGoal("   ", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing agent goal"))
		})
	})

	when("ParseToolKinds()", func() {
		it("parses each kind and dedupes, tolerating whitespace/case", func() {
			out, err := utils.ParseToolKinds([]string{"shell", "LLM", " files ", "file", "shell"})
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal([]types.ToolKind{types.ToolShell, types.ToolLLM, types.ToolFiles}))
		})

		it("skips empty entries", func() {
			out, err := utils.ParseToolKinds([]string{"", "shell", ""})
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal([]types.ToolKind{types.ToolShell}))
		})

		it("errors on an unknown entry", func() {
			_, err := utils.ParseToolKinds([]string{"shell", "network"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("network"))
		})

		it("errors when the resulting set is empty", func() {
			_, err := utils.ParseToolKinds([]string{"", "  "})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty"))
		})
	})

	when("flag classification", func() {
		it("IsNonConfigSetter only matches set-completions", func() {
			Expect(utils.IsNonConfigSetter("set-completions")).To(BeTrue())
			Expect(utils.IsNonConfigSetter("set-model")).To(BeFalse())
		})

		it("IsGeneralFlag recognizes action flags but not config aliases", func() {
			Expect(utils.IsGeneralFlag("query")).To(BeTrue())
			Expect(utils.IsGeneralFlag("interactive")).To(BeTrue())
			Expect(utils.IsGeneralFlag("model")).To(BeFalse())
			Expect(utils.IsGeneralFlag("max-tokens")).To(BeFalse())
		})

		it("IsConfigAlias is true only for non-set-, non-general flags", func() {
			Expect(utils.IsConfigAlias("model")).To(BeTrue())
			Expect(utils.IsConfigAlias("max-tokens")).To(BeTrue())
			Expect(utils.IsConfigAlias("set-model")).To(BeFalse()) // a setter
			Expect(utils.IsConfigAlias("query")).To(BeFalse())     // a general flag
		})
	})

	when("MergeMaps()", func() {
		it("copies m2 into m1 with m2 winning on conflict", func() {
			m1 := map[string]interface{}{"a": 1, "b": 2}
			m2 := map[string]interface{}{"b": 20, "c": 3}
			out := utils.MergeMaps(m1, m2)
			Expect(out).To(Equal(map[string]interface{}{"a": 1, "b": 20, "c": 3}))
		})

		it("is a no-op for an empty m2", func() {
			m1 := map[string]interface{}{"a": 1}
			Expect(utils.MergeMaps(m1, map[string]interface{}{})).To(Equal(map[string]interface{}{"a": 1}))
		})

		it("mutates and returns the same m1 (not a copy)", func() {
			m1 := map[string]interface{}{"a": 1}
			out := utils.MergeMaps(m1, map[string]interface{}{"b": 2})
			out["c"] = 3
			Expect(m1).To(HaveKey("c")) // same underlying map
		})
	})

	when("FileExists()", func() {
		it("reports true for an existing file and false otherwise", func() {
			dir := t.TempDir()
			existing := filepath.Join(dir, "there.txt")
			Expect(os.WriteFile(existing, []byte("x"), 0o600)).To(Succeed())

			Expect(utils.FileExists(existing)).To(BeTrue())
			Expect(utils.FileExists(filepath.Join(dir, "nope.txt"))).To(BeFalse())
		})

		it("reports true for an existing directory", func() {
			Expect(utils.FileExists(t.TempDir())).To(BeTrue())
		})
	})

	when("YAML config editing", func() {
		it("round-trips: read, update existing + append new, preserving comments", func() {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			original := "# top comment\nmodel: gpt-4o # inline\nmax_tokens: 100\n"
			Expect(os.WriteFile(path, []byte(original), 0o600)).To(Succeed())

			node, err := utils.ReadConfigWithComments(path)
			Expect(err).NotTo(HaveOccurred())

			err = utils.UpdateConfigNode(node, map[string]interface{}{
				"model":       "gpt-5", // update existing
				"temperature": 0.7,     // append new
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(utils.SaveConfigWithComments(path, node)).To(Succeed())

			raw, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			out := string(raw)
			Expect(out).To(ContainSubstring("# top comment"))    // comment preserved
			Expect(out).To(ContainSubstring("model: gpt-5"))     // updated
			Expect(out).To(ContainSubstring("max_tokens: 100"))  // untouched
			Expect(out).To(ContainSubstring("temperature: 0.7")) // appended

			// And it parses back to the expected values.
			var parsed map[string]interface{}
			Expect(yaml.Unmarshal(raw, &parsed)).To(Succeed())
			Expect(parsed["model"]).To(Equal("gpt-5"))
			Expect(parsed["temperature"]).To(Equal(0.7))
		})

		it("preserves scalar types (int/bool) through a read-modify-write round-trip", func() {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			Expect(os.WriteFile(path, []byte("max_tokens: 100\n"), 0o600)).To(Succeed())

			node, err := utils.ReadConfigWithComments(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(utils.UpdateConfigNode(node, map[string]interface{}{
				"max_tokens":   4096, // int
				"omit_history": true, // bool
			})).To(Succeed())
			Expect(utils.SaveConfigWithComments(path, node)).To(Succeed())

			raw, _ := os.ReadFile(path)
			var parsed map[string]interface{}
			Expect(yaml.Unmarshal(raw, &parsed)).To(Succeed())
			Expect(parsed["max_tokens"]).To(Equal(4096))   // stays int, not "4096"
			Expect(parsed["omit_history"]).To(Equal(true)) // stays bool
		})

		it("UpdateConfigNode initializes an empty document", func() {
			var node yaml.Node
			err := utils.UpdateConfigNode(&node, map[string]interface{}{"model": "gpt-5"})
			Expect(err).NotTo(HaveOccurred())

			out, err := yaml.Marshal(&node)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(ContainSubstring("model: gpt-5"))
		})

		it("ReadConfigWithComments errors on a missing file", func() {
			_, err := utils.ReadConfigWithComments(filepath.Join(t.TempDir(), "absent.yaml"))
			Expect(err).To(HaveOccurred())
		})

		it("UpdateConfigNode errors when the root is not a mapping", func() {
			node := &yaml.Node{
				Kind:    yaml.DocumentNode,
				Content: []*yaml.Node{{Kind: yaml.SequenceNode}},
			}
			err := utils.UpdateConfigNode(node, map[string]interface{}{"a": 1})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mapping node"))
		})
	})
}
