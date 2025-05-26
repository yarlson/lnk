# Lnk

**Git-native dotfiles management that doesn't suck.**

Move your dotfiles to `~/.config/lnk`, symlink them back, and use Git like normal. Supports both common configurations and host-specific setups.

```bash
lnk init
lnk add ~/.vimrc ~/.bashrc              # Common config
lnk add --host work ~/.ssh/config       # Host-specific config
lnk push "setup"
```

## Install

```bash
# Quick install (recommended)
curl -sSL https://raw.githubusercontent.com/yarlson/lnk/main/install.sh | bash
```

```bash
# Homebrew (macOS/Linux)
brew tap yarlson/lnk
brew install lnk
```

```bash
# Manual download
wget https://github.com/yarlson/lnk/releases/latest/download/lnk-$(uname -s | tr '[:upper:]' '[:lower:]')-amd64
chmod +x lnk-* && sudo mv lnk-* /usr/local/bin/lnk
```

```bash
# From source
git clone https://github.com/yarlson/lnk.git && cd lnk && go build . && sudo mv lnk /usr/local/bin/
```

## Usage

### Setup

```bash
# Fresh start
lnk init

# With existing repo
lnk init -r git@github.com:user/dotfiles.git
```

### Daily workflow

```bash
# Add files/directories (common config)
lnk add ~/.vimrc ~/.config/nvim ~/.gitconfig

# Add host-specific files
lnk add --host laptop ~/.ssh/config
lnk add --host work ~/.aws/credentials

# List managed files
lnk list                    # Common config only
lnk list --host laptop      # Laptop-specific config
lnk list --all              # All configurations

# Check status
lnk status

# Sync changes
lnk push "updated vim config"
lnk pull                    # Pull common config
lnk pull --host laptop      # Pull laptop-specific config
```

## How it works

```
Common files:
Before: ~/.vimrc (file)
After:  ~/.vimrc -> ~/.config/lnk/.vimrc (symlink)

Host-specific files:
Before: ~/.ssh/config (file)
After:  ~/.ssh/config -> ~/.config/lnk/laptop.lnk/.ssh/config (symlink)
```

Your files live in `~/.config/lnk` (a Git repo). Common files go in the root, host-specific files go in `<host>.lnk/` subdirectories. Lnk creates symlinks back to original locations. Edit files normally, use Git normally.

## Multihost Support

Lnk supports both **common configurations** (shared across all machines) and **host-specific configurations** (unique per machine).

### File Organization

```
~/.config/lnk/
├── .lnk                    # Tracks common files
├── .lnk.laptop             # Tracks laptop-specific files
├── .lnk.work               # Tracks work-specific files
├── .vimrc                  # Common file
├── .gitconfig              # Common file
├── laptop.lnk/             # Laptop-specific storage
│   ├── .ssh/
│   │   └── config
│   └── .aws/
│       └── credentials
└── work.lnk/               # Work-specific storage
    ├── .ssh/
    │   └── config
    └── .company/
        └── config
```

### Usage Patterns

```bash
# Common config (shared everywhere)
lnk add ~/.vimrc ~/.bashrc ~/.gitconfig

# Host-specific config (unique per machine)
lnk add --host $(hostname) ~/.ssh/config
lnk add --host work ~/.aws/credentials

# List configurations
lnk list                    # Common only
lnk list --host work        # Work host only
lnk list --all              # Everything

# Pull configurations
lnk pull                    # Common config
lnk pull --host work        # Work-specific config
```

## Why not just Git?

You could `git init ~/.config/lnk` and manually symlink everything. Lnk just automates the tedious parts:

- Moving files safely
- Creating relative symlinks
- Handling conflicts
- Tracking what's managed

## Examples

### First time setup

```bash
lnk init -r git@github.com:you/dotfiles.git

# Add common config (shared across all machines)
lnk add ~/.bashrc ~/.vimrc ~/.gitconfig

# Add host-specific config
lnk add --host $(hostname) ~/.ssh/config ~/.aws/credentials

lnk push "initial setup"
```

### On a new machine

```bash
lnk init -r git@github.com:you/dotfiles.git

# Pull common config
lnk pull

# Pull host-specific config (if it exists)
lnk pull --host $(hostname)
```

### Daily edits

```bash
vim ~/.vimrc                    # edit normally
lnk list                        # see common config
lnk list --host $(hostname)     # see host-specific config
lnk list --all                  # see everything
lnk status                      # check what changed
lnk push "new plugins"          # commit & push
```

### Multi-machine workflow

```bash
# On your laptop
lnk add --host laptop ~/.ssh/config
lnk add ~/.vimrc                # Common config
lnk push "laptop ssh config"

# On your work machine
lnk pull                        # Get common config
lnk add --host work ~/.aws/credentials
lnk push "work aws config"

# Back on laptop
lnk pull                        # Get updates (work config won't affect laptop)
```

## Commands

- `lnk init [-r remote]` - Create repo
- `lnk add [--host HOST] <files>` - Move files to repo, create symlinks
- `lnk rm [--host HOST] <files>` - Move files back, remove symlinks
- `lnk list [--host HOST] [--all]` - List files managed by lnk
- `lnk status` - Git status + sync info
- `lnk push [msg]` - Stage all, commit, push
- `lnk pull [--host HOST]` - Pull + restore missing symlinks

### Command Options

- `--host HOST` - Manage files for specific host (default: common configuration)
- `--all` - Show all configurations (common + all hosts) when listing
- `-r, --remote URL` - Clone from remote URL when initializing

## Technical bits

- **Single binary** (~8MB, no deps)
- **Relative symlinks** (portable)
- **XDG compliant** (`~/.config/lnk`)
- **Multihost support** (common + host-specific configs)
- **Git-native** (standard Git repo, no special formats)

## Alternatives

| Tool    | Complexity | Why choose it                                |
| ------- | ---------- | -------------------------------------------- |
| **lnk** | Minimal    | Just works, no config, Git-native, multihost |
| chezmoi | High       | Templates, encryption, cross-platform        |
| yadm    | Medium     | Git power user, encryption                   |
| dotbot  | Low        | YAML config, basic features                  |
| stow    | Low        | Perl, symlink only                           |

## Contributing

```bash
git clone https://github.com/yarlson/lnk.git
cd lnk
make deps  # Install golangci-lint
make check # Runs fmt, vet, lint, test
```

**What we use:**

- **Runtime deps**: Only `cobra` (CLI framework)
- **Test deps**: `testify` for assertions
- **Build pipeline**: Standard Makefile with quality checks

**Before submitting:**

```bash
make check  # Runs all quality checks + tests
```

**Adding features:**

- Put integration tests in `test/integration_test.go`
- Use conventional commits: `feat:`, `fix:`, `docs:`

## License

[MIT](LICENSE)
