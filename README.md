# Lnk

**Git-native dotfiles management that doesn't suck.**

Move your dotfiles to `~/.config/lnk`, symlink them back, and use Git like normal. Supports both common configurations and host-specific setups. Automatically runs bootstrap scripts to set up your environment.

```bash
lnk init -r git@github.com:user/dotfiles.git  # Clones & runs bootstrap automatically
lnk add ~/.vimrc ~/.bashrc                    # Common config
lnk add --host work ~/.ssh/config             # Host-specific config
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

# With existing repo (runs bootstrap automatically)
lnk init -r git@github.com:user/dotfiles.git

# Skip automatic bootstrap
lnk init -r git@github.com:user/dotfiles.git --no-bootstrap

# Run bootstrap script manually
lnk bootstrap
```

### Daily workflow

```bash
# Add files/directories (common config)
lnk add ~/.vimrc ~/.config/nvim ~/.gitconfig

# Add host-specific files
lnk add --host laptop ~/.ssh/config
lnk add --host work ~/.gitconfig

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
# Common config (shared everywhere)
lnk add ~/.vimrc ~/.bashrc ~/.gitconfig

# Host-specific config (unique per machine)
lnk add --host $(hostname) ~/.ssh/config
lnk add --host work ~/.gitconfig

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
# Clone dotfiles and run bootstrap automatically
lnk init -r git@github.com:you/dotfiles.git
# → Downloads dependencies, installs packages, configures environment

# Add common config (shared across all machines)
lnk add ~/.bashrc ~/.vimrc ~/.gitconfig

# Add host-specific config
lnk add --host $(hostname) ~/.ssh/config ~/.tmux.conf

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
lnk add --host work ~/.gitconfig
lnk push "work git config"

# Back on laptop
lnk pull                        # Get updates (work config won't affect laptop)
```

## Commands

- `lnk init [-r remote] [--no-bootstrap]` - Create repo (runs bootstrap automatically)
- `lnk add [--host HOST] <files>` - Move files to repo, create symlinks
- `lnk rm [--host HOST] <files>` - Move files back, remove symlinks
- `lnk list [--host HOST] [--all]` - List files managed by lnk
- `lnk status` - Git status + sync info
- `lnk push [msg]` - Stage all, commit, push
- `lnk pull [--host HOST]` - Pull + restore missing symlinks
- `lnk bootstrap` - Run bootstrap script manually

### Command Options

- `--host HOST` - Manage files for specific host (default: common configuration)
- `--all` - Show all configurations (common + all hosts) when listing
- `-r, --remote URL` - Clone from remote URL when initializing
- `--no-bootstrap` - Skip automatic execution of bootstrap script after cloning

## Technical bits

- **Single binary** (~8MB, no deps)
- **Relative symlinks** (portable)
- **XDG compliant** (`~/.config/lnk`)
- **Multihost support** (common + host-specific configs)
- **Bootstrap support** (automatic environment setup)
- **Git-native** (standard Git repo, no special formats)

## Alternatives

| Tool    | Complexity | Why choose it                                           |
| ------- | ---------- | ------------------------------------------------------- |
| **lnk** | Minimal    | Just works, no config, Git-native, multihost, bootstrap |
| chezmoi | High       | Templates, encryption, cross-platform                   |
| yadm    | Medium     | Git power user, encryption                              |
| dotbot  | Low        | YAML config, basic features                             |
| stow    | Low        | Perl, symlink only                                      |

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
