# lnk

**Git-native dotfiles manager. No config files, no templates, no ceremony.**

Track dotfiles across machines with one command. Lnk moves files into a Git repo (defaults to `~/.config/lnk`; override with `LNK_HOME` or `XDG_CONFIG_HOME`), symlinks them back, and stays out of your way.

```bash
lnk init -r git@github.com:you/dotfiles.git   # clone & bootstrap
lnk pull                                       # restore symlinks
lnk add ~/.vimrc ~/.bashrc ~/.gitconfig        # track files
lnk add --host work ~/.ssh/config              # per-machine config
lnk push "done"                                # commit & push
```

## Install

```bash
curl -sSL https://raw.githubusercontent.com/yarlson/lnk/main/install.sh | bash
```

Or with Homebrew:

```bash
brew install lnk
```

Or grab a binary from [releases](https://github.com/yarlson/lnk/releases), or build from source:

```bash
go install github.com/yarlson/lnk@latest
```

## How it works

```
Before: ~/.vimrc (regular file)
After:  ~/.vimrc → ~/.config/lnk/.vimrc (symlink into git repo)
```

Common files live at the repo root. Host-specific files go in `<hostname>.lnk/` subdirectories. A plain text `.lnk` file tracks what's managed — one path per line, no special format.

```
~/.config/lnk/
├── .lnk                 # tracked common files
├── .lnk.work            # tracked work-specific files
├── .vimrc               # common
├── .gitconfig            # common
└── work.lnk/            # host-specific storage
    └── .ssh/config
```

## Usage

### Add files

```bash
lnk add ~/.vimrc ~/.bashrc                # multiple at once
lnk add --recursive ~/.config/nvim        # each file individually
lnk add --host laptop ~/.ssh/config       # host-specific
lnk add --dry-run ~/.tmux.conf            # preview first
```

### Sync

```bash
lnk status                                # what changed (works even without remote)
lnk diff                                  # uncommitted changes
lnk diff --quiet                          # exit code only, no output
lnk diff --colors always                  # force color output (useful in scripts/redirects)
lnk push "updated vim config"             # commit & push
lnk pull                                  # pull & restore symlinks
lnk pull --host work                      # pull host-specific config
```

`status` works without a remote configured — it shows local state (dirty/clean, unpushed commits) and guides you to add a remote.

### Remove

```bash
lnk rm ~/.vimrc                           # moves file back, removes symlink
lnk rm --force ~/.bashrc                  # tracking cleanup only (no file restoration)
```

`--force` is for cleanup when the symlink is already gone (e.g., you deleted it manually). It removes the entry from `.lnk` and the stored file from the repo, but does **not** restore anything to your home directory. Use normal `lnk rm` for a full removal with restoration.

### List

```bash
lnk list                                  # common files
lnk list --host work                      # host-specific
lnk list --all                            # everything
```

### Health checks

```bash
lnk doctor --dry-run                      # preview issues
lnk doctor                                # fix broken symlinks & stale entries
```

When restoring symlinks, if a real file exists at the target location (not a symlink), it will be renamed to `<path>.lnk-backup` to preserve your data before the symlink is created. Check for `.lnk-backup` files after running `doctor` or `pull` if you expect them.

### Bootstrap

Drop a `bootstrap.sh` in your dotfiles repo. Lnk runs it automatically on `lnk init -r <url>`.

```bash
lnk init -r <url> --no-bootstrap          # skip auto-bootstrap
lnk bootstrap                             # run manually
```

## New machine setup

```bash
lnk init -r git@github.com:you/dotfiles.git
lnk pull
lnk pull --host $(hostname)
```

That's it. Bootstrap runs automatically, symlinks get restored, you're working.

## Commands

| Command                                            | What it does                                |
| -------------------------------------------------- | ------------------------------------------- |
| `init [-r url] [--force] [--no-bootstrap]`         | Create or clone a dotfiles repo             |
| `add [--host H] [--recursive] [--dry-run] <files>` | Track files (move to repo + symlink)        |
| `rm [--host H] [--force] <file>`                   | Untrack file (restore to original location) |
| `list [--host H] [--all]`                          | Show tracked files                          |
| `status`                                           | Git sync status                             |
| `diff`                                             | Uncommitted changes                         |
| `push [message]`                                   | Stage, commit, push                         |
| `pull [--host H]`                                  | Pull and restore symlinks                   |
| `doctor [--host H] [--dry-run]`                    | Find and fix repo health issues             |
| `bootstrap`                                        | Run bootstrap.sh from repo                  |

## Global Options

Available with all commands:

| Option                       | Default | What it does                                                  |
| ---------------------------- | ------- | ------------------------------------------------------------- |
| `--colors auto\|always\|never` | `auto`    | Control color output (auto: based on terminal detection)     |
| `--emoji`, `--no-emoji`        | enabled | Enable/disable emoji in output                               |
| `--quiet` or `-q`              | off     | Suppress all output (useful in scripts, exit code only)       |

## Why lnk over alternatives

|                | lnk                        | chezmoi          | yadm            | stow                     |
| -------------- | -------------------------- | ---------------- | --------------- | ------------------------ |
| Config needed  | None                       | Templates + YAML | Git knowledge   | Stow directory structure |
| Multi-host     | Built-in `--host` flag     | Templates        | Manual branches | Manual                   |
| Bulk ops       | `add` takes multiple files | One at a time    | One at a time   | Package-based            |
| Bootstrap      | Automatic on clone         | Separate scripts | Separate        | No                       |
| Health checks  | `doctor` command           | No               | No              | No                       |
| Learning curve | Minutes                    | Hours            | Medium          | Low                      |

## Contributing

```bash
git clone https://github.com/yarlson/lnk.git
cd lnk
make check    # fmt, vet, lint, test
```

## License

[MIT](LICENSE)
