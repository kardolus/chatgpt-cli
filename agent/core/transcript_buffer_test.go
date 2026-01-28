package core_test

import (
	"strings"
	"testing"

	"github.com/kardolus/chatgpt-cli/agent/core"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestTranscriptBuffer(t *testing.T) {
	spec.Run(t, "TranscriptBuffer", testTranscriptBuffer, spec.Report(report.Terminal{}))
}

func testTranscriptBuffer(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("NewTranscriptBuffer(maxBytes)", func() {
		it("clamps negative maxBytes to 0 (buffer stays empty)", func() {
			tb := core.NewTranscriptBuffer(-1)
			tb.AppendString("hello")
			Expect(tb.Len()).To(Equal(0))
			Expect(tb.String()).To(Equal(""))
		})

		it("uses 0 maxBytes to discard all content", func() {
			tb := core.NewTranscriptBuffer(0)
			tb.AppendString("hello")
			Expect(tb.Len()).To(Equal(0))
			Expect(tb.String()).To(Equal(""))
		})
	})

	when("AppendString()", func() {
		it("normalizes entries to end with newline", func() {
			tb := core.NewTranscriptBuffer(1024)
			tb.AppendString("hello")
			Expect(tb.String()).To(Equal("hello\n"))
		})

		it("does not double-add newline if already present", func() {
			tb := core.NewTranscriptBuffer(1024)
			tb.AppendString("hello\n")
			Expect(tb.String()).To(Equal("hello\n"))
		})

		it("is a no-op for empty string", func() {
			tb := core.NewTranscriptBuffer(1024)
			tb.AppendString("")
			Expect(tb.Len()).To(Equal(0))
		})
	})

	when("Appendf()", func() {
		it("formats and appends (with newline normalization)", func() {
			tb := core.NewTranscriptBuffer(1024)
			tb.Appendf("x=%d", 7)
			Expect(tb.String()).To(Equal("x=7\n"))
		})
	})

	when("truncation / cap behavior", func() {
		it("caps total length and prepends the truncation banner when there is room", func() {
			// Make max large enough to include the banner + some content.
			tb := core.NewTranscriptBuffer(40)

			// Force overflow: two appends produce more than 40 bytes.
			tb.AppendString("0123456789")     // 11 incl newline
			tb.AppendString("ABCDEFGHIJKLMN") // 15 incl newline => total 26, still under
			tb.AppendString(strings.Repeat("Z", 50))

			s := tb.String()

			// Must be capped
			Expect(tb.Len()).To(BeNumerically("<=", 40))

			// Banner should appear and should be at the beginning (it’s prepended)
			Expect(s).To(HavePrefix("\n…(truncated)\n"))

			// Only once as prefix
			Expect(strings.Count(s, "…(truncated)")).To(Equal(1))
		})

		it("does not add the banner when maxBytes is too small to fit it", func() {
			// Banner length is > 5, so condition len(banner) < max is false.
			tb := core.NewTranscriptBuffer(5)

			tb.AppendString("hello")
			tb.AppendString("world") // overflow

			s := tb.String()
			Expect(tb.Len()).To(BeNumerically("<=", 5))
			Expect(s).NotTo(ContainSubstring("…(truncated)"))
		})

		it("does not repeatedly prepend the banner across multiple truncations", func() {
			tb := core.NewTranscriptBuffer(50)

			// First truncation
			tb.AppendString(strings.Repeat("A", 200))
			s1 := tb.String()
			Expect(s1).To(HavePrefix("\n…(truncated)\n"))
			Expect(strings.Count(s1, "…(truncated)")).To(Equal(1))

			// Second truncation (append more, should still be exactly one banner prefix)
			tb.AppendString(strings.Repeat("B", 200))
			s2 := tb.String()
			Expect(tb.Len()).To(BeNumerically("<=", 50))
			Expect(s2).To(HavePrefix("\n…(truncated)\n"))
			Expect(strings.Count(s2, "…(truncated)")).To(Equal(1))
		})
	})

	when("Reset()", func() {
		it("clears the buffer", func() {
			tb := core.NewTranscriptBuffer(1024)
			tb.AppendString("hello")
			Expect(tb.Len()).To(BeNumerically(">", 0))

			tb.Reset()
			Expect(tb.Len()).To(Equal(0))
			Expect(tb.String()).To(Equal(""))
		})
	})

	when("nil receiver safety", func() {
		it("does not panic and returns empty/zero values", func() {
			var tb *core.TranscriptBuffer

			Expect(func() { tb.AppendString("hello") }).NotTo(Panic())
			Expect(func() { tb.Appendf("x=%d", 1) }).NotTo(Panic())
			Expect(func() { _ = tb.String() }).NotTo(Panic())
			Expect(func() { _ = tb.Len() }).NotTo(Panic())
			Expect(func() { tb.Reset() }).NotTo(Panic())

			Expect(tb.String()).To(Equal(""))
			Expect(tb.Len()).To(Equal(0))
		})
	})
}
