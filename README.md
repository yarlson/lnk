# Lnk

**Git-native dotfiles management that doesn't suck.**

Lnk makes managing your dotfiles straightforward, no tedious setups, no complex configurations. Just tell Lnk what files you want tracked, and it'll automatically move them into a tidy Git repository under `~/.config/lnk`. It then creates clean, portable symlinks back to their original locations. Easy.

Why bother with Lnk instead of plain old Git or other dotfile managers? Unlike traditional methods, Lnk automates the boring parts: safely relocating files, handling host-specific setups, bulk operations for multiple files, recursive directory processing, and even running your custom bootstrap scripts automatically. This means fewer manual steps and less chance of accidentally overwriting something important.

With Lnk, your dotfiles setup stays organized and effortlessly portable, letting you spend more time doing real work, not wrestling with configuration files.

```bash
lnk init -r git@github.com:user/dotfiles.git    # Clones & runs bootstrap automatically
lnk add ~/.vimrc ~/.bashrc ~/.gitconfig         # Multiple files at once
lnk add --recursive ~/.config/nvim              # Process directory contents
lnk add --dry-run ~/.tmux.conf                  # Preview changes first
lnk add --host work ~/.ssh/config               # Host-specific config
lnk doctor --dry-run                          # Check repo health
lnk push "setup"
```

## Install

```bash
# Quick install (recommended)
curl -sSL https://raw.githubusercontent.com/yarlson/lnk/main/install.sh | bash
```

```bash
# Homebrew (macOS/Linux)
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

# With existing repo (runs bootstrap automatically)
lnk init -r git@github.com:user/dotfiles.git

# Skip automatic bootstrap
lnk init -r git@github.com:user/dotfiles.git --no-bootstrap

# Force initialization (WARNING: overwrites existing managed files)
lnk init -r git@github.com:user/dotfiles.git --force

# Run bootstrap script manually
lnk bootstrap
```

### Daily workflow

```bash
# Add multiple files at once (common config)
lnk add ~/.vimrc ~/.bashrc ~/.gitconfig ~/.tmux.conf

# Add directory contents individually
lnk add --recursive ~/.config/nvim ~/.config/zsh

# Preview changes before applying
lnk add --dry-run ~/.config/git/config
lnk add --dry-run --recursive ~/.config/kitty

# Add host-specific files (supports bulk operations)
lnk add --host laptop ~/.ssh/config ~/.aws/credentials
lnk add --host work ~/.gitconfig ~/.ssh/config

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

## Safety Features

Lnk includes built-in safety checks to prevent accidental data loss:

### Data Loss Prevention

```bash
# This will be blocked if you already have managed files
lnk init -r git@github.com:user/dotfiles.git
# ❌ Directory ~/.config/lnk already contains managed files
#    💡 Use 'lnk pull' to update from remote instead of 'lnk init -r'

# Use pull instead to safely update
lnk pull

# Or force if you understand the risks (shows warning only when needed)
lnk init -r git@github.com:user/dotfiles.git --force
# ⚠️  Using --force flag: This will overwrite existing managed files
#    💡 Only use this if you understand the risks
```

### Smart Warnings

- **Contextual alerts**: Warnings only appear when there are actually managed files to overwrite
- **Clear guidance**: Error messages suggest the correct command to use
- **Force override**: Advanced users can bypass safety checks when needed

### Recovering from Accidental Deletion

If you accidentally delete a managed file without using `lnk rm`:

```bash
# File was deleted outside of lnk
rm ~/.bashrc  # Oops! Should have used 'lnk rm'

# lnk rm won't work because symlink is gone
lnk rm ~/.bashrc
# ❌ File or directory not found: ~/.bashrc

# Use --force to clean up the orphaned tracking entry
lnk rm --force ~/.bashrc
# ✅ Force removed .bashrc from lnk
```

## Bootstrap Support

Lnk automatically runs bootstrap scripts when cloning dotfiles repositories, making it easy to set up your development environment. Just add a `bootstrap.sh` file to your dotfiles repo.

### Examples

**Simple bootstrap script:**

```bash
#!/bin/bash
# bootstrap.sh
echo "Setting up development environment..."

# Install Homebrew (macOS)
if ! command -v brew &> /dev/null; then
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
fi

# Install packages
brew install git vim tmux

echo "✅ Setup complete!"
```

**Usage:**

```bash
# Automatic bootstrap on clone
lnk init -r git@github.com:you/dotfiles.git
# → Clones repo and runs bootstrap script automatically

# Skip bootstrap if needed
lnk init -r git@github.com:you/dotfiles.git --no-bootstrap

# Run bootstrap manually later
lnk bootstrap
```

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
│   └── .tmux.conf
└── work.lnk/               # Work-specific storage
    ├── .ssh/
    │   └── config
    └── .gitconfig
```

### Usage Patterns

```bash
# Common config (shared everywhere) - supports multiple files
lnk add ~/.vimrc ~/.bashrc ~/.gitconfig ~/.tmux.conf

# Process directory contents individually
lnk add --recursive ~/.config/nvim ~/.config/zsh

# Preview operations before making changes
lnk add --dry-run ~/.config/alacritty/alacritty.yml
lnk add --dry-run --recursive ~/.config/i3

# Host-specific config (unique per machine) - supports bulk operations
lnk add --host $(hostname) ~/.ssh/config ~/.aws/credentials
lnk add --host work ~/.gitconfig ~/.npmrc

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

- Moving files safely (with atomic operations)
- Creating relative symlinks
- Handling conflicts and rollback
- Tracking what's managed
- Processing multiple files efficiently
- Recursive directory traversal
- Preview mode for safety

## Examples

### First time setup

```bash
# Clone dotfiles and run bootstrap automatically
lnk init -r git@github.com:you/dotfiles.git
# → Downloads dependencies, installs packages, configures environment

# Add common config (shared across all machines) - multiple files at once
lnk add ~/.bashrc ~/.vimrc ~/.gitconfig ~/.tmux.conf

# Add configuration directories individually
lnk add --recursive ~/.config/nvim ~/.config/zsh

# Preview before adding sensitive files
lnk add --dry-run ~/.ssh/id_rsa.pub
lnk add ~/.ssh/id_rsa.pub  # Add after verification

# Add host-specific config (supports bulk operations)
lnk add --host $(hostname) ~/.ssh/config ~/.aws/credentials

lnk push "initial setup"
```

### On a new machine

```bash
# Bootstrap runs automatically
lnk init -r git@github.com:you/dotfiles.git
# → Sets up environment, installs dependencies

# Pull common config
lnk pull

# Pull host-specific config (if it exists)
lnk pull --host $(hostname)

# Or run bootstrap manually if needed
lnk bootstrap
```

### Daily edits

```bash
vim ~/.vimrc                    # edit normally
lnk list                        # see common config
lnk list --host $(hostname)     # see host-specific config
lnk list --all                  # see everything
lnk status                      # check what changed
lnk diff                        # see uncommitted changes
lnk doctor --dry-run            # check for health issues
lnk doctor                      # fix any issues found
lnk push "new plugins"          # commit & push
```

### Multi-machine workflow

```bash
# On your laptop - use bulk operations for efficiency
lnk add --host laptop ~/.ssh/config ~/.aws/credentials ~/.npmrc
lnk add ~/.vimrc ~/.bashrc ~/.gitconfig    # Common config (multiple files)
lnk push "laptop configuration"

# On your work machine
lnk pull                                   # Get common config
lnk add --host work ~/.gitconfig ~/.ssh/config
lnk add --recursive ~/.config/work-tools  # Work-specific tools
lnk push "work configuration"

# Back on laptop
lnk pull                                   # Get updates (work config won't affect laptop)
```

## Commands

- `lnk init [-r remote] [--no-bootstrap] [--force]` - Create repo (runs bootstrap automatically)
- `lnk add [--host HOST] [--recursive] [--dry-run] <files>...` - Move files to repo, create symlinks
- `lnk rm [--host HOST] [--force] <files>` - Move files back, remove symlinks
- `lnk list [--host HOST] [--all]` - List files managed by lnk
- `lnk status` - Git status + sync info
- `lnk diff` - Show uncommitted changes
- `lnk push [msg]` - Stage all, commit, push
- `lnk pull [--host HOST]` - Pull + restore missing symlinks
- `lnk doctor [--host HOST] [--dry-run]` - Diagnose and fix repository health issues
- `lnk bootstrap` - Run bootstrap script manually

### Command Options

- `--host HOST` - Manage files for specific host (default: common configuration)
- `--recursive, -r` - Add directory contents individually instead of the directory as a whole
- `--dry-run, -n` - Show what would be added without making changes
- `--all` - Show all configurations (common + all hosts) when listing
- `-r, --remote URL` - Clone from remote URL when initializing
- `--no-bootstrap` - Skip automatic execution of bootstrap script after cloning
- `--force` - Force initialization even if directory contains managed files (WARNING: overwrites existing content)
- `--force, -f` (rm) - Remove from tracking even if symlink is missing (useful if you accidentally deleted a managed file)

### Repository Health

```bash
# Preview issues without making changes
lnk doctor --dry-run

# Fix all detected issues
lnk doctor

# Check a specific host configuration
lnk doctor --host laptop
lnk doctor --host laptop --dry-run
```

The `doctor` command scans for three categories of issues:

- **Invalid entries**: files listed in `.lnk` but missing from repo storage
- **Broken symlinks**: managed files whose symlinks at `$HOME` are missing or point to the wrong location
- **Orphaned files**: files in repo storage not tracked in `.lnk`

### Output Formatting

Lnk provides flexible output formatting options to suit different environments and preferences:

#### Color Output

Control when ANSI colors are used in output:

```bash
# Default: auto-detect based on TTY
lnk init

# Force colors regardless of environment
lnk init --colors=always

# Disable colors completely
lnk init --colors=never

# Environment variable support
NO_COLOR=1 lnk init  # Disables colors (acts like --colors=never)
```

**Color modes:**

- `auto` (default): Use colors only when stdout is a TTY
- `always`: Force color output regardless of TTY
- `never`: Disable color output regardless of TTY

The `NO_COLOR` environment variable acts like `--colors=never` when set, but explicit `--colors` flags take precedence.

#### Emoji Output

Control emoji usage in output messages:

```bash
# Default: emojis enabled
lnk init

# Disable emojis
lnk init --no-emoji

# Explicitly enable emojis
lnk init --emoji
```

**Emoji flags:**

- `--emoji` (default: true): Enable emoji in output
- `--no-emoji`: Disable emoji in output

The `--emoji` and `--no-emoji` flags are mutually exclusive.

#### Examples

```bash
# Clean output for scripts/pipes
lnk init --colors=never --no-emoji

# Force colorful output in non-TTY environments
lnk init --colors=always

# Disable colors but keep emojis
lnk init --colors=never

# Disable emojis but keep colors
lnk init --no-emoji
```

### Add Command Examples

```bash
# Multiple files at once
lnk add ~/.bashrc ~/.vimrc ~/.gitconfig

# Recursive directory processing
lnk add --recursive ~/.config/nvim

# Preview changes first
lnk add --dry-run ~/.ssh/config
lnk add --dry-run --recursive ~/.config/kitty

# Host-specific bulk operations
lnk add --host work ~/.gitconfig ~/.ssh/config ~/.npmrc
```

## Technical bits

- **Single binary** (~8MB, no deps)
- **Relative symlinks** (portable)
- **XDG compliant** (`~/.config/lnk`)
- **Multihost support** (common + host-specific configs)
- **Bootstrap support** (automatic environment setup)
- **Bulk operations** (multiple files, atomic transactions)
- **Recursive processing** (directory contents individually)
- **Preview mode** (dry-run for safety)
- **Repository health checks** (diagnose and fix issues with `doctor`)
- **Data loss prevention** (safety checks with contextual warnings)
- **Git-native** (standard Git repo, no special formats)

## Alternatives

| Tool    | Complexity | Why choose it                                                                                     |
| ------- | ---------- | ------------------------------------------------------------------------------------------------- |
| **lnk** | Minimal    | Just works, no config, Git-native, multihost, bootstrap, bulk ops, dry-run, doctor, safety checks |
| chezmoi | High       | Templates, encryption, cross-platform                                                             |
| yadm    | Medium     | Git power user, encryption                                                                        |
| dotbot  | Low        | YAML config, basic features                                                                       |
| stow    | Low        | Perl, symlink only                                                                                |

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
