# Lnk

**Dotfiles, linked. No fluff.**

Lnk is a minimalist CLI tool for managing dotfiles using symlinks and Git. It moves files into a managed repository directory, replaces them with symlinks, and commits changes to Git. That's it—no templating, no secrets, no config file.

## ⚠️ Development Status

**This tool is under heavy development. Use at your own risk.**

While Lnk is functional and tested, it's still in active development. The API and behavior may change between versions. Please:

- **Backup your dotfiles** before using Lnk
- **Test in a safe environment** first
- **Review changes** before committing to important repositories
- **Report issues** if you encounter any problems

## Features

- **Simple**: Just three commands: `init`, `add`, and `rm`
- **Git-based**: Automatically commits changes with descriptive messages
- **Symlink management**: Creates relative symlinks for portability
- **XDG compliant**: Uses `$XDG_CONFIG_HOME/lnk` or `~/.config/lnk`
- **No configuration**: Works out of the box

## Installation

### From Source

```bash
git clone https://github.com/yarlson/lnk.git
cd lnk
go build -o lnk .
sudo mv lnk /usr/local/bin/
```

## Usage

### Initialize a repository

```bash
lnk init
```

This creates `$XDG_CONFIG_HOME/lnk` (or `~/.config/lnk`) and initializes a Git repository with `main` as the default branch.

**Safety Features:**
- **Idempotent**: Running `lnk init` multiple times is safe and won't break existing repositories
- **Repository Protection**: Won't overwrite existing non-lnk Git repositories (exits with error)
- **Fresh Repository Detection**: Automatically detects if a directory contains an existing repository

### Initialize with remote

```bash
lnk init --remote https://github.com/user/dotfiles.git
# or using short flag
lnk init -r git@github.com:user/dotfiles.git
```

This initializes the repository with `main` as the default branch and adds the specified URL as the `origin` remote, allowing you to sync your dotfiles with a Git hosting service.

**Remote Handling:**
- **Idempotent**: Adding the same remote URL multiple times is safe (no-op)
- **Conflict Detection**: Adding different remote URLs fails with clear error message
- **Existing Remote Support**: Works safely with repositories that already have remotes configured

### Add a file

```bash
lnk add ~/.bashrc
```

This:

1. Moves `~/.bashrc` to `$XDG_CONFIG_HOME/lnk/.bashrc`
2. Creates a symlink from `~/.bashrc` to the repository
3. Commits the change with message "lnk: added .bashrc"

### Remove a file

```bash
lnk rm ~/.bashrc
```

This:

1. Removes the symlink `~/.bashrc`
2. Moves the file back from the repository to `~/.bashrc`
3. Removes it from Git tracking and commits with message "lnk: removed .bashrc"

## Examples

```bash
# Initialize lnk
lnk init

# Initialize with remote for syncing with GitHub
lnk init --remote https://github.com/user/dotfiles.git

# Running init again is safe (idempotent)
lnk init  # No error, no changes

# Adding same remote again is safe
lnk init -r https://github.com/user/dotfiles.git  # No error, no changes

# Add some dotfiles
lnk add ~/.bashrc
lnk add ~/.vimrc
lnk add ~/.gitconfig

# Remove a file from management
lnk rm ~/.vimrc

# Your files are now managed in ~/.config/lnk with Git history
cd ~/.config/lnk
git log --oneline

# If you initialized with a remote, you can push changes
git push origin main
```

### Safety Examples

```bash
# Attempting to init over existing non-lnk repository
mkdir ~/.config/lnk && cd ~/.config/lnk
git init && echo "important" > file.txt && git add . && git commit -m "important data"
cd ~
lnk init  # ERROR: Won't overwrite existing repository

# Attempting to add conflicting remote
lnk init -r https://github.com/user/repo1.git
lnk init -r https://github.com/user/repo2.git  # ERROR: Different URL conflict
```

## Error Handling

- Adding a nonexistent file: exits with error
- Adding a directory: exits with "directories are not supported"
- Removing a non-symlink: exits with "file is not managed by lnk"
- **Repository conflicts**: `lnk init` protects existing non-lnk repositories from accidental overwrite
- **Remote conflicts**: Adding different remote URLs to existing remotes fails with descriptive error
- Git operations show stderr output on failure

## Development

### Running Tests

```bash
go test -v ./test
```

The project uses integration tests that test real file and Git operations in isolated temporary directories.

### Project Structure

```
├── cmd/                 # Cobra CLI commands
│   ├── root.go         # Root command
│   ├── init.go         # Init command
│   ├── add.go          # Add command
│   └── rm.go           # Remove command
├── internal/
│   ├── core/           # Core business logic
│   │   └── lnk.go
│   ├── fs/             # File system operations
│   │   └── filesystem.go
│   └── git/            # Git operations
│       └── git.go
├── test/               # Integration tests
│   └── integration_test.go
├── main.go             # Entry point
├── README.md           # Documentation
└── go.mod             # Dependencies
```

## License

MIT License - see LICENSE file for details.
