# Architecture

`lnk` is a single Go binary. The CLI layer in `cmd/` translates Cobra arguments into calls on a domain facade in `internal/lnk`, which composes focused collaborators.

## Layering

```
main.go
  └── cmd/                       Cobra commands, output formatting, error display
        └── internal/lnk         Facade composing collaborators
              ├── internal/initializer   init / clone / remote / detect lnk repo
              ├── internal/tracker       .lnk index file (read/write/add/remove)
              ├── internal/filemanager   add / remove (move + symlink + git, with rollback)
              ├── internal/syncer        status / diff / push / pull / list / restore symlinks
              ├── internal/doctor        find + fix invalid entries and broken symlinks
              ├── internal/bootstrapper  find + run bootstrap.sh
              ├── internal/git           subprocess git wrapper with timeouts
              ├── internal/fs            filesystem ops (validate / move / symlink)
              └── internal/lnkerror      single Error wrapper + sentinel errors
```

Dependency direction is one-way: `cmd → lnk → {initializer, tracker, filemanager, syncer, doctor, bootstrapper} → {git, fs, lnkerror}`. The leaf packages (`git`, `fs`, `lnkerror`) depend only on the standard library and on `lnkerror`.

## The `Lnk` facade

`internal/lnk.Lnk` is the only type the CLI talks to. `NewLnk(opts ...Option)` resolves the repo path, applies options (currently just `WithHost`), then constructs collaborators with the resolved host. Its public methods are thin delegates — almost every method is one line forwarding to a collaborator.

Re-exported from the facade for backwards compatibility:

- Sentinel errors (`ErrAlreadyManaged`, `ErrNotInitialized`, etc.) — re-exported from `lnkerror`.
- Type aliases: `ProgressCallback = filemanager.ProgressCallback`, `StatusInfo = syncer.StatusInfo`, `DoctorResult = doctor.Result`.
- Helpers: `DisplayPath` (replaces `$HOME` with `~`), `GetCurrentHostname`, `GetRepoPath`, `FormatManagedPath` (formats the display path where a file is/will be stored for a given host).

## Collaborator responsibilities

- **initializer.Service** — creates the repo directory, decides whether an existing `.git` is an lnk repo (zero commits or all commits start with `lnk:`), runs `git init -b main` (or `init` + `symbolic-ref`), or clones a remote and sets upstream tracking. Errors with a clear suggestion when there is pre-existing user content unless `--force`.
- **tracker.Tracker** — owns the `.lnk` / `.lnk.<host>` file: read, append (sorted), remove, write. Also resolves the host storage path (`<repo>` for common, `<repo>/<host>.lnk` for host).
- **filemanager.Manager** — `Add`, `AddMultiple` (atomic, single commit), `AddRecursiveWithProgress` (walks dirs to a flat file list), `PreviewAdd` (dry-run), `Remove`, `RemoveForce`. Three phases for batch add: validate → process (move + symlink + track) → git stage + commit. Each step pushes a rollback action; failures unwind in reverse.
- **syncer.Syncer** — git-status-derived `Status`, `Diff`, `Push` (auto-stages-all + commits if dirty, then pushes with `-u origin`), `Pull` (git pull then `RestoreSymlinks`), `List`, `RestoreSymlinks` (creates missing/wrong symlinks, backs up real files to `.lnk-backup`).
- **doctor.Checker** — `Preview` and `Fix` for two issue classes: invalid index entries (path missing in storage, or escapes storage root) and broken symlinks at `~/<relative path>`. Fix delegates symlink repair to the syncer and prunes invalid entries from the index with a single `lnk: cleaned N invalid entr(y|ies)` commit.
- **bootstrapper.Runner** — locates `bootstrap.sh` at the repo root, `chmod 0755`, runs `bash bootstrap.sh` with caller-supplied stdio.
- **git.Git** — every git subprocess used by the rest of the code. All commands run with `cmd.Dir = repoPath` and a context timeout. Treats `context.DeadlineExceeded` as `ErrGitTimeout`.
- **fs.FileSystem** — `ValidateFileForAdd` (must exist, must be regular file or directory), `ValidateSymlinkForRemove` (must be a symlink whose target lives inside the repo), `Move`/`MoveFile`/`MoveDirectory` (rename), `CreateSymlink` (relative target). Plus a free function `GetRelativePath` (home-relative, or `/`-stripped absolute for paths outside `$HOME`).

## CLI layer

- One file per subcommand under `cmd/`: `init`, `add`, `rm`, `list`, `status`, `diff`, `push`, `pull`, `doctor`, `bootstrap`.
- `cmd/root.go` builds the root command, registers persistent flags (`--colors`, `--emoji`, `--no-emoji`, `--quiet`/`-q`), wires `SetGlobalConfig`, and registers all subcommands. Long help text in `Long` is the source of truth for command descriptions.
- `cmd/output.go` defines `Writer`, `Message`, predefined message constructors (`Success`, `Error`, `Warning`, `Info`, `Target`, `Rocket`, `Sparkles`, `Link`, `Plain`, `Bold`, `Colored`), and the global `OutputConfig`. Writer exposes `Colors()` and `Quiet()` accessors for commands to query the active color and quiet-mode settings. Auto-detection runs once on first writer access; explicit flags via `SetGlobalConfig` short-circuit it.
- `cmd.DisplayError` is the single error rendering path; called from `Execute` on any error returned by a `RunE`.
- `Version` is set from `main.go` at startup via `cmd.SetVersion(version, buildTime)`; both are populated by GoReleaser ldflags.
