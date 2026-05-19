# lnk Rust CLI Implementation Plan

This plan implements the product contract in `DESIGN.md` without reviving the
deleted Go design. Each phase is independently shippable, keeps the command
surface flat, and ends with concrete checks.

## Current Preflight

- `rustc --version`: `rustc 1.93.1 (01f6ddf75 2026-02-11)`
- `cargo --version`: `cargo 1.93.1 (083ac5135 2025-12-15)`
- `git --version`: `git version 2.53.0`
- `cargo nextest --version`: unavailable locally

`cargo nextest` must be installed before Rust implementation work is finalized,
because `AGENTS.md` requires it and nextest does not run doctests; doctests stay
a separate `cargo test --doc --all-features` gate.

## Plan Rules

- `DESIGN.md` is the source of truth.
- One Rust package, many small modules. Do not create a workspace unless a real
  second package exists.
- No CI, release automation, package-management features, services, secrets,
  sudo, profile inheritance, public `--repo`, `LNK_REPO`, `use`, `plan`,
  `--save`, or `--overwrite`.
- Every phase must preserve earlier behavior unless the phase explicitly changes
  it and updates tests.
- Every filesystem mutation must be internally plan-first: compute, validate,
  then mutate.
- Every command that inspects or mutates repo state must preflight system `git`.
- Human output stays terse. JSON exists only for `profiles`, `apply`, `status`,
  and `doctor`.

## Web-Validated Sources

- Rust 2024 is stable since Rust 1.85.0:
  https://blog.rust-lang.org/2025/02/20/Rust-1.85.0/
- Cargo manifest/package structure:
  https://doc.rust-lang.org/cargo/reference/manifest.html
- Cargo workspaces are for multiple related packages:
  https://doc.rust-lang.org/cargo/reference/workspaces.html
- `std::process::Command` requires program and args to be passed separately:
  https://doc.rust-lang.org/std/process/struct.Command.html
- `std::fs` warns about TOCTOU and recommends atomic operations where possible:
  https://doc.rust-lang.org/std/fs/index.html
- Rust error model uses `Result`/error traits for expected runtime failures:
  https://doc.rust-lang.org/core/error/index.html
- XDG state/config defaults and absolute-path rule:
  https://specifications.freedesktop.org/basedir-spec/latest/
- `clap` is the current Rust CLI parser family, latest release checked in 2026:
  https://github.com/clap-rs/clap
- `assert_cmd` supports black-box CLI assertions for binary, env, stdout,
  stderr, and exit codes:
  https://docs.rs/assert_cmd/latest/assert_cmd/
- `cargo-nextest` provides faster isolated test execution and requires doctests
  to run separately:
  https://www.nexte.st/
- `git init` behavior and initial branch defaults:
  https://git-scm.com/docs/git-init
- `git status --porcelain` is stable for scripts:
  https://git-scm.com/docs/git-status
- TOML crate supports serde-compatible TOML parsing/serialization:
  https://docs.rs/toml
- Serde derive supports typed serialization/deserialization:
  https://serde.rs/derive.html

## Phase 1 - Rust Package Skeleton

Goal: create the smallest compiling Rust CLI package.

Steps:

1. Add `Cargo.toml` for a single package named `lnk`, edition `2024`, with one
   binary and one library.
2. Add `src/main.rs` as a thin process entrypoint.
3. Add `src/lib.rs` that exposes only the modules needed by the CLI.
4. Add placeholder modules matching `DESIGN.md`: `cli`, `app`, `repo`,
   `profile`, `manifest`, `paths`, `ownership`, `apply`, `state`, `status`,
   `doctor`, `scripts`, `git`, `errors`, and `migrate`.
5. Keep every placeholder small; no service module.

Validation:

- `cargo check --all-targets --all-features`
- `cargo fmt --all --check`

Web validation:

- Rust 2024 is supported by current stable Rust.
- Cargo manifest docs support this as one package; workspace docs confirm a
  workspace is unnecessary until multiple packages exist.

## Phase 2 - CLI Shape And Output Boundary

Goal: implement the exact command surface with no domain behavior yet.

Steps:

1. Add `clap` derive parsing for:
   - `lnk init`
   - `lnk profiles [--json]`
   - `lnk apply [profile...] [--dry-run] [--backup] [--json]`
   - `lnk status [--json]`
   - `lnk run <script-or-profile-script>`
   - `lnk doctor [--json]`
   - `lnk migrate v1`
   - `lnk migrate v1 --write --yes`
2. Reject unsupported flags by default through the parser.
3. Keep `main.rs` to argument parsing, dispatch, exit-code mapping, and
   output rendering.
4. Add a typed command enum that domain code receives after parsing.
5. Add snapshot-free tests for help, invalid commands, and argument routing.

Validation:

- `cargo check --all-targets --all-features`
- `cargo nextest run --all-features` after nextest is installed
- `cargo test --doc --all-features`

Web validation:

- `clap` is current and suited to declarative Rust CLI parsing.
- `assert_cmd` supports binary invocation, env setup, stdout/stderr, and exit
  code assertions for black-box CLI tests.

## Phase 3 - Errors And JSON Error Rendering

Goal: make failure behavior stable before adding mutations.

Steps:

1. Add a domain `Error` enum with stable codes.
2. Add CLI-only human rendering and JSON rendering.
3. Add stable JSON error shape:
   `{"error":{"code":"...","message":"...", ...}}`.
4. Keep error construction in domain modules and formatting at the CLI boundary.
5. Add tests for human errors and JSON errors on parseable failing commands.

Validation:

- `cargo check --all-targets --all-features`
- `cargo nextest run --all-features`

Web validation:

- Rust error docs distinguish expected runtime failures from panics; `lnk`
  should use typed `Result` errors for user/repo/filesystem failures.

## Phase 4 - Path And Environment Resolution

Goal: centralize all `$HOME`, XDG, and repo-path rules before reading repo data.

Steps:

1. Implement `paths` helpers for:
   - home directory from `HOME`
   - repo path resolution: `LNK_HOME`, then `XDG_CONFIG_HOME/lnk`, then
     `~/.config/lnk`
   - state file path: `XDG_STATE_HOME/lnk/machine.toml`, then
     `~/.local/state/lnk/machine.toml`
   - absolute XDG env validation
   - home-relative target normalization
   - `$HOME` boundary enforcement
2. Reject empty, absolute, parent-escaping, and non-normalized target paths.
3. Add unit tests for path normalization and boundary failures.

Validation:

- `cargo check --all-targets --all-features`
- `cargo nextest run --all-features`

Web validation:

- XDG spec defines config/state defaults and says relative XDG env paths are
  invalid and should be ignored.
- `std::fs` docs call out TOCTOU risks, so path checks must be paired with
  later mutation checks rather than treated as permanent proof.

## Phase 5 - Git Subprocess Wrapper

Goal: isolate all system Git use.

Steps:

1. Implement `git` module using `std::process::Command`.
2. Add `git --version` preflight used only by repo-inspecting or repo-mutating
   commands.
3. Add helpers for:
   - `git init`
   - repo existence checks
   - commit existence checks
   - dirty-state check using stable porcelain output
   - add/commit
   - tag creation for migration backup
4. Ensure help/version-style CLI paths do not preflight Git.
5. Add tests with temp repos and a PATH override for missing Git.

Validation:

- `cargo check --all-targets --all-features`
- `cargo nextest run --all-features`

Web validation:

- `std::process::Command` docs require command args to be passed with `arg` or
  `args`, not embedded in the program string.
- Git docs define `git init` behavior and `git status --porcelain` as stable
  for scripts.

## Phase 6 - Strict TOML Schemas

Goal: parse and write repo/local schemas without applying files.

Steps:

1. Implement root `lnk.toml` parsing with required `format = 1`, optional
   `name`, and unknown-field errors.
2. Implement optional `profiles/<name>/profile.toml` parsing.
3. Implement local state parsing/writing for `format`, `active_profiles`, and
   deterministic sorted `targets`.
4. Add schema fixtures for valid, missing, malformed, unsupported format, and
   unknown-field cases.
5. Keep manifests optional; do not require empty `profile.toml` files.

Validation:

- `cargo check --all-targets --all-features`
- `cargo nextest run --all-features`

Web validation:

- TOML crate supports serde-compatible parsing/serialization.
- Serde derive supports mapping typed Rust structures to serialized data.

## Phase 7 - Profile Discovery And Ownership Planning

Goal: compute effective profile file ownership with no mutations.

Steps:

1. Discover profile directories under `profiles/`.
2. Auto-include regular files under `profiles/<profile>/files/`.
3. Apply manifest exceptions for target remapping, copy mode, and explicit
   overrides.
4. Validate selected ordered profiles.
5. Detect duplicate target claims.
6. Accept duplicates only when the later claim names the earlier profile in
   `overrides`.
7. Validate script metadata and platform names.
8. Add unit tests for profile ordering, override validation, target mapping, and
   duplicate claims.

Validation:

- `cargo check --all-targets --all-features`
- `cargo nextest run --all-features`

Web validation:

- This phase depends on the strict TOML/Serde source validation from Phase 6 and
  the path-boundary validation from Phase 4.

## Phase 8 - `lnk init`

Goal: create a new v2 setup repo exactly as designed.

Steps:

1. Create the resolved repo directory if missing.
2. Write `lnk.toml` with `format = 1` and default `name`.
3. Create `profiles/base/files/` and `profiles/base/scripts/`.
4. If not a Git repo, run `git init`, stage the initial files, and commit:
   `lnk: initialize v2 setup repo`.
5. If an existing Git repo has commits and no `lnk.toml`, refuse with a direct
   message.
6. Do not run scripts.
7. Add black-box tests for empty path init, existing v2 init, existing
   unrelated Git repo refusal, and missing Git.

Validation:

- `cargo check --all-targets --all-features`
- `cargo nextest run --all-features`
- inspect temp repo layout and Git log in tests

Web validation:

- Git docs define safe `git init` reinitialization behavior and initial branch
  defaults. The implementation should not rely on a hardcoded branch name.

## Phase 9 - `lnk profiles`

Goal: inspect repo profiles and local active profiles without mutation.

Steps:

1. Load root `lnk.toml`, discovered profiles, optional manifests, and local
   state if it exists.
2. Print terse human output.
3. Add `--json` output with profile names and active profile list.
4. Do not explain script gating in normal output.
5. Add tests for no local state, with active profiles, invalid manifest, and
   JSON output.

Validation:

- `cargo check --all-targets --all-features`
- `cargo nextest run --all-features`

Web validation:

- The JSON behavior is internal product design, backed by Serde/serde_json
  typed serialization from the Phase 6 source set.

## Phase 10 - Apply Planning

Goal: compute the full `apply` plan without mutating live files.

Steps:

1. Resolve selected profiles:
   - with args: validate provided ordered list
   - without args: load active profiles from local state
2. Fail with `run 'lnk apply <profile...>' first` when no active profiles exist.
3. Build planned actions for symlink, copy, explicit override, backup, and
   conflict.
4. Classify unmanaged live targets.
5. Classify copy drift using stored source/live hashes.
6. Ensure `--dry-run` does not write local state or live files.
7. Add JSON output for dry-run plans.
8. Add unit tests for drift matrix and conflict classification.
9. Add black-box tests for dry-run behavior with temp `HOME`.

Validation:

- `cargo check --all-targets --all-features`
- `cargo nextest run --all-features`

Web validation:

- `std::fs` TOCTOU guidance means classification is not enough; Phase 11 must
  re-check conflicts at mutation time where practical and use atomic operations
  where available.

## Phase 11 - `lnk apply` Mutations

Goal: mutate files only after the plan has passed validation.

Steps:

1. Create parent directories as needed.
2. Create symlinks for symlink-mode files.
3. Copy files for copy-mode files.
4. For unmanaged conflicts, stop unless `--backup` is set.
5. Implement timestamped side-by-side backups:
   `<target>.lnk-backup-YYYYMMDD-HHMMSS`.
6. For managed symlinks, replace only when the current target is still managed
   by `lnk`.
7. For copy mode, apply the drift matrix from `DESIGN.md`.
8. Update local state only after successful non-dry-run apply.
9. Do not run scripts.
10. Add black-box tests for symlink apply, copy apply, backup, unmanaged
    conflict, active profile storage, failed apply preserving old active
    profiles, and copy drift stops.

Validation:

- `cargo check --all-targets --all-features`
- `cargo nextest run --all-features`
- manual smoke test with disposable `HOME`, `XDG_CONFIG_HOME`, and
  `XDG_STATE_HOME`

Web validation:

- `std::fs` docs require care around TOCTOU; mutation code should prefer atomic
  create/rename patterns where practical and revalidate before destructive
  operations.

## Phase 12 - `lnk status`

Goal: report repo and local machine state without mutation.

Steps:

1. Load active profiles and applied targets.
2. Report active profiles and applied targets.
3. Detect missing symlinks.
4. Detect stale or wrong symlinks.
5. Detect copy drift from recorded hashes.
6. Report repo dirty state using Git porcelain output.
7. Add `--json` output.
8. Add black-box tests for clean state, dirty repo, missing symlink, wrong
   symlink, and copy drift.

Validation:

- `cargo check --all-targets --all-features`
- `cargo nextest run --all-features`

Web validation:

- Git docs define porcelain status as stable for scripts.

## Phase 13 - `lnk doctor`

Goal: read-only health audit across repo, profiles, local state, and live files.

Steps:

1. Validate root schema.
2. Validate all profile manifests.
3. Validate home-bound target paths.
4. Validate duplicate targets and explicit overrides.
5. Validate override declarations refer to an earlier matching target.
6. Validate local state targets still have sources.
7. Detect copied-file drift.
8. Detect missing or wrong symlinks.
9. Add `--json` output.
10. Keep doctor read-only; no repair mode.
11. Add black-box tests for each reported class.

Validation:

- `cargo check --all-targets --all-features`
- `cargo nextest run --all-features`

Web validation:

- This phase combines earlier validated sources: XDG path rules, TOML/Serde
  schemas, Git porcelain status, and filesystem TOCTOU caution.

## Phase 14 - `lnk run`

Goal: execute explicit profile-owned scripts only.

Steps:

1. Parse `<script>` and `<profile>:<script>` forms.
2. For `<script>`, load active profiles and run matching scripts in order.
3. For `<profile>:<script>`, run exactly one script.
4. Enforce platform gating using Rust's current OS constant.
5. Stream stdout and stderr directly.
6. Return non-zero when any executed script fails.
7. Skip missing scripts in multi-profile mode.
8. Do not provide JSON output.
9. Add black-box tests for run order, profile-qualified run, missing script,
   platform skip, stdout/stderr streaming, and failure code.

Validation:

- `cargo check --all-targets --all-features`
- `cargo nextest run --all-features`

Web validation:

- `std::process::Command` docs support explicit argument and stdio handling;
  avoid shell string construction unless the script itself is the executable.

## Phase 15 - `lnk migrate v1`

Goal: provide explicit plan-first migration from v1 in the same Git history.

Steps:

1. Detect v1 inputs: `.lnk`, `.lnk.<host>`, host directories, and
   `bootstrap.sh`.
2. `lnk migrate v1` prints the migration plan only.
3. Refuse `--write` unless `--yes` is also provided.
4. Refuse dirty working trees before mutation.
5. Create backup tag `lnk-v1-backup-YYYYMMDD-HHMMSS`.
6. Transform the current checked-out branch directly.
7. Preserve remotes and history.
8. Map `.lnk` to `profiles/base/files/`.
9. Map `.lnk.<host>` to `profiles/<host>/files/`.
10. Convert host/common collisions into explicit host overrides.
11. Move `bootstrap.sh` to `profiles/base/scripts/setup`.
12. Remove old v1 tracking files after moving managed files.
13. Create one migration commit.
14. Write local active profiles: `base` plus current hostname profile when it
    exists.
15. Do not run scripts.
16. Add black-box tests for dry plan, dirty refusal, backup tag, same remote,
    explicit overrides, local state write, and migration commit.

Validation:

- `cargo check --all-targets --all-features`
- `cargo nextest run --all-features`
- inspect temp Git history, tags, remotes, and migration commit in tests

Web validation:

- Git tag/init/status docs validate using system Git for repo mutation, stable
  dirty checks, and backup tag creation.

## Phase 16 - End-To-End Product Flow

Goal: verify the low-cognitive-load workflow from a first-time user's view.

Steps:

1. Run a disposable full flow:
   - `lnk init`
   - create profile files
   - `lnk apply base`
   - `lnk status`
   - `lnk run setup`
   - `lnk apply`
2. Run JSON checks for `profiles`, `apply --dry-run`, `status`, and `doctor`.
3. Re-read `DESIGN.md`, `AGENTS.md`, and this plan for contradictions.
4. Remove dead abstractions and oversized modules introduced during
   implementation.
5. Update `DESIGN.md` only if implementation reveals a real design gap.

Validation:

- `cargo fmt --all --check`
- `cargo clippy --all-targets --all-features -- -D warnings`
- `cargo nextest run --all-features`
- `cargo test --doc --all-features`
- `cargo build --all-targets --all-features`
- `git diff`

Web validation:

- Final checks match `AGENTS.md`, nextest docs, Cargo docs, and the Rust 2024
  toolchain validation sources above.

## Rejected Planning Options

- Workspace-first layout: rejected because Cargo workspaces are for multiple
  related packages, and `lnk` currently needs one small CLI package with clear
  modules.
- Separate `plan` command: rejected because `DESIGN.md` already simplified this
  to `apply --dry-run`.
- Public repo flag: rejected because repo isolation belongs in env/test setup,
  not the product surface.
- CI pipeline phase: rejected for now because old CI was intentionally removed
  and no new pipeline was requested.
- Broad repair mode for `doctor`: rejected because it would blur read-only audit
  semantics and raise cognitive load.
