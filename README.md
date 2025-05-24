# Lnk

**Git-native dotfiles management that doesn't suck.**

Move your dotfiles to `~/.config/lnk`, symlink them back, and use Git like normal. That's it.

```bash
lnk init
lnk add ~/.vimrc ~/.bashrc
lnk push "setup"
```

## Install

```bash
# Quick install (recommended)
curl -sSL https://raw.githubusercontent.com/yarlson/lnk/main/install.sh | bash

# Homebrew (macOS/Linux)
brew tap yarlson/lnk
brew install lnk

# Manual download
wget https://github.com/yarlson/lnk/releases/latest/download/lnk-$(uname -s | tr '[:upper:]' '[:lower:]')-amd64
chmod +x lnk-* && sudo mv lnk-* /usr/local/bin/lnk

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
# Add files/directories
lnk add ~/.vimrc ~/.config/nvim ~/.gitconfig

# Check status
lnk status

# Sync changes
lnk push "updated vim config"
lnk pull
```

## How it works

```
Before: ~/.vimrc (file)
After:  ~/.vimrc -> ~/.config/lnk/.vimrc (symlink)
```

Your files live in `~/.config/lnk` (a Git repo). Lnk creates symlinks back to original locations. Edit files normally, use Git normally.

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
lnk add ~/.bashrc ~/.vimrc ~/.gitconfig
lnk push "initial setup"
```

### On a new machine

```bash
lnk init -r git@github.com:you/dotfiles.git
lnk pull  # auto-creates symlinks
```

### Daily edits

```bash
vim ~/.vimrc           # edit normally
lnk status             # check what changed
lnk push "new plugins" # commit & push
```

## Commands

- `lnk init [-r remote]` - Create repo
- `lnk add <files>` - Move files to repo, create symlinks
- `lnk rm <files>` - Move files back, remove symlinks
- `lnk status` - Git status + sync info
- `lnk push [msg]` - Stage all, commit, push
- `lnk pull` - Pull + restore missing symlinks

## Technical bits

- **Single binary** (~8MB, no deps)
- **Atomic operations** (rollback on failure)
- **Relative symlinks** (portable)
- **XDG compliant** (`~/.config/lnk`)
- **20 integration tests**

## Alternatives

| Tool    | Complexity | Why choose it                         |
| ------- | ---------- | ------------------------------------- |
| **lnk** | Minimal    | Just works, no config, Git-native     |
| chezmoi | High       | Templates, encryption, cross-platform |
| yadm    | Medium     | Git power user, encryption            |
| dotbot  | Low        | YAML config, basic features           |
| stow    | Low        | Perl, symlink only                    |

## FAQ

**Q: What if I already have dotfiles in Git?**  
A: `git clone your-repo ~/.config/lnk && lnk add ~/.vimrc` (adopts existing files)

**Q: How do I handle machine-specific configs?**  
A: Git branches, or just don't manage machine-specific files with lnk

**Q: Windows support?**  
A: Symlinks work on Windows 10+, but untested

**Q: Production ready?**  
A: I use it daily. It won't break your files. API might change (pre-1.0).

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
