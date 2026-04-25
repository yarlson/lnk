# Add / Remove Flow

The `add` family is the most defensively-coded path in the codebase because it mutates user files, the index, and Git in three steps that can each fail.

## Single-file add (`lnk add <file>`)

`cmd/add.go` routes single-file `add` to `Lnk.Add` (no progress, no batching) so existing CLI output stays unchanged. Steps in `filemanager.Manager.Add`:

1. `fs.ValidateFileForAdd` — must exist, must be a regular file or directory.
2. Compute `absPath` (from CWD) and `relativePath` (home-relative; `/`-stripped for paths outside `$HOME`).
3. `os.MkdirAll(filepath.Dir(destPath))` where `destPath = HostStoragePath()/relativePath`.
4. Check the index — if `relativePath` is already in `.lnk`/`.lnk.<host>`, return `ErrAlreadyManaged`.
5. `os.Stat` the source to capture mode info for the move.
6. `fs.Move(absPath, destPath, info)` — `os.Rename` (file or directory).
7. `fs.CreateSymlink(destPath, absPath)` — relative symlink. On failure, move the file back and return.
8. `tracker.AddManagedItem(relativePath)` — read, append, sort, write.
9. `git.Add(<gitPath>)` where `gitPath = relativePath` for common or `<host>.lnk/<relativePath>` for host scope.
10. `git.Add(<index file>)`.
11. `git.Commit("lnk: added <basename>")`.

Each Git/track step rolls back the prior steps (delete symlink, remove index entry, move file back) before returning.

## Multi-file add (`lnk add <fileA> <fileB> ...`)

Routes to `AddMultiple`, which runs three explicit phases:

- **validatePaths** — for every path: validate, compute abs+relative, reject duplicates against the index, capture stat. Pure read-only; any failure aborts before touching the filesystem.
- **processFiles** — for each validated file: ensure the destination directory, move into place, create the relative symlink, append to the index, push a rollback action onto a stack. Any failure unwinds the stack via `RollbackAll` (reverse order: delete symlink, remove index entry, move back).
- **commitFiles** — `git add` every storage path, `git add` the index file, then a single `git.Commit("lnk: added N files")`. On any failure, `RollbackAll` plus an error.

The result is exactly one commit per CLI invocation, even with hundreds of files.

Success output lists up to 5 source files, rendered home-relative (~/dir/file) via `displaySourcePath` to disambiguate files with identical basenames in different directories. If more than 5 files were added, additional files are collapsed into "... and N more files".

## Recursive add (`lnk add --recursive <dir>...`)

`AddRecursiveWithProgress` walks each path with `filepath.Walk`, collecting regular files and symlinks into a flat list, then forwards to `AddMultiple`. If the total exceeds 10 files (`progressThreshold`) and the caller passes a progress callback, progress is reported per file; otherwise progress is skipped to keep tests deterministic.

Progress updates with carriage-return redraws (format: `⏳ Processing N/Total: file`) are only emitted when output is a terminal (`Writer.IsTerminal()`). In non-TTY contexts (piped output), progress text is omitted entirely.

Success output lists the first 5 files with source paths rendered home-relative (~/dir/file). If more than 5 files were added, remaining files are collapsed into "... and N more files" to keep the listing compact.

## Dry run (`lnk add --dry-run`)

`PreviewAdd` runs the validation pass only — walking directories iff `recursive` — and returns the list of files that would be added. It uses the same duplicate-check against the index but performs no moves, no symlinks, no Git operations.

Output displays all files using `displaySourcePath`, which renders paths as home-relative (~/dir/file) to disambiguate files with identical basenames in different directories. The dry-run preview is not truncated; all matched files are shown for full verification before committing changes.

## Remove (`lnk rm <file>`)

`filemanager.Manager.Remove`:

1. Compute `absPath`, then `fs.ValidateSymlinkForRemove(absPath, repoPath)` — must be a symlink whose resolved target lives inside `repoPath`. Otherwise `ErrNotManaged` with a suggestion.
2. Compute `relativePath`, confirm it appears in the index (else `ErrNotManaged`).
3. `os.Readlink` to get the target, `os.Stat` the target for its mode.
4. `os.Remove` the symlink.
5. `tracker.RemoveManagedItem`.
6. `git.Remove(<gitPath>)` — uses `--cached` (and `-r` for directories) so storage stays on disk for the next step.
7. `git.Add(<index file>)`, `git.Commit("lnk: removed <basename>")`.
8. `fs.Move(target, absPath, info)` — restore the original file or directory in place of the symlink.

## Force remove (`lnk rm --force <file>`)

`RemoveForce` is for cases where the symlink is already gone or pointing nowhere useful. It skips the symlink validation, best-effort-removes the symlink, removes the index entry, best-effort `git rm --cached`, commits `lnk: force removed <basename>`, then deletes the storage copy under the repo path with `os.RemoveAll`. There is no original file to restore in this path.
