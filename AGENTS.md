# AGENTS.md

`lnk` v2 is a Rust CLI for a Git-backed userland machine setup repo.

## Source Of Truth

- Start with `DESIGN.md`.
- Treat `DESIGN.md` as the product contract for v2.
- Treat deleted Go history as migration reference only.
- Do not preserve old Go/v1 behavior by default.
- Do not add architecture prose, handoff notes, or parallel design docs.
- If the design changes, update `DESIGN.md` in the same change.

## Rust Tooling

- Rust edition: 2024.
- Format: `cargo fmt --all --check`.
- Lint: `cargo clippy --all-targets --all-features -- -D warnings`.
- Test: `cargo nextest run --all-features`.
- Doctests: `cargo test --doc --all-features`.
- Build: `cargo build --all-targets --all-features`.
- If `cargo nextest` is unavailable, install it or report the blocker; do not silently replace it with weaker coverage.

## Required Checks

- Before editing Rust: `cargo check --all-targets --all-features`.
- Before finalizing Rust changes: run format, lint, nextest, doctests, and build.
- Before changing Rust tooling, editions, lints, release behavior, or dependencies: verify current official docs first; include 2026 in web searches.
- Prefer repo-local config over agent memory.

## Design Guardrails

- Keep one obvious workflow: `lnk apply <profiles...>`, then `lnk apply`, `lnk status`, `lnk run <script>`.
- Keep scripts explicit. Never run scripts from `init`, `apply`, or `migrate`.
- Keep targets inside `$HOME`.
- Keep normal files convention-based under profile `files/`.
- Keep `profile.toml` optional; use it only for copy mode, target remapping, overrides, or script metadata.
- Keep local machine state outside Git.
- Keep unmanaged live-file conflicts blocking by default; `--backup` is the only conflict bypass.
- Do not add `use`, `plan`, `--save`, `--overwrite`, public `--repo`, `LNK_REPO`, profile inheritance, package management, services, secrets, or sudo workflows.

## Rust Implementation Guardrails

- Keep CLI parsing separate from domain behavior.
- Keep modules small and concern-based: cli, app, repo, profile, manifest, paths, ownership, apply, state, status, doctor, scripts, git, errors, migrate.
- Use system `git` through `std::process::Command`; do not use a Git library.
- Preflight `git` only for commands that inspect or mutate repo state.
- Use typed errors internally; render terse human errors and stable JSON errors only at the CLI boundary.
- Keep filesystem-mutating behavior plan-first internally: compute actions, validate conflicts, then mutate.
- Do not introduce global state except controlled local machine state.
- Do not use `unsafe` unless the design explicitly requires it and the safety invariant is documented next to the code.

## Tests

- Test CLI behavior against temporary `HOME`, `XDG_CONFIG_HOME`, and `XDG_STATE_HOME`.
- Assert repo layout, local state TOML, symlink targets, copy drift, git state, stdout/stderr, and exit codes.
- Add focused unit tests for path normalization, `$HOME` boundary checks, profile ordering, overrides, conflict classification, state parsing, and copy hashes.
- Do not port Go tests mechanically.
- Do not add mock-only tests for filesystem or Git behavior when a tempdir integration test is practical.

## Avoid

- Long explanatory comments.
- Context summaries that duplicate `DESIGN.md`.
- Stale references to host scopes, `.lnk`, `.lnk.<host>`, or the old Go v2 command model except inside migration code/tests.
- Large service files.
- New abstractions before two concrete call sites need them.
- Dependencies for trivial parsing, path, or command glue.
