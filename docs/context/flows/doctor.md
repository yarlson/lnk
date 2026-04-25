# Doctor Flow

`lnk doctor` scans for two issue classes and either reports them (`--dry-run`) or fixes them. Scope is determined by `--host` (default: common configuration).

## Issues detected

### Invalid entries

`doctor.findInvalidEntries` flags an index entry as invalid if either:

- The cleaned path begins with `..` or is absolute. This means the entry would escape the storage root — no legitimate add produces these.
- The corresponding stored file doesn't exist at `<HostStoragePath()>/<relativePath>`.

### Broken symlinks

`doctor.findBrokenSymlinks` flags an entry whose stored file _does_ exist but whose `~/<relativePath>` is not a valid symlink to it. Validity is checked with `syncer.IsValidSymlink`, which resolves relative link targets against the link's directory and compares absolute paths. Entries with paths that escape storage are skipped here (already covered as invalid entries).

## Result shape

```go
type Result struct {
    InvalidEntries []string
    BrokenSymlinks []string
    BackedUp       []string  // only populated by Fix, not Preview
}
```

`HasIssues()` and `TotalIssues()` are convenience helpers used by the CLI for messaging. `BackedUp` tracks managed items whose pre-existing real files were renamed to `.lnk-backup` during symlink restoration.

## Preview (`lnk doctor --dry-run`)

`Checker.Preview` runs both scans and returns the `Result` without mutating anything. The CLI renders broken symlinks and invalid entries in two separate sections and tells the user to re-run without `--dry-run` to apply.

## Fix (`lnk doctor`)

`Checker.Fix`:

1. Run `Preview` to compute the `Result`.
2. If there are no issues, return early.
3. If any broken symlinks were found, call `syncer.RestoreSymlinks()`. This is the same code path used by `lnk pull` and includes the `.lnk-backup` rename safety net for pre-existing real files.
4. If any invalid entries were found, rewrite the index without them, `git add` the index file, and commit with `lnk: cleaned N invalid entry|entries` (singular/plural picked by count).

The CLI then renders sections for fixed broken symlinks (including any backup notice for files renamed to `.lnk-backup`), removed invalid entries, and a summary of all fixes applied. It suggests `lnk push` to sync the cleanup commit to the remote.

## Notes on scope

- `doctor` operates on exactly one index file (common or one host) per invocation. Use multiple invocations to scan all hosts; there is no `--all`.
- A managed path that exists in both common and host scopes — possible but unusual — is checked independently in each scope.
- `Fix` preserves the order in which `Preview` was called; if a broken symlink restoration fails, invalid-entry pruning is not attempted in that run.
