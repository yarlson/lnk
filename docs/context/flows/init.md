# Init Flow

Entry point: `cmd/init.go` → `lnk.NewLnk().InitWithRemoteForce(remote, force)` → `initializer.Service`.

## Without a remote (`lnk init`)

1. `os.MkdirAll(repoPath, 0755)`.
2. If a `.git` already exists:
   - `IsLnkRepository()` → adopt silently and return.
   - Otherwise return `ErrGitRepoExists` with a suggestion to back up the existing repo.
3. Else `git init -b main` (Git 2.28+); on failure, fall back to `git init` followed by `git symbolic-ref HEAD refs/heads/main`.

The user lands with an empty Git repo at the repo path and is prompted to `lnk add <file>` next.

## With a remote (`lnk init -r <url>`)

1. If the repo path already contains `.lnk` or `.lnk.*` files (`HasUserContent`), refuse with `ErrManagedFilesExist` unless `--force`. The error suggests `lnk pull` instead.
2. `git.Clone(url)`:
   - `os.RemoveAll(repoPath)` to ensure a clean clone target.
   - `git clone <url> <repoPath>` from the parent directory (5-minute timeout).
   - Set upstream: try `branch --set-upstream-to=origin/main main`, else `origin/master master`, else best-effort `origin/HEAD`.
3. Unless `--no-bootstrap`, `bootstrapper.FindScript()` looks for `bootstrap.sh` at the repo root and runs it via `bash bootstrap.sh` with the user's stdio. A bootstrap failure is reported but does not undo the clone — the user is told to retry with `lnk bootstrap`.
4. The CLI prints next-step hints:
   - `lnk pull` to restore common symlinks.
   - `lnk pull --host <host>` for each discovered host (enumerated via `findHostConfigs` by listing `.lnk.*` files).
   - `lnk add <file>` to start managing new files.

## Adopting an existing remote on a fresh repo

`lnk.AddRemote(name, url)` (used in tests / scripted setups) forwards to `git remote add`, but is idempotent: if the remote already points at the same URL it returns nil; if it points at a different URL it errors with both URLs in the message.

## Failure handling

- `git init` failures map to `ErrGitInit` with the suggestion to verify git is installed.
- Clone failures map to `ErrGitCommand` with a network/URL hint.
- Timeouts (`context.DeadlineExceeded`) map to `ErrGitTimeout` with a system-resources suggestion.
