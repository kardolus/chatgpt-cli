# AGENTS.md

This repository is a Go-based CLI application (module: `github.com/kardolus/chatgpt-cli`) built with Cobra and Viper.  
Use this document as the default operating context when acting as an agent in this repo.

## High-level purpose

- **Project type**: Go command-line tool.
- **Entry points**: `cmd/` (Cobra commands), with core logic under `internal/`, `api/`, and `agent/`.
- **Configuration**: managed via Viper (YAML/TOML/ENV), with defaults/templates under `config/`.

## Repository layout (key paths)

- `cmd/`: CLI command definitions and wiring (Cobra).
- `internal/`: non-exported application logic.
- `api/`: API clients and domain types.
- `agent/`: agent logic (prompting, tools, orchestration, budgets, policies).
- `config/`: config defaults and templates.
- `docs/`: documentation.
- `scripts/`: dev and CI helper scripts.
- `test/`: tests and fixtures.
- `vendor/`: vendored dependencies (read-only unless explicitly asked).

## How to work in this repo

### Prerequisites

- **Go version**: ≥ 1.24.1 (as specified in `go.mod`).
    - Go is generally backward compatible, but do not use features newer than this version unless the module version is
      bumped.

### Prefer Makefile targets (repo-standard)

If a `Makefile` target exists for a task, use it instead of ad-hoc `go ...` commands. The `make` targets wrap the repo’s
intended scripts/workflows.

Start by listing available targets:

- `make help`

## Mandatory testing rule

Before considering any task “done”, you **MUST** run:

- `make unit`
- `make integration`

Both must pass with no failures.  
Do not skip integration tests unless explicitly instructed.

## Tests are mandatory

For any behavior change or new feature, you **MUST** add or update tests that prove the behavior.

- Follow the repo’s existing style: `sclevine/spec` + `gomega`.
- Tests should cover:
    - The “happy path”
    - Validation and error paths
    - Invariants (e.g., counters don’t increment on failure)
    - Typed errors via `errors.As` when applicable

Use existing tests like `agent/budget_test.go` as the canonical style reference for:

- `when` / `it` structure
- Gomega assertions
- Error typing and invariants

If you implement new functionality, add at least:

- 1 test for the primary expected behavior
- 1 test for the main failure mode / validation
- 1 test for an important invariant (no partial state updates on failure)

## Common Make targets

- Full suite (tests + lint/format/tidy):
    - `make all-tests`

- Unit tests (optionally pass ARGS=...):
    - `make unit`
    - `make unit ARGS="./... -run TestName"`

- Integration tests:
    - `make integration`

- Contract tests:
    - `make contract`

- Smoke tests:
    - `make smoke`

- Coverage report (combined):
    - `make coverage`

- Build/install local binary:
    - `make install` (uses `TARGET_OS`, default `darwin`)
    - `make reinstall`

- Build release binaries:
    - `make binaries`

- Update deps + vendor:
    - `make updatedeps`

## MCP local test servers

These run local FastMCP servers under `test/mcp/http` (requires the venv there):

- `make mcp-http`
- `make mcp-sse`

## Release / automation helpers (use with care)

- `make shipit version=<semver> message="..."`
- `make commit` (generates a commit message from `git diff`)

## Dependency management

- Use `go mod tidy` only when changing dependencies.
- Never edit `vendor/` by hand.

## Coding conventions

- Keep changes small and focused.
- Follow nearby patterns (logging, errors, config).
- Prefer clear, testable functions.
- Use structured logging (Zap is used in this repo).

## Testing expectations

- Add or update tests for any behavior change.
- Keep tests deterministic.
- Avoid real network calls unless the project already uses fixtures/mocks.
- For new external interactions, introduce an interface and test with fakes/mocks.

## Agent workflow

1. Identify ownership: is the change in `cmd/` (CLI wiring) or in `internal/`/`agent/` (logic)?
2. Find config keys and defaults (`config/` + Viper wiring).
3. Make the minimal code change.
4. Add or update tests.
5. Run: `make unit` and `make integration`.
6. Only proceed if both pass.

## Guardrails

- Never commit secrets (API keys, tokens). Use env vars and documented config.
- Do not touch `.git/` or generated caches (`cache/`, `history/`, `.pytest_cache/`) unless explicitly asked.
- Avoid large refactors unless requested.

## When you need more context

If requirements are unclear, ask for:

- The exact CLI command/flags involved.
- Expected output or behavior.
- Example inputs (config, prompts, transcripts).
- Constraints (backwards compatibility, OS/shell support).