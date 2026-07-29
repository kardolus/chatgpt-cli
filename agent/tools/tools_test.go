package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kardolus/chatgpt-cli/agent/tools"
	"github.com/kardolus/chatgpt-cli/internal/fsio"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitTools(t *testing.T) {
	spec.Run(t, "Testing the agent tools", testTools, spec.Report(report.Terminal{}))
}

func testTools(t *testing.T, when spec.G, it spec.S) {
	var ops tools.FSIOFileOps

	it.Before(func() {
		RegisterTestingT(t)
		ops = tools.NewFSIOFileOps(fsio.NewRealReader(fsio.DefaultBufferSize), &fsio.RealWriter{})
	})

	when("FSIOFileOps Write/Read", func() {
		it("round-trips file contents", func() {
			path := filepath.Join(t.TempDir(), "out.txt")
			Expect(ops.WriteFile(path, []byte("hello"))).To(Succeed())

			got, err := ops.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(got)).To(Equal("hello"))
		})

		it("ReadFile errors for a missing file", func() {
			_, err := ops.ReadFile(filepath.Join(t.TempDir(), "nope.txt"))
			Expect(err).To(HaveOccurred())
		})
	})

	when("FSIOFileOps ReplaceBytesInFile", func() {
		write := func(content string) string {
			path := filepath.Join(t.TempDir(), "f.txt")
			Expect(os.WriteFile(path, []byte(content), 0o600)).To(Succeed())
			return path
		}

		it("replaces all occurrences by default (n<=0)", func() {
			path := write("a a a")
			res, err := ops.ReplaceBytesInFile(path, []byte("a"), []byte("b"), 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.OccurrencesFound).To(Equal(3))
			Expect(res.Replaced).To(Equal(3))

			got, _ := os.ReadFile(path)
			Expect(string(got)).To(Equal("b b b"))
		})

		it("replaces only the first n when n>0", func() {
			path := write("a a a")
			res, err := ops.ReplaceBytesInFile(path, []byte("a"), []byte("b"), 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.OccurrencesFound).To(Equal(3))
			Expect(res.Replaced).To(Equal(2))

			got, _ := os.ReadFile(path)
			Expect(string(got)).To(Equal("b b a"))
		})

		it("errors on an empty old pattern", func() {
			path := write("abc")
			_, err := ops.ReplaceBytesInFile(path, []byte(""), []byte("x"), 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("non-empty"))
		})

		it("errors when the pattern is not found", func() {
			path := write("abc")
			res, err := ops.ReplaceBytesInFile(path, []byte("zzz"), []byte("x"), 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
			Expect(res.OccurrencesFound).To(Equal(0))
		})

		it("errors when the replacement produces no change", func() {
			path := write("abc")
			_, err := ops.ReplaceBytesInFile(path, []byte("a"), []byte("a"), 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no changes"))
		})
	})

	when("FSIOFileOps PatchFile", func() {
		it("applies a unified diff and reports hunks", func() {
			path := filepath.Join(t.TempDir(), "f.txt")
			Expect(os.WriteFile(path, []byte("a\nb\n"), 0o600)).To(Succeed())

			diff := []byte("@@ -1,2 +1,3 @@\n a\n+x\n b\n")
			res, err := ops.PatchFile(path, diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Hunks).To(Equal(1))

			got, _ := os.ReadFile(path)
			Expect(string(got)).To(Equal("a\nx\nb\n"))
		})

		it("errors on a malformed diff without touching the file", func() {
			path := filepath.Join(t.TempDir(), "f.txt")
			Expect(os.WriteFile(path, []byte("a\nb\n"), 0o600)).To(Succeed())

			_, err := ops.PatchFile(path, []byte("not a diff"))
			Expect(err).To(HaveOccurred())

			got, _ := os.ReadFile(path)
			Expect(string(got)).To(Equal("a\nb\n")) // unchanged
		})
	})

	when("ExecShellRunner", func() {
		var runner *tools.ExecShellRunner

		it.Before(func() {
			runner = tools.NewExecShellRunner()
			if runtime.GOOS == "windows" {
				t.Skip("POSIX shell assumptions")
			}
		})

		it("captures stdout and a zero exit code", func() {
			res, err := runner.Run(context.Background(), t.TempDir(), "echo", "hi")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Stdout).To(Equal("hi\n"))
			Expect(res.ExitCode).To(Equal(0))
		})

		it("reports a non-zero exit code without returning an error", func() {
			res, err := runner.Run(context.Background(), t.TempDir(), "sh", "-c", "exit 3")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.ExitCode).To(Equal(3))
		})

		it("runs in the given working directory", func() {
			dir := t.TempDir()
			res, err := runner.Run(context.Background(), dir, "pwd")
			Expect(err).NotTo(HaveOccurred())
			// macOS /tmp is a symlink to /private/tmp; compare by suffix.
			Expect(res.Stdout).To(ContainSubstring(filepath.Base(dir)))
		})

		it("honors a cancelled context (kills the process early)", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			start := time.Now()
			// Invoke `sleep` directly (not via `sh -c`): CommandContext then kills
			// the exact process on cancel. Wrapping in a shell can fork a child
			// that keeps the output pipe open, making Wait block the full sleep.
			res, err := runner.Run(ctx, t.TempDir(), "sleep", "5")
			elapsed := time.Since(start)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.ExitCode).NotTo(Equal(0)) // killed, not a clean exit
			// Proves cancellation actually fired rather than waiting out the sleep.
			Expect(elapsed).To(BeNumerically("<", 4*time.Second))
		})
	})
}
