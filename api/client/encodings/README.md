# Embedded tiktoken vocabularies

These Byte-Pair-Encoding vocabulary files are embedded (see `../tokenizer.go`)
so token counting works with **zero runtime network access** — tiktoken-go's
default loader would otherwise download them from
`openaipublic.blob.core.windows.net` on first use.

| File | Encoding | Used by |
|------|----------|---------|
| `o200k_base.tiktoken`  | o200k_base  | gpt-4o, gpt-4.1, gpt-4.5, gpt-5, o1/o3/o4 |
| `cl100k_base.tiktoken` | cl100k_base | gpt-4, gpt-3.5, text-embedding-* |

## Provenance

- **Source:** OpenAI `tiktoken` (https://github.com/openai/tiktoken), which
  publishes these tables at
  `https://openaipublic.blob.core.windows.net/encodings/{o200k_base,cl100k_base}.tiktoken`.
- **License:** the tiktoken project is MIT-licensed. The `tiktoken-go-loader`
  package embeds these same tables, establishing precedent for redistribution.
- **Retrieved:** 2026-07-29, unmodified.

Regenerate with:

```sh
curl -sSLO https://openaipublic.blob.core.windows.net/encodings/o200k_base.tiktoken
curl -sSLO https://openaipublic.blob.core.windows.net/encodings/cl100k_base.tiktoken
```
