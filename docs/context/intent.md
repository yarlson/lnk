# High-Level Intent

This document is written for agents.

Its purpose is not to guess what the maintainer "probably meant." Its purpose is
to separate:

- facts established directly by code, tests, and existing docs
- high-confidence inferences that are useful but not guaranteed
- unresolved intent questions that should not be guessed past

## Confidence Boundary

It is not possible to be 100% certain of maintainer intent from code alone.
What _is_ possible is to make the uncertainty explicit.

Agents should therefore treat this document as:

- 100% confidence about statements in **Code-Level Facts**
- less-than-100% confidence about statements in **High-Confidence Inferences**
- 0% authority to invent answers for statements in **Open Questions**

If a future change depends on an open question, ask the maintainer instead of
inferring policy from taste.

## Code-Level Facts

### What this repo is

- `lnk` is a single-binary Go CLI.
- The repository and README position it as a Git-native tool for managing
  dotfiles by moving them into a Git repo and replacing the originals with
  symlinks.
- The managed repo path resolves in this order:
  `LNK_HOME`, then `XDG_CONFIG_HOME/lnk`, then `~/.config/lnk`.
- There is no daemon, server, database, or custom sync service. Networked
  behavior is Git subprocess execution.

### What the tool definitely does

- `init` creates or clones the managed repo.
- `add` moves a file or directory into the repo, writes a tracking entry, and
  replaces the source path with a relative symlink.
- `rm` removes a tracked item and restores the stored file or directory to the
  original path.
- `push` stages all repo changes, commits if needed, and pushes with Git.
- `pull` runs `git pull` and then restores symlinks for the selected scope.
- `doctor` detects broken symlinks and invalid tracking entries and can fix them.
- `bootstrap` runs `bootstrap.sh` from the repo root when present.

### On-disk state model

- Common tracked items are indexed in `.lnk` and stored at the repo root.
- Host-scoped tracked items are indexed in `.lnk.<host>` and stored under
  `<host>.lnk/`.
- Tracking files are plain text: one path per line.
- Host scoping is explicit runtime selection via `--host`.
- The CLI does not automatically apply the current hostname during normal
  operations.

### Safety properties that are actually enforced

- Multi-file add is transactional: it validates first, then mutates the
  filesystem, then stages/commits, and rolls back completed work on failure.
- `rm` refuses to operate unless the target path is a symlink whose resolved
  target lives inside the managed repo.
- `pull` and `doctor` do not overwrite existing real files at restore
  destinations; they rename them to `.lnk-backup`.
- `init` refuses to adopt an existing Git repo unless it passes the current
  `IsLnkRepository` heuristic.

### Architectural facts

- `cmd/` owns Cobra CLI and output behavior.
- `internal/lnk` is the facade the CLI talks to.
- Core behavior is split across `initializer`, `tracker`, `filemanager`,
  `syncer`, `doctor`, and `bootstrapper`.
- Git and filesystem behavior are wrapped in `internal/git` and `internal/fs`.
- Tests use real Git repositories, real filesystem operations, and real symlinks.

## Behavioral Traps Agents Must Not Miss

These are factual consequences of the current implementation. They are not
preferences.

### Common and host scopes can conflict on the same target path

- The same relative path can exist in `.lnk` and `.lnk.<host>`.
- Both scopes restore to the same destination path: `~/R`.
- There is no precedence or merge model.
- In practice, whichever restore path runs last wins the symlink at `~/R`.

Agents should not describe common and host scopes as composable layers. They are
separate storage/index scopes that can contend for the same live path.

### Paths outside `$HOME` are not semantically reliable

- `add` accepts paths outside `$HOME` by storing them as `/`-stripped relative
  paths such as `etc/config`.
- `pull` and `doctor` restore using `filepath.Join(homeDir, relativePath)`,
  which targets `~/etc/config`, not `/etc/config`.

Agents should treat outside-home path management as inconsistent current
behavior, not as a supported product guarantee.

### "lnk repository" detection is heuristic, not identity-proof

- A repo is considered lnk-compatible if it has no commits, or if every commit
  subject starts with `lnk:`.
- That means repo adoption is based on commit-message shape, not on a dedicated
  repo marker.

Agents should not assume `lnk:` commit prefixes are a robust repository identity
scheme. They are a current heuristic with possible false positives and false
negatives.

### `push` includes more than files moved by `lnk`

- `push` stages all repo changes with `git add -A`.
- That includes manually added scripts, docs, bootstrap files, and any other
  tracked or untracked changes inside the managed repo.

Agents should not assume the repo only contains moved dotfiles.

### Bootstrap is an execution escape hatch

- `bootstrap.sh` is arbitrary shell executed with `bash`.
- It is repo-wide, not host-specific.
- The tool does not record bootstrap execution state.
- Re-running bootstrap re-runs the full script.

Agents should treat bootstrap as an operational extension point, not a managed
configuration primitive.

## High-Confidence Inferences

These are useful working assumptions, but they are not guaranteed product law.

### The tool is intentionally opinionated

- The current design strongly favors explicit, small workflows over flexible
  abstraction layers.
- The repo avoids config schemas, template systems, and extra metadata formats.

### The desired UX is "use Git normally"

- Command naming, README language, and implementation all point toward making
  dotfile sync feel like normal Git plus symlink restoration, not like a new
  domain-specific platform.

### Human-inspectable state matters

- The storage layout is plain on disk.
- Tracking is text-based and easy to inspect.
- Relative symlinks suggest portability and inspectability are valued.

## Safe Agent Policy

Until the maintainer answers the open questions below, agents should default to:

- treating `lnk` as a dotfiles-focused Git CLI, not a general system-state tool
- preserving explicit `--host` selection behavior
- avoiding new config formats or metadata stores
- avoiding any feature that implies precedence-merging between common and host
  scopes
- treating outside-home path support as undefined behavior that should not be
  expanded without confirmation
- preserving the current safety posture around rollback, restore, and backup

## Open Questions For The Maintainer

These are the places where code does not fully recover product intent.

1. Are paths outside `$HOME` supposed to be supported at all, or should the
   product explicitly reject them?
2. Is `lnk` intended to stay strictly a dotfiles manager, or is a broader
   "machine setup repo" scope intentional?
3. Is explicit `--host` selection a core design principle, or merely the
   current implementation?
4. Is the `lnk:` commit-prefix rule meant to be a user-visible contract, or
   only an internal safety heuristic for repo adoption?
5. Is "no config / no templates / no ceremony" a hard product boundary, or just
   current positioning?
6. Should agents treat common and host entries for the same relative path as a
   legitimate workflow, or as a sharp edge to avoid?

## Working One-Sentence Summary

`lnk` is currently a Git-native dotfiles CLI that stores tracked files in a
plain repo, restores them via symlinks, and keeps synchronization and recovery
close to normal Git workflows.
