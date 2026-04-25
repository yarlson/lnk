# Storage Layout and Tracking Model

## On-disk shape

The repo path (`LNK_HOME` / `XDG_CONFIG_HOME/lnk` / `~/.config/lnk`) is a normal Git working tree. Inside it:

```
<repo>/
├── .git/                    # standard git directory
├── .lnk                     # index of common managed items
├── .lnk.work                # index of host "work" managed items (one per host)
├── <home-relative paths>    # storage for common managed items (e.g. .vimrc, .config/nvim/init.lua)
├── work.lnk/                # storage root for host "work"
│   └── <home-relative paths>
├── laptop.lnk/
│   └── ...
└── bootstrap.sh             # optional, see flows/bootstrap.md
```

The repo path itself doubles as the storage root for **common** items. There is no `common.lnk/` directory; common files live at the repo root alongside the index files.

## Index file format (`.lnk` / `.lnk.<host>`)

- Plain text, UTF-8, one path per line, newline-terminated.
- Each entry is a path relative to the user's home directory (e.g. `.vimrc`, `.config/nvim/init.lua`). Paths outside `$HOME` are stored with the leading `/` stripped.
- The list is sorted on every write; duplicates are deduplicated on add. Empty lines are tolerated on read but not produced.
- An empty index file (after removing the last entry) is written as zero bytes (no trailing newline).
- The same relative path can appear in `.lnk` and in any number of `.lnk.<host>` files independently — common and host scopes are not merged.

## Where a managed item is stored

For a managed item with relative path `R`:

- Common scope: `<repo>/R`
- Host scope `H`: `<repo>/H.lnk/R`

The corresponding symlink in the user's environment is always `~/R`, regardless of scope. Switching the active host means switching which file `~/R` points to — only one of common or host can own a given path on a given machine at a time, since both target the same symlink location.

## Git path of a managed item

Git stages the storage path, not the home-relative path:

- Common: `git add R`
- Host `H`: `git add H.lnk/R`

The index file is staged as `.lnk` or `.lnk.<host>`.

## Hostname discovery

- `lnk.GetCurrentHostname()` returns `os.Hostname()`. The CLI does not call this implicitly — `--host` is always opaque user input. Users typically run `lnk pull --host $(hostname)` after `lnk pull` on a fresh machine.
- `cmd.findHostConfigs` enumerates hosts by listing `.lnk.*` files at the repo root (used by `lnk list --all` and `lnk init -r` for host-specific next-step hints).

## Repo-detection rules

`git.IsLnkRepository` decides whether an existing `.git` directory is safe to adopt during `lnk init`:

- No commits → treated as lnk-compatible (fresh repo).
- All commit subjects start with `lnk:` → treated as an lnk repo.
- Any commit subject without the `lnk:` prefix → not an lnk repo; `lnk init` errors with `ErrGitRepoExists`.

`initializer.HasUserContent` uses a different rule: it checks whether the directory contains `.lnk` or any `.lnk.*` file. This is the gate for `lnk init -r <url>` against a non-empty repo path; `--force` overrides it.
