package agent_test

import (
	"testing"

	"github.com/kardolus/chatgpt-cli/agent"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitUnifiedDiff(t *testing.T) {
	spec.Run(t, "UnifiedDiff", testUnifiedDiff, spec.Report(report.Terminal{}))
}

func testUnifiedDiff(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("ApplyUnifiedDiff", func() {
		it("is a no-op for empty diff", func() {
			orig := []byte("a\nb\n")
			out, err := agent.ApplyUnifiedDiff(orig, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal(orig))
		})

		it("applies an insertion", func() {
			orig := []byte("a\nb\n")
			diff := []byte(
				"@@ -1,2 +1,3 @@\n" +
					" a\n" +
					"+x\n" +
					" b\n",
			)

			out, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("a\nx\nb\n"))
		})

		it("errors on context mismatch", func() {
			orig := []byte("a\nc\n")
			diff := []byte(
				"@@ -1,2 +1,2 @@\n" +
					" a\n" +
					" b\n",
			)

			_, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("patch context mismatch"))
		})

		it("applies a deletion", func() {
			orig := []byte("a\nb\nc\n")
			diff := []byte(
				"@@ -1,3 +1,2 @@\n" +
					" a\n" +
					"-b\n" +
					" c\n",
			)

			out, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("a\nc\n"))
		})

		it("applies a replace (delete + insert)", func() {
			orig := []byte("a\nb\nc\n")
			diff := []byte(
				"@@ -1,3 +1,3 @@\n" +
					" a\n" +
					"-b\n" +
					"+B\n" +
					" c\n",
			)

			out, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("a\nB\nc\n"))
		})

		it("applies multiple hunks in one patch", func() {
			orig := []byte("a\nb\nc\nd\ne\n")
			diff := []byte(
				"@@ -1,3 +1,3 @@\n" +
					" a\n" +
					"-b\n" +
					"+B\n" +
					" c\n" +
					"@@ -4,2 +4,2 @@\n" +
					" d\n" +
					"-e\n" +
					"+E\n",
			)

			out, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("a\nB\nc\nd\nE\n"))
		})

		it("keeps untouched lines before and after hunks", func() {
			orig := []byte("0\na\nb\nc\nz\n")
			diff := []byte(
				"@@ -2,3 +2,3 @@\n" +
					" a\n" +
					"-b\n" +
					"+B\n" +
					" c\n",
			)

			out, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("0\na\nB\nc\nz\n"))
		})

		it("errors when hunk starts past EOF", func() {
			orig := []byte("a\n")
			diff := []byte(
				"@@ -10,1 +10,1 @@\n" +
					" a\n",
			)

			_, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("hunk starts past EOF"))
		})

		it("errors on overlapping or out-of-order hunks", func() {
			orig := []byte("a\nb\nc\nd\n")
			diff := []byte(
				"@@ -3,1 +3,1 @@\n" +
					"-c\n" +
					"+C\n" +
					"@@ -2,1 +2,1 @@\n" +
					"-b\n" +
					"+B\n",
			)

			_, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("overlapping or out-of-order hunks"))
		})

		it("ignores typical diff headers (diff/index/---/+++)", func() {
			orig := []byte("a\nb\n")
			diff := []byte(
				"diff --git a/file.txt b/file.txt\n" +
					"index 123..456 100644\n" +
					"--- a/file.txt\n" +
					"+++ b/file.txt\n" +
					"@@ -1,2 +1,2 @@\n" +
					" a\n" +
					"-b\n" +
					"+B\n",
			)

			out, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("a\nB\n"))
		})

		it("allows whitespace-only noise before first hunk", func() {
			orig := []byte("a\nb\n")
			diff := []byte(
				"\n\n   \n" +
					"@@ -1,2 +1,2 @@\n" +
					" a\n" +
					"-b\n" +
					"+B\n",
			)

			out, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("a\nB\n"))
		})

		it("errors if non-whitespace content appears before the first hunk (strict)", func() {
			orig := []byte("a\nb\n")
			diff := []byte(
				"THIS IS NOT A HEADER\n" +
					"@@ -1,2 +1,2 @@\n" +
					" a\n" +
					"-b\n" +
					"+B\n",
			)

			_, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing hunk header"))
		})

		it("errors on an invalid diff line prefix", func() {
			orig := []byte("a\nb\n")
			diff := []byte(
				"@@ -1,2 +1,2 @@\n" +
					" a\n" +
					"!b\n",
			)

			_, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid diff line prefix"))
		})

		it(`honors "\ No newline at end of file" marker lines`, func() {
			orig := []byte("a\nb\n")
			diff := []byte(
				"@@ -1,2 +1,2 @@\n" +
					" a\n" +
					"-b\n" +
					"+B\n" +
					`\ No newline at end of file` + "\n",
			)

			out, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("a\nB"))
		})

		it("errors when diff contains an empty line without a prefix inside a hunk", func() {
			orig := []byte("a\nb\n")

			// NOTE: Scanner yields line="" for the blank line between "\n\n"
			// which your parser rejects once cur != nil.
			diff := []byte(
				"@@ -1,2 +1,2 @@\n" +
					" a\n" +
					"\n" + // invalid: empty line without ' ', '+', '-'
					" b\n",
			)

			_, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty line without prefix"))
		})

		it("handles inserts at the beginning of the file (oldStart=1)", func() {
			orig := []byte("a\nb\n")
			diff := []byte(
				"@@ -1,2 +1,3 @@\n" +
					"+X\n" +
					" a\n" +
					" b\n",
			)

			out, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("X\na\nb\n"))
		})

		it("errors when patch context extends past EOF", func() {
			orig := []byte("a\n")
			diff := []byte(
				"@@ -1,1 +1,2 @@\n" +
					" a\n" +
					" b\n", // context requires a second line that doesn't exist
			)

			_, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("patch context extends past EOF"))
		})

		it("errors when patch deletion extends past EOF", func() {
			orig := []byte("a\n")
			diff := []byte(
				"@@ -1,1 +1,0 @@\n" +
					"-a\n" +
					"-b\n", // deletion for line that doesn't exist
			)

			_, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("patch deletion extends past EOF"))
		})

		it("supports last line without trailing newline in original", func() {
			orig := []byte("a\nb") // no trailing '\n' on last line
			diff := []byte(
				"@@ -1,2 +1,2 @@\n" +
					" a\n" +
					"-b\n" +
					"+B\n",
			)

			out, err := agent.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())

			// With your current parser/applicator behavior, patch lines are newline-terminated,
			// so the output will typically end with '\n'.
			Expect(string(out)).To(Equal("a\nB\n"))
		})
	})
}
