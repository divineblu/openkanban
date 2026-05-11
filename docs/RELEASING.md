# Releasing

OpenKanban releases are built with GoReleaser. GitHub releases publish binary
archives, and GoReleaser writes a Homebrew formula to `divineblu/homebrew-tap`.

## One-Time Homebrew Setup

1. Create the public tap repository:

   ```bash
   gh repo create divineblu/homebrew-tap --public --description "Homebrew tap for divineblu tools" --add-readme
   ```

2. Make `divineblu/openkanban` public before advertising Homebrew installs.
   Public users need access to the release archives referenced by the cask.

3. Add an Actions secret on `divineblu/openkanban` named
   `HOMEBREW_TAP_TOKEN`. Use a GitHub token that can write contents to
   `divineblu/homebrew-tap`.

## Release

Run the normal checks before tagging:

```bash
make test-unit
make test-integration
make lint
go run github.com/goreleaser/goreleaser/v2@latest check
```

Create and push a version tag, or use the `Release` workflow dispatch:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow uploads the GitHub release artifacts and updates
`divineblu/homebrew-tap/Formula/openkanban.rb`.

## Test Homebrew

After the release workflow completes:

```bash
brew uninstall --cask openkanban || true
brew untap divineblu/tap || true
brew install divineblu/tap/openkanban
openkanban version
```

Use `brew upgrade openkanban` for updates.

## Homebrew Core

This tap supports:

```bash
brew install divineblu/tap/openkanban
```

Plain `brew install openkanban` requires acceptance into `Homebrew/homebrew-core`
or `Homebrew/homebrew-cask`, which is a separate upstream submission process.
