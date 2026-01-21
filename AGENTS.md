# AGENTS.md

Operating guide for automated agents (and humans) working in this repository.

This repo is a Go-based CLI application (module: `github.com/kardolus/chatgpt-cli`) built with **Cobra** (commands) and **Viper** (configuration). Agent mode and policies live under `agent/`.

Use this document as the default working context: how to navigate the repo, how to make changes safely, and what “done” means.

---

## 1) Repo map (where things usually live)

- `cmd/` — CLI commands, flags, wiring (Cobra).
- `internal/` — core application logic (non-exported).
- `api/` — API clients + domain/types for providers.
- `agent/` — agent runtime: planning, tools, budgets, policy.
- `config/` — config defaults/templates.
- `docs/` — documentation.
- `scripts/` — repo workflows used by Make targets.
- `test/` — tests + fixtures (including MCP test servers).
- `vendor/` — vendored dependencies (**do not edit by hand**).

Rule of thumb:
- If it changes user-facing flags/output: start in `cmd/`.
- If it changes behavior/business logic: it likely belongs in `internal/`, `api/`, or `agent/`.

---

## 2) Build & test workflow (use Makefile first)

Prefer Make targets over ad-hoc commands; the scripts encode the intended workflow.

List targets:
- `make help`

“Done” means tests pass:
- `make unit`
- `make integration`

Do not skip integration tests unless explicitly instructed.

Other useful targets:
- `make all-tests` (tests + lint/format/tidy)
- `make contract`
- `make smoke`
- `make coverage`
- `make install` / `make reinstall` / `make binaries`
- `make updatedeps`

Notes:
- Run `go mod tidy` only when dependency changes are intended.
- Never edit `vendor/` directly; use the dependency workflow (`make updatedeps`).

---

## 3) Go version

- Go version is defined in `go.mod`. Avoid using language/library features newer than that version unless the module version is intentionally bumped.

---

## 4) Testing expectations (required)

For any behavior change or new feature, add or update tests.

This repo uses:
- `sclevine/spec` test structure
- `gomega` assertions

Follow existing style (see `agent/budget_test.go` as a reference).

Minimum test coverage for a change:
- Primary “happy path” behavior
- Main failure/validation path
- One invariant (e.g., no partial updates on failure)

Guidelines:
- Prefer deterministic tests.
- Avoid real network calls unless the project already uses fixtures/mocks.
- For new external interactions, introduce an interface and test with fakes/mocks.
- Use typed errors and verify with `errors.As` where appropriate.

---

## 5) Agent policy model (how the built-in agent enforces safety)

The default policy is implemented in `agent/policy.go`.

### Tool kinds

Supported step/tool types:
- `shell`
- `llm`
- `files`

If `allowed_tools` is configured, steps outside that set are denied.

### Shell safety

- Shell steps must include a non-empty `Command`.
- Denied commands are checked against the **exact** command string (trimmed). If the command is `rm` and `DeniedShellCommands` includes `rm`, it will be blocked.
- When `restrict_files_to_work_dir` is enabled and `work_dir` is set, the policy also rejects suspicious path-like shell arguments that escape the work dir (absolute paths, `~`, `..`, or path-looking args containing separators).

Practical implication:
- Prefer running commands scoped to the repo (relative paths).
- Avoid passing absolute paths or `..` in shell args when workdir restriction is on.

### File operations

File steps must include:
- `Op` (lowercased internally)
- `Path`

Some ops have additional requirements:
- `patch` requires `Data` (unified diff)
- `replace` requires `Old` pattern

Allow/deny model:
- `allowed_file_ops` is an allowlist. If it’s empty, all ops are allowed.
- If `write` is allowed, `patch` and `replace` are implicitly allowed.
- When `restrict_files_to_work_dir` is enabled, file paths are denied if they resolve outside the work dir.

---

## 6) How to make changes (agent-friendly workflow)

1. **Locate ownership**
   - `cmd/` for flags/output/wiring
   - `internal/` / `api/` / `agent/` for logic
2. **Trace config and defaults**
   - Viper wiring + `config/` templates
3. **Implement the smallest correct change**
   - Keep diffs focused; avoid broad refactors unless requested
4. **Add/update tests**
   - Cover happy path, failure path, and an invariant
5. **Run required targets**
   - `make unit`
   - `make integration`
6. **Sanity-check UX**
   - Help text, error messages, and output formatting

---

## 7) Coding conventions used in this repo

- Keep functions small and testable.
- Follow existing patterns for:
  - errors (typed errors where useful)
  - logging (Zap is used here)
  - configuration (Viper + config templates)
- Prefer clarity over cleverness.

---

## 8) Guardrails (do not violate)

- Never commit secrets (API keys/tokens). Use env vars and documented config.
- Do not modify `.git/`.
- Do not touch generated/runtime directories unless explicitly asked:
  - `cache/`, `history/` (and similar local state)
- Treat `vendor/` as read-only.

---

## 9) If you truly need more context

Even when acting autonomously, get precise signals from the codebase:
- Search for the exact command/flag under `cmd/`.
- Search for config keys under Viper wiring and `config/`.
- Find existing tests that exercise similar behavior and mirror their structure.

When requirements are ambiguous, prefer:
- conservative behavior
- backwards-compatible defaults
- explicit validation errors
