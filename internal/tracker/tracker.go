// Package tracker manages the .lnk tracking file that records which files are managed.
package tracker

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// Tracker manages the .lnk tracking file that records which files are managed.
type Tracker struct {
	repoPath string
	host     string
}

// New creates a new Tracker.
func New(repoPath, host string) *Tracker {
	return &Tracker{repoPath: repoPath, host: host}
}

// RepoPath returns the repository path.
func (t *Tracker) RepoPath() string {
	return t.repoPath
}

// LnkFileName returns the appropriate .lnk tracking file name.
func (t *Tracker) LnkFileName() string {
	if t.host == "" {
		return ".lnk"
	}
	return ".lnk." + t.host
}

// HostStoragePath returns the storage path for host-specific or common files.
func (t *Tracker) HostStoragePath() string {
	if t.host == "" {
		return t.repoPath
	}
	return filepath.Join(t.repoPath, t.host+".lnk")
}

// GetManagedItems returns the list of managed files and directories from .lnk file.
func (t *Tracker) GetManagedItems() ([]string, error) {
	lnkFile := filepath.Join(t.repoPath, t.LnkFileName())

	if _, err := os.Stat(lnkFile); os.IsNotExist(err) {
		return []string{}, nil
	}

	content, err := os.ReadFile(lnkFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read .lnk file: %w", err)
	}

	if len(content) == 0 {
		return []string{}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var items []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			items = append(items, line)
		}
	}

	return items, nil
}

// AddManagedItem adds an item to the .lnk tracking file.
func (t *Tracker) AddManagedItem(relativePath string) error {
	items, err := t.GetManagedItems()
	if err != nil {
		return fmt.Errorf("failed to get managed items: %w", err)
	}

	if slices.Contains(items, relativePath) {
		return nil // Already managed
	}

	items = append(items, relativePath)
	sort.Strings(items)

	return t.WriteManagedItems(items)
}

// RemoveManagedItem removes an item from the .lnk tracking file.
func (t *Tracker) RemoveManagedItem(relativePath string) error {
	items, err := t.GetManagedItems()
	if err != nil {
		return fmt.Errorf("failed to get managed items: %w", err)
	}

	var newItems []string
	for _, item := range items {
		if item != relativePath {
			newItems = append(newItems, item)
		}
	}

	return t.WriteManagedItems(newItems)
}

// WriteManagedItems writes the list of managed items to .lnk file.
func (t *Tracker) WriteManagedItems(items []string) error {
	lnkFile := filepath.Join(t.repoPath, t.LnkFileName())

	content := strings.Join(items, "\n")
	if len(items) > 0 {
		content += "\n"
	}

	err := os.WriteFile(lnkFile, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write .lnk file: %w", err)
	}

	return nil
}
