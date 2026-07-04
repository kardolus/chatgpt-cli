# chatgpt-cli — release runbook

How to cut and publish a new release. A release touches **two repos**:

1. **this repo** (`kardolus/chatgpt-cli`) — run the tests, tag, build binaries, and publish a
   GitHub Release with those binaries attached (the direct-download install method).
2. **the Homebrew tap** (`kardolus/homebrew-chatgpt-cli`) — bump the formula so
   `brew upgrade chatgpt-cli` picks up the new version.

Homebrew builds `chatgpt` **from source** (the formula points at the tag's source tarball),
while the pre-built `bin/chatgpt-<os>-<arch>` artifacts serve the direct-download install
documented in the README (`releases/latest/download/chatgpt-<os>-<arch>`).

The tap repo is expected as a sibling checkout next to this one (i.e. `../homebrew-chatgpt-cli`).

## Prerequisites

- **Go** (the version in `go.mod`'s `go` directive; the toolchain auto-downloads if
  `GOTOOLCHAIN=auto`).
- **golangci-lint v2** on your PATH. The lint step (`golangci-lint run`, via `.golangci.yml`)
  targets the module's Go version; a v1 golangci-lint (built with an older Go) will refuse to
  run and the release will abort at the test step.
- **`ag`** (the_silver_searcher) — `all-tests.sh` uses it to scan for stray `TODO`s.
- **`gh`** (GitHub CLI, authenticated) — to create the GitHub Release and upload binaries.
- **`chatgpt`** on your PATH — `shipit` uses the CLI itself to generate the release notes.
- A sibling **`../prompts/`** directory containing `write_release_notes.md`; `shipit`
  references it as `../prompts/write_release_notes.md`.

## 1. Cut the release

From this repo, on a clean `main` that's up to date with origin:

```shell
make shipit version=vX.Y.Z message="short release message"
```

`version` is the new semver tag (bump from the latest tag). This runs `scripts/shipit.sh`,
which:

1. Aborts if the working tree has unstaged changes.
2. `scripts/updatedeps.sh` — `go get -u ./...`, `go mod vendor`, `go mod tidy`; if anything
   changed, it auto-commits `chore: bump dependencies` and pushes.
3. `scripts/all-tests.sh` — `go mod tidy` check, `go fmt` check, **`golangci-lint run`**, a
   `TODO` scan, then unit + integration + contract tests.
4. **Release notes** — diffs the last tag to `HEAD` and pipes it through `chatgpt` +
   `../prompts/write_release_notes.md`, printing the notes to **stdout**. Copy this output —
   it's the body for the GitHub Release in step 2. (Tip: tee it, e.g.
   `make shipit ... | tee /tmp/release-notes.md`.)
5. Tags the release (`git tag -a vX.Y.Z -m "message"`) and pushes tags.
6. `scripts/binaries.sh` — builds `bin/chatgpt-<os>-<arch>` for 8 targets
   (darwin / linux / windows / freebsd across amd64 / arm64 / 386), stamping `GitCommit` and
   `GitVersion` (from `git describe`) via `-ldflags`.

To only (re)build the binaries without the rest, use `make binaries`.

## 2. Publish the GitHub Release + binaries

The tag is already pushed. Create the Release for it and attach every `bin/chatgpt-*`
artifact, using the notes from step 1.4:

```shell
gh release create vX.Y.Z ./bin/chatgpt-* \
  --title vX.Y.Z \
  --notes-file /tmp/release-notes.md
```

(Equivalently, create the Release in the web UI for the pushed tag, paste the notes, and drag
in the `bin/chatgpt-*` files.) The README's direct-download links resolve to
`releases/latest/download/chatgpt-<os>-<arch>`, so all 8 assets must be attached.

## 3. Update the Homebrew tap

In the `../homebrew-chatgpt-cli` checkout, edit `HomebrewFormula/chatgpt-cli.rb`:

- Point `url` at the new tag's source tarball:
  `https://github.com/kardolus/chatgpt-cli/archive/refs/tags/vX.Y.Z.tar.gz`
- Update `sha256` to that tarball's hash:

```shell
curl -sL https://github.com/kardolus/chatgpt-cli/archive/refs/tags/vX.Y.Z.tar.gz | shasum -a 256
```

Commit and push the tap. Users can then upgrade:

```shell
brew update && brew upgrade chatgpt-cli
```

## Notes / troubleshooting

- **Lint fails to run** (`the Go language version ... is lower than the targeted Go version`):
  your local golangci-lint is too old — install **v2**.
- **`shipit` aborts immediately**: it refuses to run with unstaged changes — commit or stash
  first.
- **Version shows wrong in the binary**: `GitVersion` comes from the pushed tag via
  `git describe --tags`; make sure the tag exists before `binaries.sh` runs (it does inside
  `shipit`).
- **`brew upgrade` doesn't see the new version**: confirm the tap commit landed and the
  formula `sha256` matches the new tarball.
