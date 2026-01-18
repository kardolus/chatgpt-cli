# AGENTS.md

This repository contains **ChatGPT CLI**: a multi-provider command-line interface for interacting with LLMs (OpenAI, Azure OpenAI, Perplexity, local models, etc.). It supports streaming + interactive chat, prompt files, image/audio I/O, MCP tool calls, and an experimental multi-step **agent mode** (ReAct and Plan/Execute) with safety/budget controls.

Use this document as the “orientation” for automated agents and contributors.

## Quick repo map

High-signal directories/files:

- `cmd/chatgpt/` — main CLI entrypoint (`main.go`) and CLI resources.
- `internal/` — core application logic (providers, chat, rendering, plumbing).
- `agent/` — experimental agent runtime (ReAct + Plan/Execute), policies, budgets, tool execution.
- `config/` — configuration handling, defaults, and structs.
- `api/` — API surfaces used by the CLI (when applicable).
- `docs/` — additional documentation.
- `scripts/` — helper scripts used for development/release.
- `test/` — test fixtures and integration tests (if present).
- `vendor/` — vendored dependencies.
- `Makefile` — developer commands.
- `README.md` — user-facing documentation (features, install, configuration, agent mode).

## Build, test, and lint

Prefer running the same commands CI runs.

### CI

CI is defined in `.github/workflows/` (notably `test.yml`). When debugging failures, mirror its steps and Go version locally.

### Makefile

If you’re using the Makefile, prefer the targets defined there (they may change over time). Typical workflows are:

- run tests
- run linting/formatting
- build the binary

Use `make help` (if present) or open `Makefile` to see the authoritative list.

### Direct Go commands

- Run unit tests:
  - `go test ./...`
- Build:
  - `go build ./cmd/chatgpt`
- Run locally:
  - `go run ./cmd/chatgpt --help`

## Running the CLI (local dev)

The CLI entrypoint is `cmd/chatgpt/main.go`.

Examples:

- Show help:
  - `go run ./cmd/chatgpt --help`
- Basic query:
  - `go run ./cmd/chatgpt "hello"`
- Interactive mode:
  - `go run ./cmd/chatgpt -i`

Many features depend on provider credentials and config; see **Configuration** in `README.md`.

## Configuration

Configuration is documented in `README.md` under the **Configuration** section. Keep code and docs consistent.

High-level expectations:

- The CLI uses a layered configuration approach (defaults + config file + environment + flags).
- Provider-specific settings exist for OpenAI/Azure/Perplexity/etc.
- Agent-specific settings include:
  - budgets (steps/tool calls/time/tokens)
  - policy rules (allowed tools, denied shell commands, workdir sandboxing)

If you add or change config fields, update:

1. config structs/defaults
2. flag bindings (if applicable)
3. README docs
4. any tests around config parsing/merging

## Agent mode (experimental)

Agent mode is a multi-step automation layer built into the CLI.

Two modes (as described in `README.md`):

- **ReAct** (`--agent` / `--agent-mode react`): iterative think → act → observe loop.
- **Plan/Execute** (`--agent --agent-mode plan`): produces an explicit plan then executes step-by-step.

### Safety model: workdir + policy + budgets

The agent runtime is intended to be safe-by-default:

- **Workdir safety**: file reads/writes can be restricted to a working directory via `--agent-work-dir .`. Attempts to access outside are denied (e.g. `kind=path_escape`).
- **Policy enforcement**: controls which tools and shell commands are allowed.
- **Budgets**: limits steps, tool calls, wall time, and/or tokens.

When making changes in `agent/`, preserve these invariants:

- Don’t broaden default permissions without a clear, reviewed rationale.
- Ensure policy violations are surfaced as structured, actionable errors.
- Avoid logging secrets (API keys, auth headers, prompt contents that may contain credentials).

### Logs

Agent runs write detailed execution logs to the cache directory (see README for the exact path used on each OS/config). If you change logging format or location, update docs and any tests relying on log output.

## MCP support

The CLI can call MCP (Model Context Protocol) tools over HTTP(S) or via STDIO. See `README.md` for:

- required headers/auth
- session management behavior
- how results are injected back into model context

When adjusting MCP behavior, validate:

- backwards compatibility with existing server expectations
- session renewal behavior
- error messages are clear for end users

## Contributing/PR hygiene (agent checklist)

When implementing changes:

1. **Find the right layer**
   - CLI wiring: `cmd/chatgpt/`
   - core logic: `internal/`
   - agent runtime: `agent/`
   - config: `config/`
2. **Add/adjust tests**
   - keep `go test ./...` green.
3. **Update docs**
   - user-facing changes: `README.md`
   - workflow/agent changes: this `AGENTS.md`
4. **Keep diffs tight**
   - avoid unrelated formatting-only changes.

## Common tasks for automated agents

### “Why is CI failing?”

- Run `go test ./...`.
- Compare to `.github/workflows/test.yml`.
- If failures are OS-dependent, reproduce with the same Go version and environment as CI.

### “Where should I add a new CLI flag?”

- Locate the CLI flag/command setup under `cmd/chatgpt/` and trace how it binds into config structs.
- Keep naming consistent with existing flags (`--help`).
- Document the flag in README.

### “Where do I implement a new provider?”

- Start in `internal/` where providers are implemented/selected.
- Ensure config supports the provider (keys, base URL, model names).
- Add tests that stub HTTP and avoid real network calls.

---

If something in this file conflicts with the code, **the code wins**. Please update `AGENTS.md` to reflect current behavior.