# Development and Release

## Requirements

- Go 1.22+
- git
- zsh
- make

## Checks

```sh
make check
go build -o hi-shell ./cmd/hi-shell
```

`make check` runs tests, vet, and shell script syntax checks. CI runs the same checks on push and pull request.

Optional: install the local pre-commit hook to run `make check` before each commit:

```sh
make install-hooks
```

## Manual Plugin Test

Test the plugin without touching `~/.zshrc`:

```sh
go build -o hi-shell ./cmd/hi-shell
export PATH="$PWD:$PATH"
zsh -f
source ./shell/hi.zsh
```

## Release

Create and push a version tag:

```sh
git tag vX.Y.Z
git push origin vX.Y.Z
```

The release workflow builds Linux/macOS archives for arm64 and x86_64, generates `checksums.txt`, generates release notes from commits since the previous `v*` tag, and uploads them to GitHub Releases.

The curl installer downloads the matching archive and verifies its checksum before installing.
