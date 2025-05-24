# Lnk

**The dotfiles manager that gets out of your way.**

Symlink your dotfiles, commit with Git, sync anywhere. Zero config, zero bloat, zero surprises.

```bash
# One command to rule them all
lnk init && lnk add ~/.vimrc && git push
```

[![Tests](https://img.shields.io/badge/tests-12%20passing-green)](./test) [![Go](https://img.shields.io/badge/go-1.21+-blue)](https://golang.org) [![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

## Why Lnk?

**For engineers who want dotfiles management without the ceremony.**

- ✅ **Actually simple**: 3 commands total (`init`, `add`, `rm`)
- ✅ **Git-native**: No abstractions, just commits with clear messages
- ✅ **Bulletproof**: Comprehensive edge case handling, won't destroy your setup
- ✅ **Portable**: Relative symlinks work across machines
- ✅ **Standards-compliant**: Respects XDG Base Directory spec
- ✅ **Zero dependencies**: Single binary, no runtime requirements

## Quick Start

```bash
# Install (30 seconds)
curl -sSL https://github.com/yarlson/lnk/releases/latest/download/lnk-linux-amd64 -o lnk
chmod +x lnk && sudo mv lnk /usr/local/bin/

# Use (60 seconds)
lnk init
lnk add ~/.bashrc ~/.vimrc ~/.gitconfig
cd ~/.config/lnk && git remote add origin git@github.com:you/dotfiles.git && git push -u origin main
```

**That's it.** Your dotfiles are now version-controlled and synced.

## Installation

### Quick Install (Recommended)

```bash
# Linux/macOS
curl -sSL https://raw.githubusercontent.com/yarlson/lnk/main/install.sh | bash

# Or manually download from releases
wget https://github.com/yarlson/lnk/releases/latest/download/lnk-$(uname -s | tr '[:upper:]' '[:lower:]')-amd64
```

### From Source

```bash
git clone https://github.com/yarlson/lnk.git && cd lnk
go build -ldflags="-s -w" -o lnk .
sudo mv lnk /usr/local/bin/
```

### Package Managers

```bash
# Homebrew (macOS/Linux)
brew install yarlson/tap/lnk

# Arch Linux
yay -S lnk-git
```

## How It Works

**The mental model is simple**: Lnk moves your dotfiles to `~/.config/lnk/` and replaces them with symlinks.

```
Before:  ~/.vimrc (actual file)
After:   ~/.vimrc -> ~/.config/lnk/.vimrc (symlink)
```

Every change gets a Git commit with descriptive messages like `lnk: added .vimrc`.

## Usage

### Initialize Once

```bash
lnk init                                          # Local repository
lnk init -r git@github.com:username/dotfiles.git # With remote
```

**Safety features** (because your dotfiles matter):
- ✅ Idempotent - run multiple times safely
- ✅ Protects existing repositories from overwrite  
- ✅ Validates remote conflicts before changes

### Manage Files

```bash
lnk add ~/.bashrc ~/.vimrc ~/.tmux.conf    # Add multiple files
lnk rm ~/.bashrc                           # Remove from management
```

### Real-World Workflow

```bash
# Set up on new machine
git clone git@github.com:you/dotfiles.git ~/.config/lnk
cd ~/.config/lnk && find . -name ".*" -exec ln -sf ~/.config/lnk/{} ~/{} \;

# Or use lnk for safety
lnk init -r git@github.com:you/dotfiles.git
git pull  # Get your existing dotfiles
# lnk automatically detects existing symlinks
```

## Examples

<details>
<summary><strong>📁 Common Development Setup</strong></summary>

```bash
# Shell & terminal
lnk add ~/.bashrc ~/.zshrc ~/.tmux.conf

# Development tools  
lnk add ~/.vimrc ~/.gitconfig ~/.ssh/config

# Language-specific
lnk add ~/.npmrc ~/.cargo/config.toml ~/.pylintrc

# Check what's managed
cd ~/.config/lnk && git log --oneline
# 7f3a12c lnk: added .pylintrc
# 4e8b33d lnk: added .cargo/config.toml  
# 2a9c45e lnk: added .npmrc
```
</details>

<details>
<summary><strong>🔄 Multi-Machine Sync</strong></summary>

```bash
# Machine 1: Initial setup
lnk init -r git@github.com:you/dotfiles.git
lnk add ~/.vimrc ~/.bashrc
cd ~/.config/lnk && git push

# Machine 2: Clone existing
lnk init -r git@github.com:you/dotfiles.git  
cd ~/.config/lnk && git pull
# Manually symlink existing files or use lnk add to adopt them

# Both machines: Keep in sync
cd ~/.config/lnk && git pull  # Get updates
cd ~/.config/lnk && git push  # Share updates
```
</details>

<details>
<summary><strong>⚠️ Error Handling</strong></summary>

```bash
# Lnk is defensive by design
lnk add /nonexistent/file
# ❌ Error: file does not exist

lnk add ~/Documents/
# ❌ Error: directories are not supported  

lnk rm ~/.bashrc  # (when it's not a symlink)
# ❌ Error: file is not managed by lnk

lnk init  # (when ~/.config/lnk has non-lnk git repo)
# ❌ Error: directory appears to contain existing Git repository
```
</details>

## Technical Details

### Architecture

```
cmd/           # CLI layer (Cobra)
├── init.go    # Repository initialization  
├── add.go     # File adoption & symlinking
└── rm.go      # File restoration

internal/
├── core/      # Business logic
├── fs/        # File system operations  
└── git/       # Git automation
```

### What Makes It Robust

- **12 integration tests** covering edge cases and error conditions
- **Zero external dependencies** at runtime
- **Atomic operations** with automatic rollback on failure
- **Relative symlinks** for cross-platform compatibility
- **XDG compliance** with fallback to `~/.config`

### Performance

- **Single binary**: ~8MB, starts in <10ms
- **Minimal I/O**: Only touches files being managed
- **Git efficiency**: Uses native Git commands, not libraries

## FAQ

<details>
<summary><strong>How is this different from GNU Stow/Chezmoi/Dotbot?</strong></summary>

| Tool | Approach | Complexity | Git Integration |
|------|----------|------------|-----------------|
| **Lnk** | Simple symlinks | Minimal | Native |
| Stow | Directory trees | Medium | Manual |
| Chezmoi | Templates + state | High | Abstracted |
| Dotbot | YAML config | Medium | Manual |

**Lnk is for developers who want Git-native dotfiles without configuration overhead.**
</details>

<details>
<summary><strong>What if I already have a dotfiles repo?</strong></summary>

```bash
# Clone your existing repo to the lnk location
git clone your-repo ~/.config/lnk

# Lnk works with any Git repo structure
lnk add ~/.vimrc  # Adopts existing files safely
```
</details>

<details>
<summary><strong>Is this production ready?</strong></summary>

**Yes, with caveats.** Lnk is thoroughly tested and handles edge cases well, but it's actively developed. 

✅ **Safe to use**: Won't corrupt your files  
✅ **Well tested**: Comprehensive integration test suite  
⚠️ **API stability**: Commands may evolve (following semver)

**Recommendation**: Try it on non-critical dotfiles first.
</details>

## Development

### Quick Dev Setup

```bash
git clone https://github.com/yarlson/lnk.git && cd lnk
make test      # Run integration tests
make build     # Build binary  
make dev       # Watch & rebuild
```

### Contributing

We follow standard Go practices:
- **Tests first**: All features need integration tests
- **Conventional commits**: `feat:`, `fix:`, `docs:`, etc.
- **No dependencies**: Keep the runtime dependency-free

## License

MIT License - use it however you want.

---

**Made by developers, for developers.** Star ⭐ if this saves you time.
