package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kardolus/chatgpt-cli/config"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitReadAPIKeyFile(t *testing.T) {
	spec.Run(t, "ReadAPIKeyFile", testReadAPIKeyFile, spec.Report(report.Terminal{}))
}

func testReadAPIKeyFile(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("the file exists and is small", func() {
		it("returns trimmed contents", func() {
			dir := t.TempDir()
			p := filepath.Join(dir, "key.txt")
			Expect(os.WriteFile(p, []byte("  sk-test-123 \n"), 0o600)).To(Succeed())

			key, err := config.ReadAPIKeyFile(p)

			Expect(err).NotTo(HaveOccurred())
			Expect(key).To(Equal("sk-test-123"))
		})

		it("accepts exactly maxAPIKeyFileBytes bytes", func() {
			dir := t.TempDir()
			p := filepath.Join(dir, "key.txt")

			// Build an exactly-max-sized content with no leading/trailing whitespace,
			// so TrimSpace doesn't change length.
			content := strings.Repeat("a", int(config.MaxAPIKeyFileBytesForTest()))
			Expect(os.WriteFile(p, []byte(content), 0o600)).To(Succeed())

			key, err := config.ReadAPIKeyFile(p)

			Expect(err).NotTo(HaveOccurred())
			Expect(key).To(Equal(content))
		})
	})

	when("the file is empty or whitespace", func() {
		it("returns an 'empty' error for empty file", func() {
			dir := t.TempDir()
			p := filepath.Join(dir, "empty.txt")
			Expect(os.WriteFile(p, []byte(""), 0o600)).To(Succeed())

			_, err := config.ReadAPIKeyFile(p)

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("api key file is empty"))
		})

		it("returns an 'empty' error for whitespace-only file", func() {
			dir := t.TempDir()
			p := filepath.Join(dir, "ws.txt")
			Expect(os.WriteFile(p, []byte(" \n\t  "), 0o600)).To(Succeed())

			_, err := config.ReadAPIKeyFile(p)

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("api key file is empty"))
		})
	})

	when("the file is too large", func() {
		it("fails when size is greater than max", func() {
			dir := t.TempDir()
			p := filepath.Join(dir, "big.txt")

			maxBytes := int(config.MaxAPIKeyFileBytesForTest())
			content := strings.Repeat("a", maxBytes+1)
			Expect(os.WriteFile(p, []byte(content), 0o600)).To(Succeed())

			_, err := config.ReadAPIKeyFile(p)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("api key file too large"))
		})
	})

	when("the path cannot be opened", func() {
		it("returns a wrapped open error", func() {
			_, err := config.ReadAPIKeyFile(filepath.Join(t.TempDir(), "does-not-exist.txt"))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to open api key file"))
			// Avoid asserting exact OS error text.
		})
	})

	when("the path is not a regular file", func() {
		it("returns a 'regular file' error for a directory path", func() {
			dir := t.TempDir()

			_, err := config.ReadAPIKeyFile(dir)

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("api key file must be a regular file"))
		})
	})

	when("path is cleaned", func() {
		it("works with a path containing .. segments", func() {
			dir := t.TempDir()
			sub := filepath.Join(dir, "sub")
			Expect(os.MkdirAll(sub, 0o700)).To(Succeed())

			target := filepath.Join(dir, "key.txt")
			Expect(os.WriteFile(target, []byte("sk-test-abc"), 0o600)).To(Succeed())

			// sub/../key.txt -> key.txt
			p := filepath.Join(sub, "..", "key.txt")

			key, err := config.ReadAPIKeyFile(p)

			Expect(err).NotTo(HaveOccurred())
			Expect(key).To(Equal("sk-test-abc"))
		})
	})
}
