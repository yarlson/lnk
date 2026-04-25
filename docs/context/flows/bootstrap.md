# Bootstrap Flow

Bootstrap is an opt-in escape hatch for "things you want done after cloning that aren't dotfiles" — installing packages, setting up shells, configuring app preferences. It is a single optional `bootstrap.sh` at the repo root.

## Discovery

`bootstrapper.Runner.FindScript`:

1. Confirms the repo is a Git repository (else `ErrNotInitialized`).
2. Returns `"bootstrap.sh"` if `<repo>/bootstrap.sh` exists, else `""` with no error.

The CLI uses the empty-string return to mean "no bootstrap configured" rather than a hard failure.

## Execution

`bootstrapper.Runner.RunScript(scriptName, stdout, stderr, stdin)`:

1. Stat `<repo>/<scriptName>` — `ErrBootstrapNotFound` if missing.
2. `os.Chmod(scriptPath, 0755)` — `ErrBootstrapPerms` on failure.
3. `exec.Command("bash", scriptPath)` with `cmd.Dir = repoPath` and the supplied stdio.
4. Run; on non-zero exit, `ErrBootstrapFailed` with the underlying error string as a suggestion.

The script always runs through `bash`, regardless of the file's shebang or executable bit. Working directory is the repo path so the script can reference its sibling files with relative paths.

## Trigger points

- `lnk init -r <url>` — runs automatically after a successful clone unless `--no-bootstrap`. A failure here is reported with a warning but does not roll back the clone; the user is told to retry with `lnk bootstrap`.
- `lnk bootstrap` — runs the script on demand. Prints a "no bootstrap script found" message with a sample template if the file is absent.

## I/O wiring

The CLI passes `os.Stdin`, `os.Stdout`, `os.Stderr` through to the script so it behaves like any other shell command: interactive prompts work, color codes pass through, and progress indicators render live. The lnk Writer's quiet/emoji/color settings do **not** filter the script's output.

## Boundaries

- lnk does not parse, lint, or sandbox the script. The user owns its content.
- lnk does not record execution state. Re-running `lnk bootstrap` re-runs the full script.
- A bootstrap script is repo-wide, not per-host. There is no `bootstrap.<host>.sh` convention.
