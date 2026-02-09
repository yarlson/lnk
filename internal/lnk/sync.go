package lnk

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yarlson/lnk/internal/lnkerror"
)

// StatusInfo contains repository sync status information
type StatusInfo struct {
	Ahead  int
	Behind int
	Remote string
	Dirty  bool
}

// Status returns the repository sync status
func (l *Lnk) Status() (*StatusInfo, error) {
	// Check if repository is initialized
	if !l.git.IsGitRepository() {
		return nil, lnkerror.WithSuggestion(ErrNotInitialized, "run 'lnk init' first")
	}

	gitStatus, err := l.git.GetStatus()
	if err != nil {
		return nil, err
	}

	return &StatusInfo{
		Ahead:  gitStatus.Ahead,
		Behind: gitStatus.Behind,
		Remote: gitStatus.Remote,
		Dirty:  gitStatus.Dirty,
	}, nil
}

// Diff returns the diff output for uncommitted changes in the repository.
// If color is true, the output will include ANSI color codes.
func (l *Lnk) Diff(color bool) (string, error) {
	// Check if repository is initialized
	if !l.git.IsGitRepository() {
		return "", lnkerror.WithSuggestion(ErrNotInitialized, "run 'lnk init' first")
	}

	return l.git.Diff(color)
}

// Push stages all changes and creates a sync commit, then pushes to remote
func (l *Lnk) Push(message string) error {
	// Check if repository is initialized
	if !l.git.IsGitRepository() {
		return lnkerror.WithSuggestion(ErrNotInitialized, "run 'lnk init' first")
	}

	// Check if there are any changes
	hasChanges, err := l.git.HasChanges()
	if err != nil {
		return err
	}

	if hasChanges {
		// Stage all changes
		if err := l.git.AddAll(); err != nil {
			return err
		}

		// Create a sync commit
		if err := l.git.Commit(message); err != nil {
			return err
		}
	}

	// Push to remote (this will be a no-op in tests since we don't have real remotes)
	// In real usage, this would push to the actual remote repository
	return l.git.Push()
}

// Pull fetches changes from remote and restores symlinks as needed
func (l *Lnk) Pull() ([]string, error) {
	// Check if repository is initialized
	if !l.git.IsGitRepository() {
		return nil, lnkerror.WithSuggestion(ErrNotInitialized, "run 'lnk init' first")
	}

	// Pull changes from remote (this will be a no-op in tests since we don't have real remotes)
	if err := l.git.Pull(); err != nil {
		return nil, err
	}

	// Find all managed files in the repository and restore symlinks
	restored, err := l.RestoreSymlinks()
	if err != nil {
		return nil, fmt.Errorf("failed to restore symlinks: %w", err)
	}

	return restored, nil
}

// List returns the list of files and directories currently managed by lnk
func (l *Lnk) List() ([]string, error) {
	// Check if repository is initialized
	if !l.git.IsGitRepository() {
		return nil, lnkerror.WithSuggestion(ErrNotInitialized, "run 'lnk init' first")
	}

	// Get managed items from .lnk file
	managedItems, err := l.getManagedItems()
	if err != nil {
		return nil, fmt.Errorf("failed to get managed items: %w", err)
	}

	return managedItems, nil
}

// GetCommits returns the list of commits for testing purposes
func (l *Lnk) GetCommits() ([]string, error) {
	return l.git.GetCommits()
}

// RestoreSymlinks finds all managed items from .lnk file and ensures they have proper symlinks
func (l *Lnk) RestoreSymlinks() ([]string, error) {
	var restored []string

	// Get managed items from .lnk file (now containing relative paths)
	managedItems, err := l.getManagedItems()
	if err != nil {
		return nil, fmt.Errorf("failed to get managed items: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	for _, relativePath := range managedItems {
		// Generate repository name from relative path
		storagePath := l.getHostStoragePath()
		repoItem := filepath.Join(storagePath, relativePath)

		// Check if item exists in repository
		if _, err := os.Stat(repoItem); os.IsNotExist(err) {
			continue // Skip missing items
		}

		// Determine where the symlink should be created
		symlinkPath := filepath.Join(homeDir, relativePath)

		// Check if symlink already exists and is correct
		if l.isValidSymlink(symlinkPath, repoItem) {
			continue
		}

		// Ensure parent directory exists
		symlinkDir := filepath.Dir(symlinkPath)
		if err := os.MkdirAll(symlinkDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", symlinkDir, err)
		}

		// Remove existing file/symlink if it exists
		if _, err := os.Lstat(symlinkPath); err == nil {
			if err := os.RemoveAll(symlinkPath); err != nil {
				return nil, fmt.Errorf("failed to remove existing item %s: %w", symlinkPath, err)
			}
		}

		// Create symlink
		if err := l.fs.CreateSymlink(repoItem, symlinkPath); err != nil {
			return nil, err
		}

		restored = append(restored, relativePath)
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
