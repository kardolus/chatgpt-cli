# AGENTS.md

This repository is a Go-based CLI application (module: `github.com/kardolus/chatgpt-cli`) built with Cobra/Viper.

Use this document as the default operating context when acting as an agent in this repo.

## High-level purpose
- Project type: Go command-line tool.
- Entry points: `cmd/` (Cobra commands), with supporting packages under `internal/`, `api/`, and `agent/`.
- Configuration: likely via Viper (YAML/TOML/ENV) and files under `config/`.

## Repository layout (key paths)
- `cmd/`: CLI command definitions / wiring (Cobra).
- `internal/`: non-exported application logic.
- `api/`: API clients/types (project-specific).
- `agent/`: agent-related logic (prompting, tools, orchestration).
- `config/`: config defaults/templates.
- `docs/`: documentation.
- `scripts/`: dev scripts.
- `test/`: tests/fixtures.
- `vendor/`: vendored dependencies (treat as read-only unless explicitly asked).

## How to work in this repo
### Prerequisites
- Go version: **1.24.1** (per `go.mod`).

### Common commands
If a `Makefile` target exists for a task, prefer it; otherwise use the Go toolchain.

- Run tests:
  - `go test ./...`
- Run a specific package test:
  - `go test ./internal/... -run TestName`
- Build:
  - `go build ./...`
- Run the CLI locally (typical patterns):
  - `go run . --help`
  - `go run ./cmd/<subcommand> --help` (if commands are separate mains)

### Dependency management
- Prefer `go mod tidy` only when changing dependencies.
- Avoid editing `vendor/` manually.

## Coding conventions / expectations
- Keep changes small and focused.
- Follow existing patterns in nearby code (logging, error handling, config).
- Prefer clear, testable functions.
- Use structured logging (Zap is a dependency).

## Testing expectations
- Add/adjust tests for behavior changes.
- Keep tests deterministic; avoid network calls unless the project already uses fixtures/mocks.
- If introducing new behavior around external APIs, add a mockable interface and test with fakes.

## Agent workflow (how to proceed on tasks)
1. Identify the correct command/package that owns the behavior (`cmd/` wiring vs `internal/` logic).
2. Locate config keys and defaults (likely under `config/` + Viper wiring).
3. Make the minimal code change.
4. Add/update tests.
5. Run `go test ./...` and ensure the CLI still builds.

## Guardrails
- Do not commit secrets (API keys, tokens). Prefer env vars and documented config.
- Do not modify `.git` or generated caches (`.pytest_cache/`, `cache/`, `history/`) unless explicitly requested.
- Avoid large refactors unless requested.

## When you need more context
If requirements are ambiguous, ask for:
- The exact CLI command/flags involved and expected output.
- Example inputs (config files, prompts, transcripts).
- Any constraints (backwards compatibility, supported OS/shell).
