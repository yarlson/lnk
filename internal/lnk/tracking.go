package lnk

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// getManagedItems returns the list of managed files and directories from .lnk file
func (l *Lnk) getManagedItems() ([]string, error) {
	lnkFile := filepath.Join(l.repoPath, l.getLnkFileName())

	// If .lnk file doesn't exist, return empty list
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

// addManagedItem adds an item to the .lnk tracking file
func (l *Lnk) addManagedItem(relativePath string) error {
	// Get current items
	items, err := l.getManagedItems()
	if err != nil {
		return fmt.Errorf("failed to get managed items: %w", err)
	}

	// Check if already exists
	for _, item := range items {
		if item == relativePath {
			return nil // Already managed
		}
	}

	// Add new item using relative path
	items = append(items, relativePath)

	// Sort for consistent ordering
	sort.Strings(items)

	return l.writeManagedItems(items)
}

// removeManagedItem removes an item from the .lnk tracking file
func (l *Lnk) removeManagedItem(relativePath string) error {
	// Get current items
	items, err := l.getManagedItems()
	if err != nil {
		return fmt.Errorf("failed to get managed items: %w", err)
	}

	// Remove item using relative path
	var newItems []string
	for _, item := range items {
		if item != relativePath {
			newItems = append(newItems, item)
		}
	}

	return l.writeManagedItems(newItems)
}

// writeManagedItems writes the list of managed items to .lnk file
func (l *Lnk) writeManagedItems(items []string) error {
	lnkFile := filepath.Join(l.repoPath, l.getLnkFileName())

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
