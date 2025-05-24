# Lnk

**Dotfiles, linked. No fluff.**

Lnk is a minimalist CLI tool for managing dotfiles using symlinks and Git. It moves files into a managed repository directory, replaces them with symlinks, and commits changes to Git. That's it—no templating, no secrets, no config file.

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

This creates `$XDG_CONFIG_HOME/lnk` (or `~/.config/lnk`) and initializes a Git repository.

### Initialize with remote

```bash
lnk init --remote https://github.com/user/dotfiles.git
# or using short flag
lnk init -r git@github.com:user/dotfiles.git
```

This initializes the repository and adds the specified URL as the `origin` remote, allowing you to sync your dotfiles with a Git hosting service.

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

## Error Handling

- Adding a nonexistent file: exits with error
- Adding a directory: exits with "directories are not supported"
- Removing a non-symlink: exits with "file is not managed by lnk"
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
