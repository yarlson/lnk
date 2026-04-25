# Sync Flow — status / diff / push / pull / list

All sync operations require the repo path to be a Git repository; otherwise they return `ErrNotInitialized` with `run 'lnk init' first`.

## Status (`lnk status`)

`syncer.Status` calls `git.GetStatus`, which:

1. Checks if a remote exists (`origin`, or any remote if `origin` is missing). If no remote, `Remote` is set to empty string.
2. Detects dirty state via `git status --porcelain`.
3. Resolves the upstream tracking branch via `rev-parse --abbrev-ref --symbolic-full-name @{u}`. If no upstream is set, defaults to `origin/main`.
4. Counts ahead via `rev-list --count <upstream>..HEAD` (falls back to all-local-commits if the upstream branch doesn't exist remotely).
5. Counts behind via `rev-list --count HEAD..<upstream>`. Behind is always 0 when there is no upstream.

`StatusInfo{Ahead, Behind, Remote, Dirty}` is rendered by `cmd/status.go` through four branches: no remote (guides user to add a remote, shows local state), dirty (suggests commit or push), up-to-date (synced), or ahead/behind (suggests push/pull). When dirty or a remote exists, the display references the actual repo path (via `lnk.DisplayPath(lnk.GetRepoPath())`) so messages adapt when the repo is in a custom location via `LNK_HOME` or `XDG_CONFIG_HOME`.

## Diff (`lnk diff`)

`syncer.Diff(color)` runs `git diff --color=never|always` in the repo path. The CLI respects the `--colors` flag (auto-detected or explicit) and routes output through `Writer`. When `--quiet` is set, the command suppresses all output through `Writer`, returning only the exit code. When the diff is empty, the CLI prints a structured "No uncommitted changes" message instead (unless `--quiet` suppresses it).

## Push (`lnk push [message]`)

1. `git.HasChanges` — if the working tree is dirty, `git add -A` then `git commit -m <message>`. The default message is `lnk: sync configuration files`; users can override by passing one positional arg.
2. `git push -u origin` (5-minute timeout). Setting upstream every time is intentional — it makes the first push from a freshly-cloned-or-initialized repo work without extra setup.

If there are no changes, push proceeds straight to `git push -u origin`. The CLI then prints commit + sync messaging.

## Pull (`lnk pull [--host H]`)

1. `git pull origin` (5-minute timeout).
2. `RestoreSymlinks` walks the index for the active scope (common or host) and ensures `~/<relativePath>` is a symlink to the stored file, returning `RestoreInfo{Restored, BackedUp}`:
   - Skip entries whose stored file doesn't exist (a partial pull, host not present in repo, etc.).
   - Skip entries whose symlink already resolves to the expected target (`IsValidSymlink` compares absolute paths after resolving relative targets against the link's directory).
   - `os.MkdirAll` the symlink's parent directory.
   - If `~/<relativePath>` exists and is a regular file or directory, rename it to `<path>.lnk-backup` (preserve user data, append relative path to `BackedUp` list).
   - If it exists and is a stale symlink, `os.Remove` it.
   - `fs.CreateSymlink(repoItem, symlinkPath)` — relative symlink, append relative path to `Restored` list.

The CLI separates outcomes: if `Restored` is non-empty, display the list of restored symlinks and any backup notice (files renamed to .lnk-backup), else display `All symlinks already in place`. When `--host` is set, the host name is included in messaging.

## List (`lnk list [--host H | --all]`)

`syncer.List` returns the index entries for the active scope. The CLI has three modes:

- Default — common configuration only.
- `--host H` — that single host.
- `--all` — common, then every host found by enumerating `.lnk.*` files at the repo root, each rendered as its own section. For each host section, the CLI emits a `lnk pull --host <host>` hint to guide restoration.

`lnk list` requires a Git repo at the repo path (same `ErrNotInitialized` check). The list does not verify that managed items still exist or that their symlinks are healthy — that's the job of `lnk doctor`.

## Restore-only path

`syncer.RestoreSymlinks` is also called by `doctor` when fixing broken symlinks, so the `pull` and `doctor` paths converge on a single implementation. Tests around symlink validity, backup behavior, and stale-link replacement live next to `internal/lnk/sync_test.go`.
