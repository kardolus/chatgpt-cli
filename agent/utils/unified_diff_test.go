package utils_test

import (
	"github.com/kardolus/chatgpt-cli/agent/utils"
	"testing"

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
			out, err := utils.ApplyUnifiedDiff(orig, nil)
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

			out, err := utils.ApplyUnifiedDiff(orig, diff)
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

			_, err := utils.ApplyUnifiedDiff(orig, diff)
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

			out, err := utils.ApplyUnifiedDiff(orig, diff)
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

			out, err := utils.ApplyUnifiedDiff(orig, diff)
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

			out, err := utils.ApplyUnifiedDiff(orig, diff)
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

			out, err := utils.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("0\na\nB\nc\nz\n"))
		})

		it("errors when hunk starts past EOF", func() {
			orig := []byte("a\n")
			diff := []byte(
				"@@ -10,1 +10,1 @@\n" +
					" a\n",
			)

			_, err := utils.ApplyUnifiedDiff(orig, diff)
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

			_, err := utils.ApplyUnifiedDiff(orig, diff)
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

			out, err := utils.ApplyUnifiedDiff(orig, diff)
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

			out, err := utils.ApplyUnifiedDiff(orig, diff)
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

			_, err := utils.ApplyUnifiedDiff(orig, diff)
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

			_, err := utils.ApplyUnifiedDiff(orig, diff)
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

			out, err := utils.ApplyUnifiedDiff(orig, diff)
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

			_, err := utils.ApplyUnifiedDiff(orig, diff)
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

			out, err := utils.ApplyUnifiedDiff(orig, diff)
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

			_, err := utils.ApplyUnifiedDiff(orig, diff)
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

			_, err := utils.ApplyUnifiedDiff(orig, diff)
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

			out, err := utils.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())

			// With your current parser/applicator behavior, patch lines are newline-terminated,
			// so the output will typically end with '\n'.
			Expect(string(out)).To(Equal("a\nB\n"))
		})

		it("allows trailing whitespace differences for context lines", func() {
			orig := []byte("a\nb\nc\n")
			diff := []byte(
				"@@ -1,3 +1,3 @@\n" +
					" a\n" +
					" b   \n" + // context line with trailing spaces
					"-c\n" +
					"+C\n",
			)

			out, err := utils.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("a\nb\nC\n"))
		})

		it("still requires exact match for deletions (whitespace mismatch fails)", func() {
			orig := []byte("a\nb\n")
			diff := []byte(
				"@@ -1,2 +1,1 @@\n" +
					" a\n" +
					"-b   \n", // deletion line has extra spaces; should NOT match "b\n"
			)

			_, err := utils.ApplyUnifiedDiff(orig, diff)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("patch deletion mismatch"))
		})

		it("fuzzy placement chooses the closest match when context appears multiple times", func() {
			orig := []byte(
				"header\n" +
					"a\n" +
					"b\n" +
					"c\n" +
					"mid\n" +
					"a\n" +
					"b\n" +
					"c\n" +
					"footer\n",
			)

			// The context block "a\nb\nc\n" appears twice.
			// We lie in the header and point near the SECOND occurrence, and expect it to patch the second.
			diff := []byte(
				"@@ -6,3 +6,3 @@\n" +
					" a\n" +
					"-b\n" +
					"+B\n" +
					" c\n",
			)

			out, err := utils.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(out)).To(Equal(
				"header\n" +
					"a\n" +
					"b\n" +
					"c\n" +
					"mid\n" +
					"a\n" +
					"B\n" +
					"c\n" +
					"footer\n",
			))
		})

		it("fuzzy-applies when hunk header oldStart is wrong but context matches", func() {
			orig := []byte(" roses are red\nviolets are blue\n sugar is sweet\nand so are you\n")

			// Wrong oldStart on purpose (says start at line 2, but the hunk matches starting at line 1).
			diff := []byte(
				"@@ -2,4 +2,4 @@\n" +
					"  roses are red\n" + // TWO spaces: ' ' prefix + content-leading space
					" violets are blue\n" + // ONE space: ' ' prefix; content starts with 'v'
					"- sugar is sweet\n" +
					"+ sugar is SWEET\n" +
					" and so are you\n", // ONE space: ' ' prefix; content starts with 'a'
			)

			hunks, err := utils.ParseUnifiedDiff(diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(hunks).To(HaveLen(1))

			out, err := utils.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal(" roses are red\nviolets are blue\n sugar is SWEET\nand so are you\n"))
		})

		it("fuzzy is required when header points to wrong place but match exists elsewhere", func() {
			// Make file long enough so oldStart=20 is within EOF.
			orig := []byte(
				" roses are red\n" +
					"violets are blue\n" +
					" sugar is sweet\n" +
					"and so are you\n" +
					"pad1\npad2\npad3\npad4\npad5\npad6\npad7\npad8\npad9\npad10\npad11\npad12\npad13\npad14\npad15\npad16\n",
			)

			// oldStart is wrong: points at "pad..." not the poem.
			diff := []byte(
				"@@ -20,4 +20,4 @@\n" +
					"  roses are red\n" + // ' ' prefix + leading space in content
					" violets are blue\n" + // ' ' prefix only
					"- sugar is sweet\n" +
					"+ sugar is SWEET\n" +
					" and so are you\n",
			)

			out, err := utils.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(out)).To(Equal(
				" roses are red\n" +
					"violets are blue\n" +
					" sugar is SWEET\n" +
					"and so are you\n" +
					"pad1\npad2\npad3\npad4\npad5\npad6\npad7\npad8\npad9\npad10\npad11\npad12\npad13\npad14\npad15\npad16\n",
			))
		})

		it("fuzzy placement chooses the closest match when context appears multiple times (forced fuzzy)", func() {
			orig := []byte(
				"header\n" +
					"a\n" +
					"b\n" +
					"c\n" +
					"mid\n" +
					"a\n" +
					"b\n" +
					"c\n" +
					"footer\n",
			)

			// oldStart=5 -> preferredIdx=4 points at "mid\n" (cannot match "a\n")
			// Both occurrences match; closest to line 5 is the SECOND block.
			diff := []byte(
				"@@ -5,3 +5,3 @@\n" +
					" a\n" +
					"-b\n" +
					"+B\n" +
					" c\n",
			)

			out, err := utils.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(out)).To(Equal(
				"header\n" +
					"a\n" +
					"b\n" +
					"c\n" +
					"mid\n" +
					"a\n" +
					"B\n" +
					"c\n" +
					"footer\n",
			))
		})

		it("fuzzy placement never matches before the current origIdx (ordering constraint)", func() {
			orig := []byte(
				"a\n" +
					"b\n" +
					"c\n" +
					"X\n" +
					"a\n" +
					"b\n" +
					"c\n",
			)

			// Hunk1 replaces X -> Y (consumes through line 4)
			// Hunk2 wants to patch the "a b c" block.
			// We lie and point header near the FIRST block, but origIdx is already past it.
			diff := []byte(
				"@@ -4,1 +4,1 @@\n" +
					"-X\n" +
					"+Y\n" +
					"@@ -1,3 +1,3 @@\n" + // malicious/wrong header: points to first block
					" a\n" +
					"-b\n" +
					"+B\n" +
					" c\n",
			)

			out, err := utils.ApplyUnifiedDiff(orig, diff)
			Expect(err).NotTo(HaveOccurred())

			// It must patch the SECOND block (lines 5-7), not the first.
			Expect(string(out)).To(Equal(
				"a\n" +
					"b\n" +
					"c\n" +
					"Y\n" +
					"a\n" +
					"B\n" +
					"c\n",
			))
		})
	})
}
