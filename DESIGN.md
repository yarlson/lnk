# lnk v2 CLI Design

## Status

This document defines the proposed v2 CLI design for `lnk`.

It is intentionally opinionated. The goal is not to preserve every v1 behavior.
The goal is to produce a CLI that is clearer, safer, easier to script, and
harder to misuse.

This document replaces the v1-oriented design framing. It should be treated as
the source of truth for the v2 command model and safety invariants.

## Problem Statement

v1 has a good core idea: keep dotfiles management close to normal Git, keep
state visible on disk, and avoid templates, hidden metadata, or excessive
ceremony.

But the current CLI has design debt:

- `push` is overloaded: it stages, may commit, and then pushes.
- `pull` is overloaded: it updates the repo and mutates the live machine by
  restoring symlinks.
- `doctor` defaults to mutation even though its name suggests inspection.
- `init -r` mixes two workflows: local repo creation and remote clone.
- host scope is modeled as a flag on some commands rather than a consistent
  product concept.
- the same relative path can exist in common and host scopes, producing
  last-write-wins restore behavior on the same machine path.
- paths outside `$HOME` are currently accepted even though restore semantics are
  not correct for them.
- `bootstrap.sh` is arbitrary code execution and should not run implicitly on a
  trust boundary crossing unless the user explicitly asked for it.

v2 exists to fix those problems without changing the product's core shape:

- single binary
- Git-backed repo
- plain-text tracking files
- symlink-based restore
- explicit host-specific storage

## Design Goals

v2 should optimize for:

- clear separation between repo operations and machine-state operations
- a command surface that matches user expectations
- stronger safety around destructive or machine-mutating actions
- consistent host/common behavior
- stable automation behavior
- inspectable on-disk state
- low ceremony

v2 should not optimize for:

- preserving every v1 command shape
- supporting arbitrary system paths outside `$HOME`
- magical host inference
- layering or merging multiple host scopes at once
- adding config systems, templates, or policy engines

## User Model

The primary user is a Git-fluent developer managing dotfiles across multiple
machines.

They think in three separate state domains:

1. the repo
2. the tracked set
3. the current machine's live files and symlinks

The CLI must reflect those domains directly.

## Core Product Model

`lnk` manages two kinds of state:

### 1. Repo State

This is the Git working tree stored under the repo path.

- common tracked entries live in `.lnk`
- host-specific tracked entries live in `.lnk.<host>`
- common managed files live at repo root
- host-specific managed files live under `<host>.lnk/`

### 2. Machine State

This is the current machine's home directory and symlink layout.

Machine restore is not "all scopes". A machine has an **effective profile**:

- common only, or
- common plus one named host scope

That distinction matters. Storage inspection may span many hosts. Machine
application must not.

## Scope Model

v2 keeps explicit host selection, but narrows where it applies.

### Storage Scopes

Used for commands that inspect or mutate tracked storage:

- common
- one host
- all hosts, when explicitly supported

### Machine Profile

Used for commands that mutate the current machine:

- common only
- common + one host

`lnk` must never apply multiple host scopes to one machine in a single restore
operation.

## Hard Invariants

These are product rules, not implementation suggestions.

### Unique Path Ownership

A relative path may be owned by **exactly one** scope across the repo:

- common, or
- one named host

If `.vimrc` is tracked in common, it cannot also be tracked in `work`.
If a user wants to change scope, they must move ownership explicitly.

This closes the largest v1 safety hole.

### Home-Only Paths

v2 only supports paths inside `$HOME`.

If a path is outside `$HOME`, `lnk add` must fail with a direct error.
v2 must not pretend to support such paths.

### Explicit Machine Mutation

Commands that mutate the machine filesystem must be visibly machine-oriented.

`pull` must not relink files.
`restore` and `update` are the only commands that should recreate symlinks on
the current machine.

### Read-Only By Default

Inspection-style commands must not mutate by default.

`doctor` must scan by default and repair only with `--fix`.

### No Implicit Bootstrap On Clone

Cloning a repo crosses a trust boundary.

v2 must not auto-run `bootstrap.sh` after clone unless the user explicitly opts
in, e.g. with `--bootstrap`.

### Durable Repo Identity

v2 should add a minimal repo marker file, e.g. `.lnkrepo`, to replace commit
message heuristics as the primary repo identity check.

The file should be intentionally small and inspectable, for example:

```text
version=2
```

Existing v1 repos can be adopted and upgraded, but repo identity should not be
based primarily on commit subject prefixes in v2.

## Command Strategy

v2 stays mostly flat. The tool is small enough that deep command trees would add
ceremony without improving clarity.

The command surface should be:

```text
lnk init
lnk clone <url> [--bootstrap]

lnk add [--host H] <path...>
lnk move <path> (--to-common | --to-host H)
lnk remove [--host H] <path>
lnk forget [--host H] <path>
lnk list [--host H | --all]

lnk status
lnk diff
lnk commit [-m <message>]
lnk push
lnk pull
lnk restore [--host H] [--dry-run]
lnk update [--host H]

lnk doctor [--host H | --all] [--fix]

lnk bootstrap
```

## Command Semantics

### `lnk init`

Create or adopt a local `lnk` repo at the configured repo path.

- never clones
- never runs bootstrap
- may adopt an existing compatible repo
- writes the v2 repo marker when initializing or upgrading

### `lnk clone <url> [--bootstrap]`

Clone a remote repo into the repo path.

- clone only
- no symlink restore
- no bootstrap unless `--bootstrap` is explicitly passed
- after success, guide the user toward `lnk restore` or `lnk update`

`init -r` is not a v2 command shape.

### `lnk add [--host H] <path...>`

Track one or more paths.

- common when `--host` is omitted
- host-specific when `--host` is provided
- fails if any path is already owned by another scope
- fails for paths outside `$HOME`
- preserves atomic multi-file behavior

### `lnk move <path> (--to-common | --to-host H)`

Move ownership of a tracked path between scopes.

- this is the only supported way to re-scope a tracked path
- must be atomic
- must update storage layout, tracking index, and Git state together

This command exists to make scope transfer explicit rather than smuggling it
through remove-and-readd workflows.

### `lnk remove [--host H] <path>`

Stop managing a path and restore the stored file back to the current machine.

- requires the path to be tracked in the specified scope
- machine-restoring operation
- safe default for normal untracking

### `lnk forget [--host H] <path>`

Stop managing a path without restoring it to the current machine.

- preserves the stored repo copy
- removes tracking only
- safer replacement for the destructive parts of v1 `rm --force`

This exists for cleanup and repair situations where the live path is already
gone or intentionally unmanaged.

### `lnk list [--host H | --all]`

Show tracked storage state.

- default: common
- `--host H`: one host scope
- `--all`: common plus every host scope

This is a storage inspection command, not a health check.

### `lnk status`

Show repo sync state only.

- dirty/clean
- ahead/behind
- remote/upstream state

Never mutates the repo or machine.

### `lnk diff`

Show uncommitted repo diff only.

Never mutates the repo or machine.

### `lnk commit [-m <message>]`

Create a commit from current repo changes.

- stages all repo changes
- creates a commit
- does not push

This replaces the surprising commit side effect currently hidden inside `push`.

### `lnk push`

Push existing commits only.

- never stages
- never commits

If the working tree is dirty, the command should fail clearly and tell the user
to run `lnk commit` or use Git directly.

### `lnk pull`

Pull remote repo changes only.

- never restores symlinks
- never mutates the current machine outside the repo path

If conflicts occur, surface them as Git state. Do not attempt clever recovery.

### `lnk restore [--host H] [--dry-run]`

Apply tracked state to the current machine.

- without `--host`: restore common only
- with `--host H`: restore common, then apply host `H`
- may rename conflicting real files to `.lnk-backup`
- `--dry-run` previews what would be restored and what would be backed up

`restore` is the explicit machine-state command.

### `lnk update [--host H]`

Convenience command for:

1. `lnk pull`
2. `lnk restore [--host H]`

This is the only bundled repo+machine workflow command in v2.

It exists because the workflow is common and legitimate, but it must stay
honest about what it does.

`update` should not support `--dry-run` because `pull` cannot be truthfully
previewed in a meaningful way.

### `lnk doctor [--host H | --all] [--fix]`

Repository and restore-state audit.

Default behavior is read-only scan.

`--fix` enables repair.

The scan should detect at least:

- invalid index entries
- broken or missing symlinks
- duplicate ownership across scopes
- v1 legacy entries outside `$HOME`
- missing repo marker

`doctor --all` is valid because it is an inspection or repair command over
stored scopes, not a machine profile application command.

If duplicate ownership exists, `restore` and `update` should refuse to proceed
until the collision is resolved.

### `lnk bootstrap`

Explicitly run `bootstrap.sh`.

No hidden execution. No implicit "setup completed" assumptions. This is an
escape hatch owned by the repo author and invoked by the user.

## Naming Rationale

The v2 names are chosen to reduce semantic surprise:

- `clone` is distinct from `init`
- `commit` is distinct from `push`
- `pull` is distinct from `restore`
- `update` is the explicit combined workflow
- `forget` communicates "stop tracking but don't restore"

Rejected alternatives:

- deep noun trees like `lnk repo clone` or `lnk sync push`
  - too much ceremony for a small tool
- a single `sync` command as the main workflow primitive
  - too ambiguous
- replacing `--host` with a universal `--scope`
  - too abstract for machine-application commands

## Output Model

The human output model should preserve the product tone:

- terse
- truthful
- grep-friendly
- no celebratory narration
- no meaning carried by color alone

v2 should also improve machine consumption:

- support `--format json` on all structured commands
- reserve plain text as the default human format
- keep `diff` as patch-oriented text output

In JSON mode:

- no hint prose
- no emoji
- stable field names
- stable error shape

## Error Philosophy

Errors should name:

- what failed
- which path, host, or repo state was involved
- what the user should do next when the next step is obvious

Examples:

- dirty working tree on `push`
- duplicate path ownership on `add`
- outside-home path rejection on `add`
- restore blocked by cross-scope collision
- no remote configured on `pull` or `push`

v2 should prefer refusing dangerous ambiguity over guessing.

## Safety Rules

### Restore Safety

When restoring:

- existing real files are never silently overwritten
- conflicting files are renamed to `.lnk-backup`
- backup actions are reported clearly

### Conflict Blocking

If duplicate ownership exists across scopes:

- `doctor` reports it
- `restore` fails
- `update` fails

This state is not recoverable through "best effort" behavior.

### Explicit Destruction

Any future command that deletes stored repo content must require an explicit
opt-in beyond a normal remove flow.

v2 should not smuggle content deletion into cleanup-oriented commands.

## Migration From v1

v2 should provide a compatibility path, but not preserve v1 semantics forever.

### Recommended command mapping

```text
v1                         -> v2
lnk init -r <url>          -> lnk clone <url>
lnk rm                     -> lnk remove
lnk rm --force             -> lnk forget
lnk push "msg"             -> lnk commit -m "msg" && lnk push
lnk pull --host H          -> lnk update --host H
lnk doctor                 -> lnk doctor --fix
```

### Compatibility policy

- keep old names as temporary aliases where feasible
- print deprecation warnings in human mode
- remove deprecated forms from help and docs immediately
- do not preserve old overloaded semantics under old names forever

### Upgrade tasks for existing repos

When a v1 repo is first touched by a v2 binary, the tool should:

1. detect missing repo marker
2. detect duplicate ownership across scopes
3. detect outside-home legacy entries
4. refuse unsafe machine-application commands until blocking issues are fixed
5. offer `doctor --fix` where a safe automatic upgrade exists

## Implementation Guidance

The implementation should preserve the existing strong qualities of the repo:

- plain-text tracking files
- atomic add behavior
- real Git subprocesses
- real filesystem-backed tests

The main implementation changes implied by this design are:

- split current `push` behavior into `commit` and `push`
- split current `pull` behavior into `pull`, `restore`, and `update`
- make `doctor` read-only by default
- add ownership-collision detection across `.lnk` and `.lnk.*`
- reject outside-home paths at add time
- add a repo marker file and migration path
- replace force-remove semantics with explicit `forget`

## Final Standard

The v2 CLI should be judged by this question:

Can a careful engineer predict, from the command name alone, whether the
command changes:

- the repo,
- the current machine,
- both,
- or neither?

If the answer is unclear, the design is still wrong.
