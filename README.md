# Lnk

**The missing middle: Safer than simple, simpler than complex.**

Git-native dotfiles management that won't break your setup. Zero config, zero bloat, zero surprises.

```bash
# The power of Git, the safety of proper engineering
lnk init && lnk add ~/.vimrc && git push
```

[![Tests](https://img.shields.io/badge/tests-12%20passing-green)](./test) [![Go](https://img.shields.io/badge/go-1.21+-blue)](https://golang.org) [![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

## Why Lnk?

**The dotfiles manager that fills the missing gap.**

While chezmoi offers 100+ features and Home Manager requires learning Nix, **Lnk focuses on doing the essentials perfectly**:

- 🎯 **Safe simplicity**: More robust than Dotbot, simpler than chezmoi
- 🛡️ **Bulletproof operations**: Comprehensive edge case handling (unlike minimal tools)
- ⚡ **Zero friction**: No YAML configs, no templates, no learning curve
- 🔧 **Git-native**: Clean commits, standard workflow, no abstractions
- 📦 **Zero dependencies**: Single binary vs Python/Node/Ruby runtimes
- 🚀 **Production ready**: 12 integration tests, proper error handling

**The market gap**: Tools are either too simple (and unsafe) or too complex (and overwhelming). Lnk is the **Goldilocks solution** – just right for developers who want reliability without complexity.

## Quick Start

```bash
# Install (30 seconds)
curl -sSL https://github.com/yarlson/lnk/releases/latest/download/lnk-linux-amd64 -o lnk
chmod +x lnk && sudo mv lnk /usr/local/bin/

# Use (60 seconds)
lnk init -r git@github.com:you/dotfiles.git
lnk add ~/.bashrc ~/.vimrc ~/.gitconfig
cd ~/.config/lnk && git push -u origin main
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
lnk init -r git@github.com:you/dotfiles.git
cd ~/.config/lnk && git pull  # Get your existing dotfiles
# lnk automatically detects existing symlinks

# Or clone existing manually for complex setups
git clone git@github.com:you/dotfiles.git ~/.config/lnk
cd ~/.config/lnk && find . -name ".*" -exec ln -sf ~/.config/lnk/{} ~/{} \;
```

## Examples

<details>
<summary><strong>📁 Common Development Setup</strong></summary>

```bash
# Initialize with remote (recommended)
lnk init -r git@github.com:you/dotfiles.git

# Shell & terminal
lnk add ~/.bashrc ~/.zshrc ~/.tmux.conf

# Development tools  
lnk add ~/.vimrc ~/.gitconfig ~/.ssh/config

# Language-specific
lnk add ~/.npmrc ~/.cargo/config.toml ~/.pylintrc

# Push to remote
cd ~/.config/lnk && git push -u origin main

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

### Feature Positioning

| Feature | Lnk | Dotbot | yadm | chezmoi | Home Manager |
|---------|-----|--------|------|---------|--------------|
| **Simplicity** | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Safety/Edge Cases** | ✅ | ❌ | ⚠️ | ✅ | ✅ |
| **Git Integration** | ✅ | ❌ | ✅ | ⚠️ | ❌ |
| **Zero Dependencies** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Cross-Platform** | ✅ | ✅ | ⚠️ | ✅ | ⚠️ |
| **Learning Curve** | Minutes | Minutes | Hours | Days | Weeks |
| **File Templating** | ❌ | ❌ | Basic | Advanced | Advanced |
| **Built-in Encryption** | ❌ | ❌ | ✅ | ✅ | Plugin |
| **Package Management** | ❌ | ❌ | ❌ | ❌ | ✅ |

**Lnk's niche**: Maximum safety and Git integration with minimum complexity.

### Performance

- **Single binary**: ~8MB, starts in <10ms
- **Minimal I/O**: Only touches files being managed
- **Git efficiency**: Uses native Git commands, not libraries

## FAQ

<details>
<summary><strong>How is this different from other dotfiles managers?</strong></summary>

| Tool | Stars | Approach | Complexity | Learning Curve | Git Integration | Cross-Platform | Key Strength |
|------|-------|----------|------------|----------------|-----------------|----------------|--------------|
| **Lnk** | - | Simple symlinks + safety | **Minimal** | **Minutes** | **Native** | ✅ | **Safe simplicity** |
| chezmoi | 15k | Templates + encryption | High | Hours/Days | Abstracted | ✅ | Feature completeness |
| Mackup | 14.9k | App config sync | Medium | Hours | Manual | macOS/Linux | GUI app settings |
| Home Manager | 8.1k | Declarative Nix | **Very High** | **Weeks** | Manual | Linux/macOS | Package + config unity |
| Dotbot | 7.4k | YAML symlinks | Low | Minutes | Manual | ✅ | Pure simplicity |
| yadm | 5.7k | Git wrapper | Medium | Hours | **Native** | Unix-like | Git-centric power |

**Lnk fills the "safe simplicity" gap** – easier than chezmoi/yadm, safer than Dotbot, more capable than plain Git.

</details>

<details>
<summary><strong>Why choose Lnk over the alternatives?</strong></summary>

**Choose Lnk if you want:**
- ✅ **Safety first**: Bulletproof edge case handling, won't break existing setups
- ✅ **Git-native workflow**: No abstractions, just clean commits with clear messages  
- ✅ **Zero learning curve**: 3 commands, works like Git, no configuration files
- ✅ **Zero dependencies**: Single binary, no Python/Node/Ruby runtime requirements
- ✅ **Production ready**: Comprehensive test suite, proper error handling

**Choose others if you need:**
- **chezmoi**: Heavy templating, password manager integration, Windows-first
- **Mackup**: GUI app settings sync via Dropbox/iCloud (macOS focus)
- **Home Manager**: Nix ecosystem, package management, declarative everything
- **Dotbot**: Ultra-minimal YAML configuration (no safety features)
- **yadm**: Git power user features, encryption, bare repo workflow

**The sweet spot**: Lnk is for developers who want dotfiles management **without the ceremony** – all the safety and Git integration you need, none of the complexity you don't.

</details>

<details>
<summary><strong>When NOT to use Lnk?</strong></summary>

**Lnk might not be for you if you need:**

❌ **File templating**: Different configs per machine → use **chezmoi**  
❌ **Built-in encryption**: Secrets in dotfiles → use **chezmoi** or **yadm**  
❌ **GUI app settings**: Mac app preferences → use **Mackup**  
❌ **Package management**: Installing software → use **Home Manager** (Nix)  
❌ **Complex workflows**: Multi-step bootstrapping → use **chezmoi** or custom scripts  
❌ **Windows-first**: Native Windows support → use **chezmoi**  

**Lnk's philosophy**: Do one thing (symlink management) extremely well, let other tools handle their specialties. You can always combine Lnk with other tools as needed.

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
