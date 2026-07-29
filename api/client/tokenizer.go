package client

import (
	"embed"
	"encoding/base64"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

// Accurate BPE token counting for OpenAI models via tiktoken-go, with the two
// vocabularies we need embedded so there is never a runtime network download
// (tiktoken-go's default loader fetches them from openaipublic.blob...).
//
// Non-OpenAI models (Perplexity, LLaMA, Atlas, ...) don't map to a tiktoken
// encoding, so callers fall back to the char/word heuristic in history.go.

const (
	encodingO200KBase  = "o200k_base"  // gpt-4o / gpt-4.1 / gpt-4.5 / gpt-5 / o-series
	encodingCL100KBase = "cl100k_base" // gpt-4 / gpt-3.5 / text-embedding-*
)

//go:embed encodings/o200k_base.tiktoken encodings/cl100k_base.tiktoken
var encodingsFS embed.FS

// embeddedBPELoader satisfies tiktoken.BpeLoader by serving the vocab from the
// embedded files instead of the network. tiktoken-go hands us the full blob
// URL (e.g. ".../o200k_base.tiktoken"); we resolve it by base filename so an
// encoding we don't embed fails safely (ReadFile errors) rather than silently
// serving the wrong table.
type embeddedBPELoader struct{}

func (embeddedBPELoader) LoadTiktokenBpe(tiktokenBpeFile string) (map[string]int, error) {
	base := path.Base(tiktokenBpeFile) // e.g. "o200k_base.tiktoken"

	contents, err := encodingsFS.ReadFile("encodings/" + base)
	if err != nil {
		return nil, fmt.Errorf("no embedded tiktoken vocab for %q: %w", base, err)
	}

	ranks := make(map[string]int)
	for i, line := range strings.Split(string(contents), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, " ")
		if len(parts) != 2 {
			return nil, fmt.Errorf("malformed tiktoken line %d in %s: %q", i+1, base, line)
		}
		token, err := base64.StdEncoding.DecodeString(parts[0])
		if err != nil {
			return nil, fmt.Errorf("bad base64 on line %d in %s: %w", i+1, base, err)
		}
		rank, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("bad rank on line %d in %s: %w", i+1, base, err)
		}
		ranks[string(token)] = rank
	}
	if len(ranks) == 0 {
		return nil, fmt.Errorf("no ranks loaded from %s", base)
	}
	return ranks, nil
}

// tiktoken-go's loader is a package-global; set it once to our embedded loader.
var setLoaderOnce sync.Once

// encoders caches the built *tiktoken.Tiktoken per encoding name, since
// tiktoken.GetEncoding recompiles the CoreBPE (and its regex) on every call.
var (
	encodersMu sync.Mutex
	encoders   = map[string]*tiktoken.Tiktoken{}
)

// modelEncodingPrefixes maps a model-name prefix to its tiktoken encoding.
// ORDER MATTERS: the first matching prefix wins, so the o200k_base entries
// (gpt-4o, gpt-4.1, ...) must precede the broader cl100k_base "gpt-4" entry.
// tiktoken-go v0.1.8 predates gpt-5 and the o1/o3/o4 series, so we can't defer
// to its EncodingForModel; this table is the single place to add new families.
var modelEncodingPrefixes = []struct {
	prefix   string
	encoding string
}{
	// o200k_base: modern chat + reasoning models
	{"gpt-4o", encodingO200KBase},
	{"chatgpt-4o", encodingO200KBase},
	{"gpt-4.1", encodingO200KBase},
	{"gpt-4.5", encodingO200KBase},
	{"gpt-5", encodingO200KBase},
	{"o1", encodingO200KBase},
	{"o3", encodingO200KBase},
	{"o4", encodingO200KBase},
	// cl100k_base: older gpt-4 / gpt-3.5 families and cl100k embeddings
	{"gpt-4", encodingCL100KBase},
	{"gpt-3.5", encodingCL100KBase},
	{"text-embedding-ada-002", encodingCL100KBase},
	{"text-embedding-3", encodingCL100KBase},
}

// encodingForModel maps an OpenAI model name to its tiktoken encoding. The
// second return is false for models with no known OpenAI encoding (non-OpenAI
// providers, or a model we don't recognize) so the caller can fall back.
func encodingForModel(model string) (string, bool) {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return "", false
	}
	for _, e := range modelEncodingPrefixes {
		if strings.HasPrefix(m, e.prefix) {
			return e.encoding, true
		}
	}
	return "", false
}

// encoderFor returns a cached tiktoken encoder for the given encoding name.
func encoderFor(encoding string) (*tiktoken.Tiktoken, error) {
	setLoaderOnce.Do(func() { tiktoken.SetBpeLoader(embeddedBPELoader{}) })

	encodersMu.Lock()
	defer encodersMu.Unlock()

	if enc, ok := encoders[encoding]; ok {
		return enc, nil
	}
	enc, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		return nil, err
	}
	encoders[encoding] = enc
	return enc, nil
}

// tokenizeCount returns the exact BPE token count of text for the given model.
// ok is false when the model has no known OpenAI encoding (caller should fall
// back to a heuristic) or the encoder failed to load.
func tokenizeCount(model, text string) (int, bool) {
	encoding, ok := encodingForModel(model)
	if !ok {
		return 0, false
	}
	enc, err := encoderFor(encoding)
	if err != nil {
		return 0, false
	}
	return len(enc.EncodeOrdinary(text)), true
}
