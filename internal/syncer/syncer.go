// Package syncer handles synchronization operations: status, diff, push, pull, list, and symlink restoration.
package syncer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yarlson/lnk/internal/fs"
	"github.com/yarlson/lnk/internal/git"
	"github.com/yarlson/lnk/internal/lnkerror"
	"github.com/yarlson/lnk/internal/tracker"
)

// StatusInfo contains repository sync status information.
type StatusInfo struct {
	Ahead  int
	Behind int
	Remote string
	Dirty  bool
}

// Syncer handles synchronization operations.
type Syncer struct {
	repoPath string
	host     string
	git      *git.Git
	fs       *fs.FileSystem
	tracker  *tracker.Tracker
}

// New creates a new Syncer.
func New(repoPath, host string, g *git.Git, f *fs.FileSystem, t *tracker.Tracker) *Syncer {
	return &Syncer{
		repoPath: repoPath,
		host:     host,
		git:      g,
		fs:       f,
		tracker:  t,
	}
}

// Status returns the repository sync status.
func (s *Syncer) Status() (*StatusInfo, error) {
	if !s.git.IsGitRepository() {
		return nil, lnkerror.WithSuggestion(lnkerror.ErrNotInitialized, "run 'lnk init' first")
	}

	gitStatus, err := s.git.GetStatus()
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
func (s *Syncer) Diff(color bool) (string, error) {
	if !s.git.IsGitRepository() {
		return "", lnkerror.WithSuggestion(lnkerror.ErrNotInitialized, "run 'lnk init' first")
	}

	return s.git.Diff(color)
}

// Push stages all changes and creates a sync commit, then pushes to remote.
func (s *Syncer) Push(message string) error {
	if !s.git.IsGitRepository() {
		return lnkerror.WithSuggestion(lnkerror.ErrNotInitialized, "run 'lnk init' first")
	}

	hasChanges, err := s.git.HasChanges()
	if err != nil {
		return err
	}

	if hasChanges {
		if err := s.git.AddAll(); err != nil {
			return err
		}

		if err := s.git.Commit(message); err != nil {
			return err
		}
	}

	return s.git.Push()
}

// Pull fetches changes from remote and restores symlinks as needed.
func (s *Syncer) Pull() ([]string, error) {
	if !s.git.IsGitRepository() {
		return nil, lnkerror.WithSuggestion(lnkerror.ErrNotInitialized, "run 'lnk init' first")
	}

	if err := s.git.Pull(); err != nil {
		return nil, err
	}

	restored, err := s.RestoreSymlinks()
	if err != nil {
		return nil, fmt.Errorf("failed to restore symlinks: %w", err)
	}

	return restored, nil
}

// List returns the list of files and directories currently managed by lnk.
func (s *Syncer) List() ([]string, error) {
	if !s.git.IsGitRepository() {
		return nil, lnkerror.WithSuggestion(lnkerror.ErrNotInitialized, "run 'lnk init' first")
	}

	managedItems, err := s.tracker.GetManagedItems()
	if err != nil {
		return nil, fmt.Errorf("failed to get managed items: %w", err)
	}

	return managedItems, nil
}

// GetCommits returns the list of commits.
func (s *Syncer) GetCommits() ([]string, error) {
	return s.git.GetCommits()
}

// RestoreSymlinks finds all managed items and ensures they have proper symlinks.
func (s *Syncer) RestoreSymlinks() ([]string, error) {
	var restored []string

	managedItems, err := s.tracker.GetManagedItems()
	if err != nil {
		return nil, fmt.Errorf("failed to get managed items: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	for _, relativePath := range managedItems {
		storagePath := s.tracker.HostStoragePath()
		repoItem := filepath.Join(storagePath, relativePath)

		if _, err := os.Stat(repoItem); os.IsNotExist(err) {
			continue
		}

		symlinkPath := filepath.Join(homeDir, relativePath)

		if s.IsValidSymlink(symlinkPath, repoItem) {
			continue
		}

		symlinkDir := filepath.Dir(symlinkPath)
		if err := os.MkdirAll(symlinkDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", symlinkDir, err)
		}

		if _, err := os.Lstat(symlinkPath); err == nil {
			if err := os.RemoveAll(symlinkPath); err != nil {
				return nil, fmt.Errorf("failed to remove existing item %s: %w", symlinkPath, err)
			}
		}

		if err := s.fs.CreateSymlink(repoItem, symlinkPath); err != nil {
			return nil, err
		}

		restored = append(restored, relativePath)
	}

	return restored, nil
}

// IsValidSymlink checks if the given path is a symlink pointing to the expected target.
func (s *Syncer) IsValidSymlink(symlinkPath, expectedTarget string) bool {
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		return false
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return false
	}

	target, err := os.Readlink(symlinkPath)
	if err != nil {
		return false
	}

	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(symlinkPath), target)
	}

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
