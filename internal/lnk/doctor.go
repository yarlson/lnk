package lnk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yarlson/lnk/internal/lnkerror"
)

// DoctorResult contains the results of a doctor scan or execution.
type DoctorResult struct {
	// InvalidEntries are .lnk entries whose stored files no longer exist in the repo.
	InvalidEntries []string
	// BrokenSymlinks are managed entries whose symlinks at $HOME are broken or missing.
	BrokenSymlinks []string
}

// HasIssues returns true if any issues were found.
func (r *DoctorResult) HasIssues() bool {
	return len(r.InvalidEntries) > 0 || len(r.BrokenSymlinks) > 0
}

// TotalIssues returns the total number of issues found.
func (r *DoctorResult) TotalIssues() int {
	return len(r.InvalidEntries) + len(r.BrokenSymlinks)
}

// PreviewDoctor scans the repository for all types of issues WITHOUT making
// any changes. This is the read-only preview used by --dry-run.
// It checks for: invalid .lnk entries and broken symlinks.
func (l *Lnk) PreviewDoctor() (*DoctorResult, error) {
	// Check if repository is initialized
	if !l.git.IsGitRepository() {
		return nil, lnkerror.WithSuggestion(ErrNotInitialized, "run 'lnk init' first")
	}

	result := &DoctorResult{}

	// 1. Find invalid entries (tracked in .lnk but missing from repo storage)
	invalidEntries, err := l.findInvalidEntries()
	if err != nil {
		return nil, err
	}
	result.InvalidEntries = invalidEntries

	// 2. Find broken symlinks (tracked in .lnk, file exists in repo, but symlink is broken)
	brokenSymlinks, err := l.findBrokenSymlinks()
	if err != nil {
		return nil, err
	}
	result.BrokenSymlinks = brokenSymlinks

	return result, nil
}

// Doctor scans the repository for all types of issues and fixes them.
// It removes invalid entries and restores broken symlinks.
func (l *Lnk) Doctor() (*DoctorResult, error) {
	// Use PreviewDoctor for the scan phase
	result, err := l.PreviewDoctor()
	if err != nil {
		return nil, err
	}

	// If nothing to fix, return early
	if !result.HasIssues() {
		return result, nil
	}

	// 1. Fix broken symlinks (restore them via RestoreSymlinks)
	if len(result.BrokenSymlinks) > 0 {
		if _, err := l.RestoreSymlinks(); err != nil {
			return nil, fmt.Errorf("failed to restore symlinks: %w", err)
		}
	}

	// 2. Remove invalid entries from .lnk file
	if len(result.InvalidEntries) > 0 {
		managedItems, err := l.getManagedItems()
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

		if err := l.writeManagedItems(validItems); err != nil {
			return nil, fmt.Errorf("failed to update tracking file: %w", err)
		}

		if err := l.git.Add(l.getLnkFileName()); err != nil {
			return nil, err
		}

		word := "entries"
		if len(result.InvalidEntries) == 1 {
			word = "entry"
		}
		commitMsg := fmt.Sprintf("lnk: cleaned %d invalid %s", len(result.InvalidEntries), word)
		if err := l.git.Commit(commitMsg); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// findInvalidEntries returns .lnk entries whose stored files no longer exist in the repo.
func (l *Lnk) findInvalidEntries() ([]string, error) {
	managedItems, err := l.getManagedItems()
	if err != nil {
		return nil, fmt.Errorf("failed to get managed items: %w", err)
	}

	if len(managedItems) == 0 {
		return []string{}, nil
	}

	storagePath := l.getHostStoragePath()
	var invalidItems []string

	for _, relativePath := range managedItems {
		// Security: validate the path doesn't escape the storage directory
		cleaned := filepath.Clean(relativePath)
		if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
			// Path traversal attempt or absolute path — treat as invalid
			invalidItems = append(invalidItems, relativePath)
			continue
		}

		// Check if the stored file exists in the repository
		storedFile := filepath.Join(storagePath, cleaned)
		if _, err := os.Stat(storedFile); os.IsNotExist(err) {
			// File no longer exists in repo — invalid entry
			invalidItems = append(invalidItems, relativePath)
			continue
		}
	}

	return invalidItems, nil
}

// findBrokenSymlinks returns managed entries whose symlinks at $HOME are broken or missing.
// Only checks entries that have valid stored files in the repo (not invalid entries).
func (l *Lnk) findBrokenSymlinks() ([]string, error) {
	managedItems, err := l.getManagedItems()
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

	storagePath := l.getHostStoragePath()
	var brokenSymlinks []string

	for _, relativePath := range managedItems {
		// Skip invalid paths (path traversal, etc.)
		cleaned := filepath.Clean(relativePath)
		if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
			continue
		}

		// Only check entries whose repo file exists (otherwise it's an invalid entry, not a broken symlink)
		repoItem := filepath.Join(storagePath, cleaned)
		if _, err := os.Stat(repoItem); os.IsNotExist(err) {
			continue
		}

		// Check if the symlink at $HOME is valid
		symlinkPath := filepath.Join(homeDir, relativePath)
		if !l.isValidSymlink(symlinkPath, repoItem) {
			brokenSymlinks = append(brokenSymlinks, relativePath)
		}
	}

	return brokenSymlinks, nil
}
