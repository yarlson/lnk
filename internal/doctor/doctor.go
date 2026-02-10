// Package doctor handles repository health scanning and repair.
package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yarlson/lnk/internal/git"
	"github.com/yarlson/lnk/internal/lnkerror"
	"github.com/yarlson/lnk/internal/syncer"
	"github.com/yarlson/lnk/internal/tracker"
)

// Result contains the results of a doctor scan or execution.
type Result struct {
	InvalidEntries []string
	BrokenSymlinks []string
}

// HasIssues returns true if any issues were found.
func (r *Result) HasIssues() bool {
	return len(r.InvalidEntries) > 0 || len(r.BrokenSymlinks) > 0
}

// TotalIssues returns the total number of issues found.
func (r *Result) TotalIssues() int {
	return len(r.InvalidEntries) + len(r.BrokenSymlinks)
}

// Checker handles repository health scanning and repair.
type Checker struct {
	repoPath string
	host     string
	git      *git.Git
	tracker  *tracker.Tracker
	syncer   *syncer.Syncer
}

// New creates a new health Checker.
func New(repoPath, host string, g *git.Git, t *tracker.Tracker, s *syncer.Syncer) *Checker {
	return &Checker{
		repoPath: repoPath,
		host:     host,
		git:      g,
		tracker:  t,
		syncer:   s,
	}
}

// Preview scans the repository for all types of issues WITHOUT making any changes.
func (d *Checker) Preview() (*Result, error) {
	if !d.git.IsGitRepository() {
		return nil, lnkerror.WithSuggestion(lnkerror.ErrNotInitialized, "run 'lnk init' first")
	}

	result := &Result{}

	invalidEntries, err := d.findInvalidEntries()
	if err != nil {
		return nil, err
	}
	result.InvalidEntries = invalidEntries

	brokenSymlinks, err := d.findBrokenSymlinks()
	if err != nil {
		return nil, err
	}
	result.BrokenSymlinks = brokenSymlinks

	return result, nil
}

// Fix scans the repository for all types of issues and fixes them.
func (d *Checker) Fix() (*Result, error) {
	result, err := d.Preview()
	if err != nil {
		return nil, err
	}

	if !result.HasIssues() {
		return result, nil
	}

	// Fix broken symlinks via the syncer's RestoreSymlinks.
	if len(result.BrokenSymlinks) > 0 {
		if _, err := d.syncer.RestoreSymlinks(); err != nil {
			return nil, fmt.Errorf("failed to restore symlinks: %w", err)
		}
	}

	// Remove invalid entries from .lnk file.
	if len(result.InvalidEntries) > 0 {
		managedItems, err := d.tracker.GetManagedItems()
		if err != nil {
			return nil, fmt.Errorf("failed to get managed items: %w", err)
		}

		removedSet := make(map[string]bool, len(result.InvalidEntries))
		for _, item := range result.InvalidEntries {
			removedSet[item] = true
		}

		var validItems []string
		for _, item := range managedItems {
			if !removedSet[item] {
				validItems = append(validItems, item)
			}
		}

		if err := d.tracker.WriteManagedItems(validItems); err != nil {
			return nil, fmt.Errorf("failed to update tracking file: %w", err)
		}

		if err := d.git.Add(d.tracker.LnkFileName()); err != nil {
			return nil, err
		}

		word := "entries"
		if len(result.InvalidEntries) == 1 {
			word = "entry"
		}
		commitMsg := fmt.Sprintf("lnk: cleaned %d invalid %s", len(result.InvalidEntries), word)
		if err := d.git.Commit(commitMsg); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// findInvalidEntries returns .lnk entries whose stored files no longer exist in the repo.
func (d *Checker) findInvalidEntries() ([]string, error) {
	managedItems, err := d.tracker.GetManagedItems()
	if err != nil {
		return nil, fmt.Errorf("failed to get managed items: %w", err)
	}

	if len(managedItems) == 0 {
		return []string{}, nil
	}

	storagePath := d.tracker.HostStoragePath()
	var invalidItems []string

	for _, relativePath := range managedItems {
		cleaned := filepath.Clean(relativePath)
		if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
			invalidItems = append(invalidItems, relativePath)
			continue
		}

		storedFile := filepath.Join(storagePath, cleaned)
		if _, err := os.Stat(storedFile); os.IsNotExist(err) {
			invalidItems = append(invalidItems, relativePath)
			continue
		}
	}

	return invalidItems, nil
}

// findBrokenSymlinks returns managed entries whose symlinks at $HOME are broken or missing.
func (d *Checker) findBrokenSymlinks() ([]string, error) {
	managedItems, err := d.tracker.GetManagedItems()
	if err != nil {
		return nil, fmt.Errorf("failed to get managed items: %w", err)
	}

	if len(managedItems) == 0 {
		return []string{}, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	storagePath := d.tracker.HostStoragePath()
	var brokenSymlinks []string

	for _, relativePath := range managedItems {
		cleaned := filepath.Clean(relativePath)
		if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
			continue
		}

		repoItem := filepath.Join(storagePath, cleaned)
		if _, err := os.Stat(repoItem); os.IsNotExist(err) {
			continue
		}

		symlinkPath := filepath.Join(homeDir, relativePath)
		if !d.syncer.IsValidSymlink(symlinkPath, repoItem) {
			brokenSymlinks = append(brokenSymlinks, relativePath)
		}
	}

	return brokenSymlinks, nil
}
