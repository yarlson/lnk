package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yarlson/lnk/internal/fs"
	"github.com/yarlson/lnk/internal/git"
)

// Lnk represents the main application logic
type Lnk struct {
	repoPath string
	host     string // Host-specific configuration
	git      *git.Git
	fs       *fs.FileSystem
}

type Option func(*Lnk)

// WithHost sets the host for host-specific configuration
func WithHost(host string) Option {
	return func(l *Lnk) {
		l.host = host
	}
}

// NewLnk creates a new Lnk instance with optional configuration
func NewLnk(opts ...Option) *Lnk {
	repoPath := getRepoPath()
	lnk := &Lnk{
		repoPath: repoPath,
		host:     "",
		git:      git.New(repoPath),
		fs:       fs.New(),
	}

	for _, opt := range opts {
		opt(lnk)
	}

	return lnk
}

// GetCurrentHostname returns the current system hostname
func GetCurrentHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}
	return hostname, nil
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

// getHostStoragePath returns the storage path for host-specific or common files
func (l *Lnk) getHostStoragePath() string {
	if l.host == "" {
		// Common configuration - store in root of repo
		return l.repoPath
	}
	// Host-specific configuration - store in host subdirectory
	return filepath.Join(l.repoPath, l.host+".lnk")
}

// getLnkFileName returns the appropriate .lnk tracking file name
func (l *Lnk) getLnkFileName() string {
	if l.host == "" {
		return ".lnk"
	}
	return ".lnk." + l.host
}

// getRelativePath converts an absolute path to a relative path from home directory
func getRelativePath(absPath string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check if the file is under home directory
	relPath, err := filepath.Rel(homeDir, absPath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	// If the relative path starts with "..", the file is outside home directory
	// In this case, use the absolute path as relative (without the leading slash)
	if strings.HasPrefix(relPath, "..") {
		// Use absolute path but remove leading slash and drive letter (for cross-platform)
		cleanPath := strings.TrimPrefix(absPath, "/")
		return cleanPath, nil
	}

	return relPath, nil
}

// Init initializes the lnk repository
func (l *Lnk) Init() error {
	return l.InitWithRemote("")
}

// InitWithRemote initializes the lnk repository, optionally cloning from a remote
func (l *Lnk) InitWithRemote(remoteURL string) error {
	if remoteURL != "" {
		// Clone from remote
		return l.Clone(remoteURL)
	}

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
			return fmt.Errorf("❌ Directory \033[31m%s\033[0m contains an existing Git repository\n   💡 Please backup or move the existing repository before initializing lnk", l.repoPath)
		}
	}

	// No existing repository, initialize Git repository
	if err := l.git.Init(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	return nil
}

// Clone clones a repository from the given URL
func (l *Lnk) Clone(url string) error {
	if err := l.git.Clone(url); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
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

// Add moves a file or directory to the repository and creates a symlink
func (l *Lnk) Add(filePath string) error {
	// Validate the file or directory
	if err := l.fs.ValidateFileForAdd(filePath); err != nil {
		return err
	}

	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get relative path for tracking
	relativePath, err := getRelativePath(absPath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// Generate repository path from relative path
	storagePath := l.getHostStoragePath()
	destPath := filepath.Join(storagePath, relativePath)

	// Ensure destination directory exists (including parent directories for host-specific files)
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Check if this relative path is already managed
	managedItems, err := l.getManagedItems()
	if err != nil {
		return fmt.Errorf("failed to get managed items: %w", err)
	}
	for _, item := range managedItems {
		if item == relativePath {
			return fmt.Errorf("❌ File is already managed by lnk: \033[31m%s\033[0m", relativePath)
		}
	}

	// Check if it's a directory or file
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	// Move to repository (handles both files and directories)
	if info.IsDir() {
		if err := l.fs.MoveDirectory(absPath, destPath); err != nil {
			return fmt.Errorf("failed to move directory to repository: %w", err)
		}
	} else {
		if err := l.fs.MoveFile(absPath, destPath); err != nil {
			return fmt.Errorf("failed to move file to repository: %w", err)
		}
	}

	// Create symlink
	if err := l.fs.CreateSymlink(destPath, absPath); err != nil {
		// Try to restore the original if symlink creation fails
		if info.IsDir() {
			_ = l.fs.MoveDirectory(destPath, absPath) // Ignore error in cleanup
		} else {
			_ = l.fs.MoveFile(destPath, absPath) // Ignore error in cleanup
		}
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	// Add to .lnk tracking file using relative path
	if err := l.addManagedItem(relativePath); err != nil {
		// Try to restore the original state if tracking fails
		_ = os.Remove(absPath) // Ignore error in cleanup
		if info.IsDir() {
			_ = l.fs.MoveDirectory(destPath, absPath) // Ignore error in cleanup
		} else {
			_ = l.fs.MoveFile(destPath, absPath) // Ignore error in cleanup
		}
		return fmt.Errorf("failed to update tracking file: %w", err)
	}

	// Add both the item and .lnk file to git in a single commit
	// For host-specific files, we need to add the relative path from repo root
	gitPath := relativePath
	if l.host != "" {
		gitPath = filepath.Join(l.host+".lnk", relativePath)
	}
	if err := l.git.Add(gitPath); err != nil {
		// Try to restore the original state if git add fails
		_ = os.Remove(absPath)                // Ignore error in cleanup
		_ = l.removeManagedItem(relativePath) // Ignore error in cleanup
		if info.IsDir() {
			_ = l.fs.MoveDirectory(destPath, absPath) // Ignore error in cleanup
		} else {
			_ = l.fs.MoveFile(destPath, absPath) // Ignore error in cleanup
		}
		return fmt.Errorf("failed to add item to git: %w", err)
	}

	// Add .lnk file to the same commit
	if err := l.git.Add(l.getLnkFileName()); err != nil {
		// Try to restore the original state if git add fails
		_ = os.Remove(absPath)                // Ignore error in cleanup
		_ = l.removeManagedItem(relativePath) // Ignore error in cleanup
		if info.IsDir() {
			_ = l.fs.MoveDirectory(destPath, absPath) // Ignore error in cleanup
		} else {
			_ = l.fs.MoveFile(destPath, absPath) // Ignore error in cleanup
		}
		return fmt.Errorf("failed to add .lnk file to git: %w", err)
	}

	// Commit both changes together
	basename := filepath.Base(relativePath)
	if err := l.git.Commit(fmt.Sprintf("lnk: added %s", basename)); err != nil {
		// Try to restore the original state if commit fails
		_ = os.Remove(absPath)                // Ignore error in cleanup
		_ = l.removeManagedItem(relativePath) // Ignore error in cleanup
		if info.IsDir() {
			_ = l.fs.MoveDirectory(destPath, absPath) // Ignore error in cleanup
		} else {
			_ = l.fs.MoveFile(destPath, absPath) // Ignore error in cleanup
		}
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

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
		return fmt.Errorf("❌ File is not managed by lnk: \033[31m%s\033[0m", relativePath)
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
		return fmt.Errorf("failed to remove from git: %w", err)
	}

	// Add .lnk file to the same commit
	if err := l.git.Add(l.getLnkFileName()); err != nil {
		return fmt.Errorf("failed to add .lnk file to git: %w", err)
	}

	// Commit both changes together
	basename := filepath.Base(relativePath)
	if err := l.git.Commit(fmt.Sprintf("lnk: removed %s", basename)); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	// Move back from repository (handles both files and directories)
	if info.IsDir() {
		if err := l.fs.MoveDirectory(target, absPath); err != nil {
			return fmt.Errorf("failed to restore directory: %w", err)
		}
	} else {
		if err := l.fs.MoveFile(target, absPath); err != nil {
			return fmt.Errorf("failed to restore file: %w", err)
		}
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
	Dirty  bool
}

// Status returns the repository sync status
func (l *Lnk) Status() (*StatusInfo, error) {
	// Check if repository is initialized
	if !l.git.IsGitRepository() {
		return nil, fmt.Errorf("❌ Lnk repository not initialized\n   💡 Run \033[1mlnk init\033[0m first")
	}

	gitStatus, err := l.git.GetStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository status: %w", err)
	}

	return &StatusInfo{
		Ahead:  gitStatus.Ahead,
		Behind: gitStatus.Behind,
		Remote: gitStatus.Remote,
		Dirty:  gitStatus.Dirty,
	}, nil
}

// Push stages all changes and creates a sync commit, then pushes to remote
func (l *Lnk) Push(message string) error {
	// Check if repository is initialized
	if !l.git.IsGitRepository() {
		return fmt.Errorf("❌ Lnk repository not initialized\n   💡 Run \033[1mlnk init\033[0m first")
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
		return nil, fmt.Errorf("❌ Lnk repository not initialized\n   💡 Run \033[1mlnk init\033[0m first")
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

// List returns the list of files and directories currently managed by lnk
func (l *Lnk) List() ([]string, error) {
	// Check if repository is initialized
	if !l.git.IsGitRepository() {
		return nil, fmt.Errorf("❌ Lnk repository not initialized\n   💡 Run \033[1mlnk init\033[0m first")
	}

	// Get managed items from .lnk file
	managedItems, err := l.getManagedItems()
	if err != nil {
		return nil, fmt.Errorf("failed to get managed items: %w", err)
	}

	return managedItems, nil
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
			return nil, fmt.Errorf("failed to create symlink for %s: %w", relativePath, err)
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
