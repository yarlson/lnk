package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yarlson/lnk/internal/fs"
	"github.com/yarlson/lnk/internal/git"
)

// Lnk represents the main application logic
type Lnk struct {
	repoPath string
	git      *git.Git
	fs       *fs.FileSystem
}

// NewLnk creates a new Lnk instance
func NewLnk() *Lnk {
	repoPath := getRepoPath()
	return &Lnk{
		repoPath: repoPath,
		git:      git.New(repoPath),
		fs:       fs.New(),
	}
}

// getRepoPath returns the path to the lnk repository directory
func getRepoPath() string {
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// Fallback to current directory if we can't get home
			xdgConfig = "."
		} else {
			xdgConfig = filepath.Join(homeDir, ".config")
		}
	}
	return filepath.Join(xdgConfig, "lnk")
}

// Init initializes the lnk repository
func (l *Lnk) Init() error {
	// Create the repository directory
	if err := os.MkdirAll(l.repoPath, 0755); err != nil {
		return fmt.Errorf("failed to create lnk directory: %w", err)
	}

	// Check if there's already a Git repository
	if l.git.IsGitRepository() {
		// Repository exists, check if it's a lnk repository
		if l.git.IsLnkRepository() {
			// It's a lnk repository, init is idempotent - do nothing
			return nil
		} else {
			// It's not a lnk repository, error to prevent data loss
			return fmt.Errorf("directory %s appears to contain an existing Git repository that is not managed by lnk. Please backup or move the existing repository before initializing lnk", l.repoPath)
		}
	}

	// No existing repository, initialize Git repository
	if err := l.git.Init(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	return nil
}

// AddRemote adds a remote to the repository
func (l *Lnk) AddRemote(name, url string) error {
	if err := l.git.AddRemote(name, url); err != nil {
		return fmt.Errorf("failed to add remote %s: %w", name, err)
	}
	return nil
}

// Add moves a file to the repository and creates a symlink
func (l *Lnk) Add(filePath string) error {
	// Validate the file
	if err := l.fs.ValidateFileForAdd(filePath); err != nil {
		return err
	}

	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Calculate destination path in repo
	basename := filepath.Base(absPath)
	destPath := filepath.Join(l.repoPath, basename)

	// Move file to repository
	if err := l.fs.MoveFile(absPath, destPath); err != nil {
		return fmt.Errorf("failed to move file to repository: %w", err)
	}

	// Create symlink
	if err := l.fs.CreateSymlink(destPath, absPath); err != nil {
		// Try to restore the file if symlink creation fails
		_ = l.fs.MoveFile(destPath, absPath) // Ignore error in cleanup
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	// Stage and commit the file
	if err := l.git.AddAndCommit(basename, fmt.Sprintf("lnk: added %s", basename)); err != nil {
		// Try to restore the original state if commit fails
		_ = os.Remove(absPath)               // Ignore error in cleanup
		_ = l.fs.MoveFile(destPath, absPath) // Ignore error in cleanup
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

// Remove removes a symlink and restores the original file
func (l *Lnk) Remove(filePath string) error {
	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Validate that this is a symlink managed by lnk
	if err := l.fs.ValidateSymlinkForRemove(absPath, l.repoPath); err != nil {
		return err
	}

	// Get the target path in the repository
	target, err := os.Readlink(absPath)
	if err != nil {
		return fmt.Errorf("failed to read symlink: %w", err)
	}

	// Convert relative path to absolute if needed
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(absPath), target)
	}

	basename := filepath.Base(target)

	// Remove the symlink
	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	// Move file back from repository
	if err := l.fs.MoveFile(target, absPath); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	// Remove from Git and commit
	if err := l.git.RemoveAndCommit(basename, fmt.Sprintf("lnk: removed %s", basename)); err != nil {
		// Try to restore the symlink if commit fails
		_ = l.fs.MoveFile(absPath, target)      // Ignore error in cleanup
		_ = l.fs.CreateSymlink(target, absPath) // Ignore error in cleanup
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

// GetCommits returns the list of commits for testing purposes
func (l *Lnk) GetCommits() ([]string, error) {
	return l.git.GetCommits()
}

// StatusInfo contains repository sync status information
type StatusInfo struct {
	Ahead  int
	Behind int
	Remote string
}

// Status returns the repository sync status
func (l *Lnk) Status() (*StatusInfo, error) {
	// Check if repository is initialized
	if !l.git.IsGitRepository() {
		return nil, fmt.Errorf("lnk repository not initialized - run 'lnk init' first")
	}

	gitStatus, err := l.git.GetStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository status: %w", err)
	}

	return &StatusInfo{
		Ahead:  gitStatus.Ahead,
		Behind: gitStatus.Behind,
		Remote: gitStatus.Remote,
	}, nil
}

// Push stages all changes and creates a sync commit, then pushes to remote
func (l *Lnk) Push(message string) error {
	// Check if repository is initialized
	if !l.git.IsGitRepository() {
		return fmt.Errorf("lnk repository not initialized - run 'lnk init' first")
	}

	// Check if there are any changes
	hasChanges, err := l.git.HasChanges()
	if err != nil {
		return fmt.Errorf("failed to check for changes: %w", err)
	}

	if hasChanges {
		// Stage all changes
		if err := l.git.AddAll(); err != nil {
			return fmt.Errorf("failed to stage changes: %w", err)
		}

		// Create a sync commit
		if err := l.git.Commit(message); err != nil {
			return fmt.Errorf("failed to commit changes: %w", err)
		}
	}

	// Push to remote (this will be a no-op in tests since we don't have real remotes)
	// In real usage, this would push to the actual remote repository
	if err := l.git.Push(); err != nil {
		return fmt.Errorf("failed to push to remote: %w", err)
	}

	return nil
}

// Pull fetches changes from remote and restores symlinks as needed
func (l *Lnk) Pull() ([]string, error) {
	// Check if repository is initialized
	if !l.git.IsGitRepository() {
		return nil, fmt.Errorf("lnk repository not initialized - run 'lnk init' first")
	}

	// Pull changes from remote (this will be a no-op in tests since we don't have real remotes)
	if err := l.git.Pull(); err != nil {
		return nil, fmt.Errorf("failed to pull from remote: %w", err)
	}

	// Find all managed files in the repository and restore symlinks
	restored, err := l.RestoreSymlinks()
	if err != nil {
		return nil, fmt.Errorf("failed to restore symlinks: %w", err)
	}

	return restored, nil
}

// RestoreSymlinks finds all files in the repository and ensures they have proper symlinks
func (l *Lnk) RestoreSymlinks() ([]string, error) {
	var restored []string

	// Read all files in the repository
	entries, err := os.ReadDir(l.repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read repository directory: %w", err)
	}

	for _, entry := range entries {
		// Skip hidden files and directories (like .git)
		if entry.Name()[0] == '.' {
			continue
		}

		// Skip directories
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		repoFile := filepath.Join(l.repoPath, filename)

		// Determine where the symlink should be
		// For config files, we'll place them in the user's home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		symlinkPath := filepath.Join(homeDir, filename)

		// Check if symlink already exists and is correct
		if l.isValidSymlink(symlinkPath, repoFile) {
			continue
		}

		// Remove existing file/symlink if it exists
		if _, err := os.Lstat(symlinkPath); err == nil {
			if err := os.Remove(symlinkPath); err != nil {
				return nil, fmt.Errorf("failed to remove existing file %s: %w", symlinkPath, err)
			}
		}

		// Create symlink
		if err := l.fs.CreateSymlink(repoFile, symlinkPath); err != nil {
			return nil, fmt.Errorf("failed to create symlink for %s: %w", filename, err)
		}

		restored = append(restored, filename)
	}

	return restored, nil
}

// isValidSymlink checks if the given path is a symlink pointing to the expected target
func (l *Lnk) isValidSymlink(symlinkPath, expectedTarget string) bool {
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		return false
	}

	// Check if it's a symlink
	if info.Mode()&os.ModeSymlink == 0 {
		return false
	}

	// Check if it points to the correct target
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		return false
	}

	// Convert relative path to absolute if needed
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(symlinkPath), target)
	}

	// Clean both paths for comparison
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}

	expectedAbs, err := filepath.Abs(expectedTarget)
	if err != nil {
		return false
	}

	return targetAbs == expectedAbs
}
