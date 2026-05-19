# lnk v2 Design Draft

## Status

This document defines the draft v2 product and architecture direction for
`lnk`.

The old v2 host-scope design is replaced. Deleted Go history is migration
reference only, not the product contract.

v2 is a Rust implementation line. It should keep the same product qualities
that made v1 useful: Git-backed state, inspectable files, low ceremony, and no
daemon. It should not preserve every v1 behavior.

## Product Definition

`lnk` is a Git-backed userland machine setup repo.

It manages:

- ordered profiles
- files owned by those profiles
- explicit scripts owned by those profiles
- local machine selection and applied-state records outside Git

It does not manage:

- system paths outside `$HOME`
- packages as a declarative abstraction
- services
- secrets
- sudo workflows
- profile inheritance
- hidden script execution

The primary user is a Git-fluent developer who wants one repo to carry
machine setup state across machines without turning that repo into a private
configuration platform.

## Primary Workflow

The common path should require one main command after the repo exists:

```text
lnk apply base mac work
```

That command:

- validates the selected profiles
- stores them as this machine's profile list
- applies their files
- records applied targets in local state

After that, normal repeated use is:

```text
lnk apply
lnk status
lnk run setup
```

`lnk apply` without profile arguments uses this machine's stored profile list.
Scripts stay explicit through `lnk run`; applying files never runs scripts.

Preview uses the same command:

```text
lnk apply --dry-run base mac work
```

There is no separate "select profiles" command and no separate "plan" command.
The tool has one obvious mutation path, and dry-run is the non-mutating version
of that path.

## Design Goals

v2 should optimize for:

- low cognitive load
- predictable mutation boundaries
- reviewable Git state
- inspectable local machine state
- explicit profile composition
- easy migration from v1
- simple implementation boundaries

v2 should not optimize for:

- preserving old command names
- supporting arbitrary system configuration
- modeling package managers or service managers
- clever automatic inference
- a generic task runner
- compatibility with every v1 edge case

## State Model

`lnk` has two state domains.

### Repo State

Repo state lives in Git and is shared across machines.

It includes:

- repo format metadata
- profile definitions
- profile-owned files
- profile-owned scripts
- explicit override declarations
- copy-mode and target-mapping metadata

Repo state is desired setup state. It must be safe to review in normal Git
diffs.

### Local Machine State

Local machine state lives outside Git under XDG state.

It includes:

- active ordered profiles for this machine
- applied target records
- source profile and source path for each applied target
- mode used for each target (`symlink` or `copy`)
- hashes for copied files so drift can be detected

Local state must not be stored in the setup repo. Active profiles and applied
file records are facts about one machine, not facts about the shared repo.

## Repository Layout

The repo root contains a minimal `lnk.toml`.

Schema:

- `format` is required and must be integer `1`.
- `name` is optional display metadata.
- unknown fields are errors.

```toml
format = 1
name = "machine-setup"
```

`lnk init` writes both fields. The default `name` is the repo directory name.
Users may edit or remove `name`; behavior must not depend on it.

Profiles live under `profiles/<name>/`:

```text
lnk.toml
profiles/
  base/
    profile.toml
    files/
      .zshrc
      .config/nvim/init.lua
    scripts/
      setup
      check
  mac/
    profile.toml
    files/
      .config/karabiner/karabiner.json
  work/
    profile.toml
    files/
      .gitconfig
    scripts/
      setup
```

`profile.toml` is optional. A missing `profile.toml` means an empty manifest.
Convention is the default; manifests exist only when behavior stops being
obvious.

Profile names come from directory names, not from manifest fields. A profile
manifest must not rename its directory.

Profile manifest schema:

```toml
[files."<source-path-under-files>"]
target = "<home-relative-target>"     # optional; defaults to the source path
mode = "symlink"                      # optional; symlink or copy
overrides = "<profile>"               # optional explicit override of one profile

[scripts."<script-name>"]
platforms = ["macos", "linux"]        # optional; macos, linux, windows
```

Rules:

- `files` keys are paths relative to `profiles/<profile>/files/`.
- `target` values are paths relative to `$HOME`.
- `mode` defaults to `symlink`.
- `overrides` names the earlier profile whose same target this file intentionally
  replaces.
- `scripts` keys are script names under `profiles/<profile>/scripts/`.
- `platforms` gates script execution by Rust's `std::env::consts::OS`.
- unknown fields are errors.

## Profiles

A machine uses an ordered list of profiles:

```text
base mac work
```

Profiles are applied in order. Order is meaningful, but silent override is not
allowed.

There is no special `common` or host scope in v2. `base` is only a profile name.
Users may choose another name.

Profiles do not inherit from other profiles. Composition is expressed by the
machine's ordered active profile list.

## Files

Regular files under `profiles/<name>/files/` are auto-included.

Default behavior:

- source path: `profiles/<profile>/files/<relative-path>`
- target path: `$HOME/<relative-path>`
- mode: `symlink`

Manifest entries are required only for exceptions:

- copy mode
- target remapping
- explicit override
- future metadata that changes behavior

Example:

```toml
[files.".gitconfig"]
mode = "copy"

[files.".ssh/config"]
target = ".ssh/config"
overrides = "base"
```

Targets must stay inside `$HOME`. v2 does not support system paths.

### Symlink Mode

Symlink mode is the default.

`lnk apply` creates a symlink at the target path pointing at the profile-owned
source file in the repo.

### Copy Mode

Copy mode must be explicit in `profile.toml`.

Copy mode writes a real file to the target path. Because copies can drift,
`lnk` records source hashes in local state so `lnk status` can report whether
the live copy differs from the repo source.

Copy mode does not imply automatic sync back to the repo.

## Overrides And Conflicts

If two selected profiles claim the same target path, apply fails unless the
later profile explicitly declares an override.

Default conflict behavior is a hard error.

Explicit override makes profile order visible and reviewable:

```toml
[files.".gitconfig"]
overrides = "base"
```

The override declaration belongs to the overriding profile.

The target path is implied by the file entry. Repeating it inside `overrides`
would make users restate information the tool already knows.

This prevents ordered profiles from becoming silent last-write-wins overlays.

## Existing Target Safety

If `lnk apply` sees an existing real file at a target path and that target is
not recorded as managed by `lnk`, it must stop and report a conflict.

Mutation through conflicts requires an explicit backup policy:

```text
lnk apply --backup
```

`--backup` renames conflicting files before applying.

Backups are written next to the original target with a timestamped suffix:

```text
<target>.lnk-backup-YYYYMMDD-HHMMSS
```

Timestamped backups avoid collisions and make repeated apply attempts
predictable.

For copy-mode files, apply uses local state hashes to classify drift:

- source changed, live target unchanged since last apply: update the copy
- live target changed, source unchanged: report local drift and stop
- both source and live target changed: report divergent copy and stop

`--backup` backs up the live target before writing the new copy.

There is no `--overwrite`. If a user wants to discard a live file, they should
remove it themselves or use `--backup` and then delete the backup. A setup tool
should not make destructive replacement feel like a normal mode.

## Scripts

Scripts are profile-owned and explicit.

They live by convention under:

```text
profiles/<profile>/scripts/<script-name>
```

Examples:

```text
profiles/base/scripts/setup
profiles/base/scripts/check
profiles/work/scripts/setup
```

Scripts are discovered by convention. `profile.toml` may add metadata such as
platform gating when needed:

```toml
[scripts.setup]
platforms = ["macos"]
```

Scripts never run as a side effect of `lnk apply`.

Platform gating is enforced by `lnk run`. A script whose `platforms` list does
not include the current platform is skipped with a terse message. `lnk doctor`
validates platform names. `lnk profiles` does not need to explain script gating
unless a future verbose mode is added.

Supported execution forms:

```text
lnk run check
lnk run work:setup
```

`lnk run check` runs the named script across active profiles in order, skipping
profiles that do not define it.

`lnk run work:setup` runs exactly one script from one profile.

## Local State

Local state is a human-readable TOML file under XDG state, for example:

```text
$XDG_STATE_HOME/lnk/machine.toml
```

If `XDG_STATE_HOME` is unset, use:

```text
~/.local/state/lnk/machine.toml
```

The file records active profiles and applied targets.

Schema:

- `format` is required and must be integer `1`.
- `active_profiles` is an ordered list of profile names.
- `targets` is a list of applied target entries.
- each target path may appear at most once.
- unknown fields are errors.

Example:

```toml
format = 1
active_profiles = ["base", "mac", "work"]

[[targets]]
target = ".zshrc"
profile = "base"
source = "profiles/base/files/.zshrc"
mode = "symlink"

[[targets]]
target = ".gitconfig"
profile = "work"
source = "profiles/work/files/.gitconfig"
mode = "copy"
source_hash = "sha256:..."
applied_hash = "sha256:..."
```

For symlink targets, `source_hash` and `applied_hash` are omitted.

For copy targets:

- `source_hash` is the repo source hash at the time of apply.
- `applied_hash` is the live target hash immediately after apply.

The state file should be rewritten deterministically, sorted by target path.

## Repo Location

v2 resolves the setup repo path in this order:

1. `LNK_HOME`
2. `XDG_CONFIG_HOME/lnk`
3. `~/.config/lnk`

There is no public `--repo` flag and no `LNK_REPO`.

Tests should isolate the repo path through environment variables rather than
adding product-facing configuration knobs.

## Git Model

`lnk` uses system `git`.

Rust v2 must shell out to `git` via `std::process::Command`. It must not replace
Git behavior with a Git library.

Commands that inspect or mutate repo state must preflight that `git` is
installed and return a direct error when it is missing.

Help, version output, and shell completion must not require `git`.

`lnk` should not become a full Git porcelain. Users can still use Git directly.

## Command Surface

The v2 command surface should stay flat:

```text
lnk init
lnk profiles
lnk apply [profile...] [--dry-run] [--backup]
lnk status
lnk run <script>
lnk run <profile>:<script>
lnk doctor
lnk migrate v1
lnk migrate v1 --write --yes
```

### `lnk init`

Initialize a setup repo at the resolved repo path.

It creates:

- `lnk.toml`
- `profiles/base/files/`
- `profiles/base/scripts/`

If the repo path is not already a Git repository, `init` runs `git init` and
creates an initial commit:

```text
lnk: initialize v2 setup repo
```

If the repo path is an existing Git repository with commits and no `lnk.toml`,
`init` refuses and tells the user to use `lnk migrate v1` for old repos or an
empty directory for a new v2 repo. It must not guess whether an arbitrary Git
repo is safe to adopt.

It must not run scripts.

### `lnk profiles`

List profiles found under `profiles/`.

This is repo inspection only. It should also show this machine's stored profile
list when local state exists. It must not mutate the repo or machine.

### `lnk apply [profile...]`

Apply files for the selected profile list.

Selection rules:

- with profiles: validate, store, and apply the provided ordered list
- without profiles: use active profiles from local state
- with `--dry-run`: show the plan without storing profiles or changing files

When profiles are passed, local state is updated only after a successful
non-dry-run apply. A failed apply must not change this machine's stored profile
list.

The apply plan should include:

- files to symlink
- files to copy
- explicit overrides
- unmanaged target conflicts
- copied-file drift when known
- missing profiles or invalid metadata

`apply` mutates files only. It never runs scripts.

Existing unmanaged conflicts block apply unless `--backup` is provided.

If no profiles are passed and no local profile list exists, `apply` fails with a
direct next step:

```text
run 'lnk apply <profile...>' first
```

### `lnk status`

Report repo and local machine state.

At minimum it should report:

- active profiles
- applied targets
- missing symlinks
- stale or wrong symlinks
- copied files whose live content drifted from the recorded source hash
- repo dirty state

`status` must not mutate.

### `lnk run <script>`

Run the named script across active profiles in order.

Profiles without the script are skipped.

Script execution should stream stdout and stderr directly.

### `lnk run <profile>:<script>`

Run exactly one script from exactly one profile.

This is the precise debugging and one-off execution path.

### `lnk doctor`

Audit repo and local machine health.

It should detect:

- invalid root `lnk.toml`
- invalid profile manifests
- target paths escaping `$HOME`
- duplicate target claims without explicit override
- override declarations that do not correspond to an earlier profile target
- local state targets whose source no longer exists
- copied-file drift
- missing or wrong symlinks

`doctor` is read-only by default. Any future repair mode must be explicit.

## Output And JSON

Human output should be terse and truthful.

Structured commands should support JSON output:

- `profiles`
- `apply`
- `status`
- `doctor`

Errors should have a stable JSON shape in JSON mode.

`diff`, if added later, should remain patch-oriented text.

Do not invent large result schemas for simple acknowledgements. JSON exists for
automation, not ceremony.

JSON mode is selected with a command-local advanced flag:

```text
--json
```

Default output is human text.

Rules:

- successful JSON results are written to stdout
- JSON errors are written to stderr
- JSON output contains no hints, color, emoji, or prose-only fields
- only commands with structured output should expose `--json`
- commands like `run` should not grow JSON output until there is a real
  automation use case

Stable error shape:

```json
{
  "error": {
    "code": "conflict.unmanaged_target",
    "message": "target already exists and is not managed by lnk",
    "target": ".zshrc"
  }
}
```

## Rust Architecture

The Rust implementation should use focused modules rather than one large
service file.

Suggested module shape:

```text
cli
app
repo
profile
manifest
paths
ownership
apply
state
status
doctor
scripts
git
errors
migrate
```

Rules:

- CLI code parses arguments and formats output.
- Domain modules return typed results.
- Git subprocess logic lives in `git`.
- Path normalization and `$HOME` boundary checks live in `paths`.
- Profile conflict and override logic lives in `ownership`.
- Local state read/write lives in `state`.
- Migration logic lives in `migrate`.

Avoid reviving the deleted Go v2 shape. Large command or service files are a
known failure mode for this repo.

## Test Strategy

Parity and product behavior should be tested black-box at the CLI level.

Tests should create disposable homes and repos:

- temporary `HOME`
- temporary `XDG_CONFIG_HOME`
- temporary `XDG_STATE_HOME`
- optional bare Git remotes

They should assert:

- stdout and stderr where behavior depends on output
- exit codes
- repo layout
- local state TOML
- symlink targets
- copied-file content and drift detection
- Git commits and dirty state where relevant

Rust should also have focused unit tests for dense safety logic:

- path normalization
- `$HOME` boundary enforcement
- profile ordering
- override validation
- unmanaged target conflict classification
- local state parsing
- copied-file hash comparison

Do not port Go tests concept-for-concept. Test the v2 product contract.

## Migration From v1

Migration is explicit and plan-first.

Commands:

```text
lnk migrate v1
lnk migrate v1 --write --yes
```

`lnk migrate v1` prints the plan only.

`lnk migrate v1 --write --yes` mutates the current repo.

Migration rules:

- refuse a dirty working tree
- create backup tag `lnk-v1-backup-YYYYMMDD-HHMMSS`
- transform the current checked-out branch directly
- preserve Git history
- preserve Git remotes
- create one migration commit
- do not run scripts

v1 mapping:

- `.lnk` becomes profile `base`
- `.lnk.<host>` becomes profile `<host>`
- common files move to `profiles/base/files/`
- host files move to `profiles/<host>/files/`
- host/common path collisions become explicit host overrides
- v1 `bootstrap.sh` becomes `profiles/base/scripts/setup`
- migrated files default to symlink mode

Old v1 tracking files are removed in the migration commit:

- remove `.lnk`
- remove `.lnk.<host>`
- remove emptied `<host>.lnk/` directories after moving managed files

Other non-managed repo files are left in place. Migration should not invent a
`legacy/` directory because that adds cleanup work and uncertainty.

After migration, local active profiles should default to:

- `base`
- plus the current hostname profile if that profile exists

The command must print that local state was written because local state is
outside Git.

## Decision Log

- v2 is a redesign, not a narrow cleanup of the existing Go v2 prototype.
- v2 is a Rust implementation line.
- `lnk` is a userland machine setup repo, not only a dotfiles symlink manager.
- Profiles are the primary unit.
- A machine uses an ordered profile list.
- Profile conflicts require explicit override declarations.
- There is one manifest per profile, located with the profile.
- Files live under their owning profile.
- File target mapping is implicit by default, explicit for exceptions.
- File mode defaults to symlink; copy mode must be explicit.
- Scripts are named and explicit.
- Scripts are discovered by convention and optionally annotated in manifests.
- Machine profile selection lives outside Git.
- Local state is TOML under XDG state.
- Repo identity is root `lnk.toml`, not a hidden marker.
- `apply <profile...>` is the primary workflow: it stores this machine's profile
  list and applies files.
- `apply --dry-run` replaces a separate `plan` command.
- Script execution uses `lnk run`.
- Existing unmanaged live files block apply unless `--backup` is passed.
- There is no `use` command and no `--save`; profile selection is part of
  `apply`.
- There is no `--overwrite`; destructive replacement is intentionally omitted.
- Migration is explicit, plan-first, and can mutate only with `--write --yes`.
- Migration preserves same Git history and remotes.
- `profile.toml` is optional, even when a profile has scripts.
- Script platform gating is enforced by `run` and validated by `doctor`.
- Backups use timestamped side-by-side files.
- `init` creates an initial Git commit for new repos.
- Migration removes old v1 tracking files instead of preserving a legacy copy.
- JSON mode uses command-local `--json`.

## Non-Goals

v2 must not include:

- package manager abstraction
- service management
- secret management
- sudo escalation
- system-path targets outside `$HOME`
- profile inheritance
- hidden or automatic script execution
- automatic copy-mode inference
- host-name-to-profile mapping stored in Git
- compatibility aliases for every v1 command
- deep command trees
- destructive overwrite mode for apply

## Open Risks

- Copy mode can drift. The local state hash model must be clear enough that
  users understand what `status` is reporting.
- Migration transforms the current branch directly. The mandatory backup tag and
  dirty-tree refusal are not optional.
- Platform metadata can grow into a bad task runner if scripts become too
  configurable.
- JSON output can become schema bureaucracy. Keep it tied to structured
  commands only.

## Implementation Decisions To Preserve

- Keep schemas minimal and strict: required fields are few, unknown fields are
  errors.
- Prefer directory convention over manifest entries for normal symlinked files.
- Keep `profile.toml` optional. Requiring empty manifests is clerical work.
- Do not add repo-local machine selection. Active profiles stay outside Git.
- Do not add a separate profile-selection command unless a real workflow needs
  it. `apply <profile...>` already selects and applies.
- Do not add a public `--repo` escape hatch. Use environment isolation in tests.
- Do not run scripts from `init`, `apply`, or `migrate`.
- Keep copy drift blocking by default. Copy mode is allowed, but local edits are
  user data and must not be silently replaced.
- Keep migration destructive only after `--write --yes`, dirty-tree refusal, and
  backup tag creation.

## Final Standard

The design is acceptable only if a user can predict from the command name
whether it changes:

- the setup repo
- local machine state
- live files
- scripts/processes
- nothing

If that is unclear, the design is still too clever.
