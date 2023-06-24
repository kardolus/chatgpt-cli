# Release Procedures Manual

## 1. Update the dependencies

It's a good practice to update the dependencies before cutting a new release

```shell
./scripts/updatedeps.sh
```

## 2. Create a tag

Use the git-cli to create a tag, for example:

```shell
git tag -a v1.0.5 -m "Add list-models flag"
```

Next, push the tag:

```shell
git push origin --tags
```

## 3. Create binaries

From the root of `kardolus/chatgpt-cli`, run the following script to create binaries for various architectures:

```shell
./scripts/binaries.sh
```

## 4. Create a GitHub release

Create a GitHub release for the tag we just pushed out. Upload the binaries created in the previous step. Add this
section with update instructions to the end:

```markdown
## How to Update

### Using Homebrew (macOS)

\```shell 
brew upgrade chatgpt-cli
\```

### Direct Download

For a quick and easy installation without compiling, you can directly download the pre-built binary for your operating
system and architecture.
```

## 5. Bump the version

Bump the version in the `README` of `kardolus/chatgpt-cli` and in the Homebrew
formulae (`kardolus/homebrew-chatgpt-cli/HomebrewFormula/chatgpt-cli.rb`). Update the sha256 of the macOS binaries
using:

```shell
sha256sum /path/to/binary
```
