# Build, Release, and CI

## Local build

- Go 1.25 module: `github.com/yarlson/lnk`. Direct dependencies are `spf13/cobra` and `stretchr/testify`.
- `make build` produces `./lnk`, injecting `main.version` and `main.buildTime` via `-ldflags`. The version string defaults to `git describe --tags --always --dirty` and falls back to `dev`.
- `make check` runs `fmt`, `vet`, `lint`, and `test` and is the developer-side equivalent of CI.
- `Makefile` also exposes `cross-compile`/`release` targets that build for `linux|darwin|windows × amd64|arm64` plus a SHA256 manifest, but these are legacy paths superseded by GoReleaser.

## Install paths

- One-shot install script: `install.sh` (also published via `curl | bash` from the GitHub raw URL).
- Homebrew formula published to a tap by GoReleaser.
- Pre-built binaries on GitHub Releases (one per OS/arch).
- `go install github.com/yarlson/lnk@latest` for a from-source install.

## GoReleaser

`.goreleaser.yml` drives release artifacts: cross-platform binaries with embedded version/buildtime, archives, checksums, and the Homebrew tap update. Snapshot builds are exercised in CI.

## GitHub Actions

Three workflows under `.github/workflows/`:

- **ci.yml** — runs on push to `main` and on PRs. Three jobs: `test` (gofmt strict + `go vet` + `go test -race -coverprofile` + Codecov upload), `lint` (golangci-lint), and `build` (depends on `test` and `lint`; runs `go build ./...` and `goreleaser build --snapshot --clean`).
- **release.yml** — runs on tag push (`v*`). Runs `go test ./...` then `goreleaser release --clean` with `HOMEBREW_TAP_TOKEN` so the Homebrew tap repo can be updated.
- **validate.yml** — runs on PRs that touch `.goreleaser.yml`, `main.go`, `cmd/**`, `internal/**`, or Go module files. Runs `goreleaser check` and a snapshot build to fail fast on release-config regressions.

## Linting

- `.golangci.yml` configures the linters used by both `make lint` and CI's `lint` job.
- `gofmt -l .` strict-zero check is enforced in CI; running `make fmt` (i.e., `go fmt ./...`) keeps it green.

## Distribution shape

- One binary, no runtime files. Configuration is the user's repo path and (optionally) the `LNK_HOME`, `XDG_CONFIG_HOME`, and `NO_COLOR` environment variables.
- The binary shells out to system `git`. There is no embedded git library.
