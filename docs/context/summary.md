# lnk — Project Summary

## What

`lnk` is a Git-native dotfiles manager distributed as a single Go CLI. It moves user-selected files into a Git-managed repository, leaves relative symlinks behind in their original location, and uses normal Git operations (push, pull, diff, status) for synchronization across machines. A plain text `.lnk` index records which files are managed; per-host configurations are tracked in parallel `.lnk.<host>` indexes with a sibling `<host>.lnk/` storage directory.

## Architecture

Single Go binary with two layers:

- `cmd/` — Cobra CLI: one file per subcommand, plus a structured `Writer`/`Message` output layer that handles colors, emoji, and quiet mode.
- `internal/` — domain logic split into focused collaborators wired together by the `lnk.Lnk` facade. Each package owns one concern (initializer, tracker, filemanager, syncer, doctor, bootstrapper) and depends on two thin wrappers: `git` (subprocess git) and `fs` (filesystem + symlinks).

A single error type (`lnkerror.Error`) wraps sentinel errors with optional path and suggestion fields; the CLI renders these uniformly via `cmd.DisplayError`.

## Core Flow

1. `lnk init [-r url]` — create or clone the repo at `LNK_HOME` / `XDG_CONFIG_HOME/lnk` / `~/.config/lnk`. With `-r`, automatically locate and run `bootstrap.sh` unless `--no-bootstrap`.
2. `lnk add <files>` — validate, move into the repo, create a relative symlink in place, append to the `.lnk` index, stage, and commit. Multi-file and recursive variants commit atomically with rollback on any failure.
3. `lnk push [msg]` / `lnk pull` — `push` stages-all + commits dirty changes then `git push -u origin`; `pull` does `git pull` then walks the `.lnk` index to recreate any missing or stale symlinks.
4. `lnk doctor [--dry-run]` — find invalid index entries (paths missing in storage) and broken symlinks, then fix them.

## System State

- One deployable artifact: the `lnk` binary.
- Repository state is on the user's filesystem: a single Git working tree at the configured repo path, plus `~/<relative paths>` symlinks pointing into it.
- No daemon, no server, no network state beyond the user's Git remote.

## Capabilities

- Add files or directories, individually or recursively, with dry-run preview.
- Per-host configurations via `--host` (mutually exclusive index + storage namespace).
- Bootstrap script (`bootstrap.sh`) discovery and execution on clone or on demand.
- Sync status (ahead/behind/dirty), uncommitted diff, commit and push, pull and re-link.
- Health checks: detect and repair broken symlinks and stale index entries.
- Backs up pre-existing real files at symlink destinations to `<path>.lnk-backup` instead of overwriting on `pull`.
- Output controls: `--colors auto|always|never`, `--emoji` / `--no-emoji`, `--quiet`/`-q`, plus `NO_COLOR` env.

## Tech Stack

- Go 1.25.
- Cobra for CLI parsing.
- testify for tests.
- System `git` invoked as a subprocess (with timeouts).
- GoReleaser for cross-platform binary releases (Linux, macOS, Windows × amd64/arm64) and Homebrew tap publishing.
- GitHub Actions for CI (test/lint/build), release on tag push, and PR validation of the GoReleaser config.
