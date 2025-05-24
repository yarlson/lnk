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

	// Initialize Git repository
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
