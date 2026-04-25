# Terminology

- **repo path** — the on-disk directory holding the Git working tree. Resolved as `LNK_HOME` if set, else `XDG_CONFIG_HOME/lnk`, else `~/.config/lnk`.
- **managed item** — a path (file or directory) that lnk has moved into the repo path and replaced with a symlink. Identified by its path relative to the user's home directory.
- **`.lnk` file** — plain-text, newline-separated index of managed items for the common (non-host) configuration. Lives at the root of the repo path.
- **`.lnk.<host>` file** — same format as `.lnk` but for a host-specific configuration. Each host has its own independent index.
- **host storage path** — directory inside the repo where host-specific managed items are stored. For host `H`, this is `<repo>/H.lnk/`. For the common configuration, it is the repo root itself.
- **common configuration** — managed items shared across all machines, indexed by `.lnk` and stored at the repo root.
- **host-specific configuration** — managed items scoped to a named host, indexed by `.lnk.<host>` and stored under `<host>.lnk/`. A host name is supplied with `--host`/`-H` and is opaque to lnk.
- **relative path** — the home-relative path used both as the index entry and as the path under host storage. For paths outside `$HOME`, the leading `/` is stripped instead of being made home-relative.
- **lnk repository** — a Git repository that either has no commits or whose commit subjects all begin with `lnk:`. This is how `lnk init` decides an existing Git directory is safe to adopt vs. error.
- **lnk-style commit** — a commit whose message starts with `lnk:` (e.g., `lnk: added .vimrc`, `lnk: removed .bashrc`, `lnk: cleaned 2 invalid entries`).
- **bootstrap script** — `bootstrap.sh` at the repo path. Runs automatically after `lnk init -r <url>` unless `--no-bootstrap`, and on demand via `lnk bootstrap`.
- **dirty** — the working tree has uncommitted changes (`git status --porcelain` is non-empty).
- **ahead / behind** — local commits not yet on the upstream tracking branch / upstream commits not yet local.
- **invalid entry** — a path listed in `.lnk`/`.lnk.<host>` that no longer corresponds to a stored file in the repo, or that escapes the storage path (`..` or absolute). Cleaned by `lnk doctor`.
- **broken symlink** — a managed item that exists in storage but whose `~/<relative path>` is not a symlink pointing at the stored file. Repaired by `lnk doctor` and by `lnk pull`.
- **`.lnk-backup` file** — file or directory renamed from `~/<relative path>` when `lnk pull` finds a regular file/directory where a symlink should exist. Preserves user data instead of overwriting.
- **RestoreInfo** — return type of `Pull()` and `RestoreSymlinks()`. Contains two lists: `Restored` (relative paths where symlinks were created) and `BackedUp` (relative paths where pre-existing files were renamed to `.lnk-backup`).
