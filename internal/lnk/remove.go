package lnk

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yarlson/lnk/internal/lnkerror"
)

// Remove removes a symlink and restores the original file or directory
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

	// Get relative path for tracking
	relativePath, err := getRelativePath(absPath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// Check if this relative path is managed
	managedItems, err := l.getManagedItems()
	if err != nil {
		return fmt.Errorf("failed to get managed items: %w", err)
	}

	found := false
	for _, item := range managedItems {
		if item == relativePath {
			found = true
			break
		}
	}
	if !found {
		return lnkerror.WithPath(ErrNotManaged, relativePath)
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

	// Check if target is a directory or file
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("failed to stat target: %w", err)
	}

	// Remove the symlink
	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	// Remove from .lnk tracking file using relative path
	if err := l.removeManagedItem(relativePath); err != nil {
		return fmt.Errorf("failed to update tracking file: %w", err)
	}

	// Generate the correct git path for removal
	gitPath := relativePath
	if l.host != "" {
		gitPath = filepath.Join(l.host+".lnk", relativePath)
	}
	if err := l.git.Remove(gitPath); err != nil {
		return err
	}

	// Add .lnk file to the same commit
	if err := l.git.Add(l.getLnkFileName()); err != nil {
		return err
	}

	// Commit both changes together
	basename := filepath.Base(relativePath)
	if err := l.git.Commit(fmt.Sprintf("lnk: removed %s", basename)); err != nil {
		return err
	}

	// Move back from repository (handles both files and directories)
	if err := l.fs.Move(target, absPath, info); err != nil {
		return err
	}

	return nil
}

// RemoveForce removes a file from lnk tracking even if the symlink no longer exists.
// This is useful when a user accidentally deletes a managed file without using lnk rm.
func (l *Lnk) RemoveForce(filePath string) error {
	// Get relative path for tracking
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	relativePath, err := getRelativePath(absPath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// Check if this relative path is managed
	managedItems, err := l.getManagedItems()
	if err != nil {
		return fmt.Errorf("failed to get managed items: %w", err)
	}

	found := false
	for _, item := range managedItems {
		if item == relativePath {
			found = true
			break
		}
	}
	if !found {
		return lnkerror.WithPath(ErrNotManaged, relativePath)
	}

	// Remove symlink if it exists (ignore errors - it may already be gone)
	_ = os.Remove(absPath)

	// Remove from .lnk tracking file
	if err := l.removeManagedItem(relativePath); err != nil {
		return fmt.Errorf("failed to update tracking file: %w", err)
	}

	// Generate the correct git path for removal
	gitPath := relativePath
	if l.host != "" {
		gitPath = filepath.Join(l.host+".lnk", relativePath)
	}

	// Remove from git (ignore errors - file may not be in git index)
	_ = l.git.Remove(gitPath)

	// Add .lnk file to the commit
	if err := l.git.Add(l.getLnkFileName()); err != nil {
		return err
	}

	// Commit the change
	basename := filepath.Base(relativePath)
	if err := l.git.Commit(fmt.Sprintf("lnk: force removed %s", basename)); err != nil {
		return err
	}

	// Try to delete the repository copy if it exists
	repoPath := filepath.Join(l.repoPath, gitPath)
	if _, err := os.Stat(repoPath); err == nil {
		// File exists in repo, remove it
		if err := os.RemoveAll(repoPath); err != nil {
			return fmt.Errorf("failed to remove repository copy: %w", err)
		}
	}

	return nil
}
