# Practices

Conventions and invariants that are enforced by code or test, not aspirational style.

## Error model

- All user-facing errors are sentinel `error` values defined in `internal/lnkerror` and the relevant subpackages.
- Sentinels are wrapped via the single `lnkerror.Error` type using `Wrap`, `WithPath`, `WithSuggestion`, or `WithPathAndSuggestion`. There is no other custom error type.
- The CLI surfaces errors only through `cmd.DisplayError`, which renders the wrapped path and suggestion uniformly with emoji/color settings.
- Every Cobra command sets `SilenceUsage: true` and `SilenceErrors: true` so Cobra never prints raw errors; output goes through the structured writer.

## Output

- All formatted output flows through `cmd.Writer` and `cmd.Message`, never `fmt.Println` directly.
- Color is decided by `--colors auto|always|never` plus `NO_COLOR` (env wins only in `auto` mode).
- `--emoji` and `--no-emoji` are mutually exclusive (enforced via Cobra `MarkFlagsMutuallyExclusive`).
- `--quiet`/`-q` suppresses all `Writer` output; the only signal is the exit code.
- Auto-detection of TTY happens once on first use; explicit flags pin the config and skip detection.
- Progress updates with carriage-return redraws only appear when output is a terminal (`Writer.IsTerminal()`). In piped or redirected contexts, progress text is omitted entirely to prevent log corruption.

## Path display

- **Storage paths** shown in CLI output use `lnk.FormatManagedPath(host, originalPath)` to ensure consistent formatting across commands. `FormatManagedPath` computes the canonical storage location (accounting for host scoping), then displays it home-relative (with `~`) or `/`-stripped for paths outside `$HOME`.
- **Source paths** in preview and batch output use `displaySourcePath` to render home-relative paths (~/dir/file), allowing files with identical basenames in different directories to remain disambiguated. Falls back to the original input on path resolution failure.
- When listing many files in batch output, only the first 5 entries are shown in detail, followed by "... and N more files" to keep output compact and readable.

## Repo-path resolution

- Order is fixed: `LNK_HOME` env > `XDG_CONFIG_HOME/lnk` > `~/.config/lnk`. If the home directory is unavailable, the path falls back to `./lnk`.
- Commands always read `lnk.GetRepoPath()`; never inline a default path.

## Add/remove are atomic

- `Add` and `AddMultiple` execute in three phases (validate → process → git commit). Any failure rolls back all completed steps in reverse order via `RollbackAll`.
- The unit of atomicity is one git commit per CLI invocation. Multi-file `add` produces a single commit (`lnk: added N files` / `lnk: added N files recursively`), not one per file.
- `Remove` (non-force) refuses to act unless the path is a symlink whose target is inside the repo path; this is a safety check in `fs.ValidateSymlinkForRemove`.

## Symlink shape

- Symlinks created by lnk are **relative** (`filepath.Rel` between link and target). This keeps the repo portable across home-directory locations.
- `pull`/`doctor` validate symlinks by resolving the target and comparing absolute paths to the expected stored file.
- On `pull`, if `~/<relative path>` exists as a real file or directory (not a symlink), it is renamed to `<path>.lnk-backup` rather than removed. Stale symlinks are removed.

## Git invocation

- All git operations go through `internal/git`, which runs system `git` with a context timeout: 30s for local operations, 5m for clone/push/pull.
- `IsLnkRepository` treats a Git repo as an lnk repo iff it has zero commits or every commit subject starts with `lnk:`. This is how init decides whether to adopt an existing `.git`.
- Commit subjects follow `lnk: <action> <object>` (e.g., `lnk: added .vimrc`, `lnk: removed .bashrc`, `lnk: cleaned N invalid entries`, `lnk: sync configuration files`).
- If `user.name` / `user.email` are unset in the repo, `ensureGitConfig` writes `Lnk User` / `lnk@localhost` so commits never fail on a fresh machine.

## Host scoping

- Host scoping is a runtime choice (`WithHost("name")`), not a state. The `Lnk` facade re-wires its collaborators with the host value during `NewLnk`.
- An empty host means common configuration; collaborators that need to choose between `.lnk` vs `.lnk.<host>` and root vs `<host>.lnk/` ask `Tracker` (`LnkFileName`, `HostStoragePath`).
- Common and host configurations never share state: separate index files, separate storage roots.

## Testing

- Tests live next to the code they exercise. `cmd/root_test.go` and `internal/lnk/*_test.go` are the largest, exercising commands and the facade end-to-end against a real Git repo in a tempdir.
- Tests use real git, real filesystem, and real symlinks — there is no mocking of `git` or `fs`.

## CI gates

- `gofmt -l` must be empty, `go vet ./...` clean, `golangci-lint` clean, `go test -race ./...` passing, and `goreleaser build --snapshot --clean` succeeding before merge.
- Releases are tag-driven (`v*`) and run GoReleaser end-to-end with the Homebrew tap token.
