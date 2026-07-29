package client

import (
	"testing"

	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/history"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitTokenizer(t *testing.T) {
	spec.Run(t, "Testing the tokenizer", testTokenizer, spec.Report(report.Terminal{}))
}

func testTokenizer(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("encodingForModel()", func() {
		it("maps modern models to o200k_base", func() {
			for _, m := range []string{"gpt-4o", "gpt-4o-2024-08-06", "chatgpt-4o-latest",
				"gpt-4.1", "gpt-4.5-preview", "gpt-5", "gpt-5-mini", "o1", "o1-mini", "o3", "o4-mini"} {
				enc, ok := encodingForModel(m)
				Expect(ok).To(BeTrue(), m)
				Expect(enc).To(Equal(encodingO200KBase), m)
			}
		})

		it("maps the older gpt-4 / gpt-3.5 families to cl100k_base", func() {
			for _, m := range []string{"gpt-4", "gpt-4-0613", "gpt-4-32k", "gpt-3.5-turbo",
				"gpt-3.5-turbo-0125", "text-embedding-ada-002", "text-embedding-3-small"} {
				enc, ok := encodingForModel(m)
				Expect(ok).To(BeTrue(), m)
				Expect(enc).To(Equal(encodingCL100KBase), m)
			}
		})

		it("returns ok=false for non-OpenAI or unknown models", func() {
			for _, m := range []string{"", "llama-3.1-70b", "sonar", "claude-3-5-sonnet",
				"gemini-2.5-pro", "mistral-large"} {
				_, ok := encodingForModel(m)
				Expect(ok).To(BeFalse(), m)
			}
		})
	})

	when("embeddedBPELoader", func() {
		var loader embeddedBPELoader

		it("loads an embedded vocab by base filename", func() {
			ranks, err := loader.LoadTiktokenBpe("https://openaipublic.blob.core.windows.net/encodings/o200k_base.tiktoken")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(ranks)).To(BeNumerically(">", 100000))
		})

		it("fails safely for an encoding we do not embed (no wrong-table fallback)", func() {
			_, err := loader.LoadTiktokenBpe("https://openaipublic.blob.core.windows.net/encodings/p50k_base.tiktoken")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("p50k_base.tiktoken"))
		})
	})

	when("tokenizeCount()", func() {
		it("matches OpenAI's known cl100k_base count", func() {
			// enc.encode("tiktoken is great!") == [83 8251 2488 382 2212 0] -> 6 tokens
			n, ok := tokenizeCount("gpt-4", "tiktoken is great!")
			Expect(ok).To(BeTrue())
			Expect(n).To(Equal(6))
		})

		it("uses a different encoding for gpt-4o than gpt-4 where they diverge", func() {
			// o200k_base handles multilingual text more compactly than cl100k_base.
			const jp = "日本語のテキスト"
			cl, ok1 := tokenizeCount("gpt-4", jp)
			o200, ok2 := tokenizeCount("gpt-4o", jp)
			Expect(ok1).To(BeTrue())
			Expect(ok2).To(BeTrue())
			Expect(cl).To(Equal(8))
			Expect(o200).To(Equal(6))
		})

		it("falls back (ok=false) for models without an OpenAI encoding", func() {
			_, ok := tokenizeCount("llama-3.1-70b", "hello world")
			Expect(ok).To(BeFalse())
			_, ok = tokenizeCount("", "hello world")
			Expect(ok).To(BeFalse())
		})
	})

	when("countTokens()", func() {
		entriesOf := func(contents ...string) []history.History {
			var out []history.History
			for _, c := range contents {
				out = append(out, history.History{Message: api.Message{Role: "user", Content: c}})
			}
			return out
		}

		it("uses tiktoken for OpenAI models (tighter than the heuristic for prose)", func() {
			const text = "The quick brown fox jumps over the lazy dog."
			total, rolling := countTokens(entriesOf(text), "gpt-4o")
			Expect(total).To(Equal(10)) // exact BPE
			Expect(rolling).To(Equal([]int{10}))
			Expect(total).To(BeNumerically("<", heuristicTokenCount(text))) // 10 < 22
		})

		it("falls back to the heuristic for non-OpenAI models", func() {
			const text = "some provider specific model output here"
			total, _ := countTokens(entriesOf(text), "llama-3.1-70b")
			Expect(total).To(Equal(heuristicTokenCount(text)))
		})

		it("tolerates non-string content without panicking", func() {
			entries := []history.History{{Message: api.Message{Role: "user", Content: nil}}}
			total, rolling := countTokens(entries, "gpt-4o")
			Expect(total).To(Equal(0))
			Expect(rolling).To(Equal([]int{0}))
		})

		it("sums per-message counts across the whole history", func() {
			total, rolling := countTokens(entriesOf("hello world", "tiktoken is great!"), "gpt-4")
			Expect(rolling).To(Equal([]int{2, 6}))
			Expect(total).To(Equal(8))
		})
	})
}
