package core

import (
	"fmt"
	"os"
	"os/exec"
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
	return l.git.Init()
}

// Clone clones a repository from the given URL
func (l *Lnk) Clone(url string) error {
	return l.git.Clone(url)
}

// AddRemote adds a remote to the repository
func (l *Lnk) AddRemote(name, url string) error {
	return l.git.AddRemote(name, url)
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
	if err := l.fs.Move(absPath, destPath, info); err != nil {
		return err
	}

	// Create symlink
	if err := l.fs.CreateSymlink(destPath, absPath); err != nil {
		// Try to restore the original if symlink creation fails
		_ = l.fs.Move(destPath, absPath, info)
		return err
	}

	// Add to .lnk tracking file using relative path
	if err := l.addManagedItem(relativePath); err != nil {
		// Try to restore the original state if tracking fails
		_ = os.Remove(absPath)
		_ = l.fs.Move(destPath, absPath, info)
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
		_ = os.Remove(absPath)
		_ = l.removeManagedItem(relativePath)
		_ = l.fs.Move(destPath, absPath, info)
		return err
	}

	// Add .lnk file to the same commit
	if err := l.git.Add(l.getLnkFileName()); err != nil {
		// Try to restore the original state if git add fails
		_ = os.Remove(absPath)
		_ = l.removeManagedItem(relativePath)
		_ = l.fs.Move(destPath, absPath, info)
		return err
	}

	// Commit both changes together
	basename := filepath.Base(relativePath)
	if err := l.git.Commit(fmt.Sprintf("lnk: added %s", basename)); err != nil {
		// Try to restore the original state if commit fails
		_ = os.Remove(absPath)
		_ = l.removeManagedItem(relativePath)
		_ = l.fs.Move(destPath, absPath, info)
		return err
	}

	return nil
}

// AddMultiple adds multiple files or directories to the repository in a single transaction
func (l *Lnk) AddMultiple(paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	// Phase 1: Validate all paths first
	var relativePaths []string
	var absolutePaths []string
	var infos []os.FileInfo

	for _, filePath := range paths {
		// Validate the file or directory
		if err := l.fs.ValidateFileForAdd(filePath); err != nil {
			return fmt.Errorf("validation failed for %s: %w", filePath, err)
		}

		// Get absolute path
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", filePath, err)
		}

		// Get relative path for tracking
		relativePath, err := getRelativePath(absPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", filePath, err)
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

		// Get file info
		info, err := os.Stat(absPath)
		if err != nil {
			return fmt.Errorf("failed to stat path %s: %w", filePath, err)
		}

		relativePaths = append(relativePaths, relativePath)
		absolutePaths = append(absolutePaths, absPath)
		infos = append(infos, info)
	}

	// Phase 2: Process all files - move to repository and create symlinks
	var rollbackActions []func() error

	for i, absPath := range absolutePaths {
		relativePath := relativePaths[i]
		info := infos[i]

		// Generate repository path from relative path
		storagePath := l.getHostStoragePath()
		destPath := filepath.Join(storagePath, relativePath)

		// Ensure destination directory exists
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			// Rollback previous operations
			l.rollbackOperations(rollbackActions)
			return fmt.Errorf("failed to create destination directory: %w", err)
		}

		// Move to repository
		if err := l.fs.Move(absPath, destPath, info); err != nil {
			// Rollback previous operations
			l.rollbackOperations(rollbackActions)
			return fmt.Errorf("failed to move %s: %w", absPath, err)
		}

		// Create symlink
		if err := l.fs.CreateSymlink(destPath, absPath); err != nil {
			// Try to restore the file we just moved, then rollback others
			_ = l.fs.Move(destPath, absPath, info)
			l.rollbackOperations(rollbackActions)
			return fmt.Errorf("failed to create symlink for %s: %w", absPath, err)
		}

		// Add to tracking
		if err := l.addManagedItem(relativePath); err != nil {
			// Restore this file and rollback others
			_ = os.Remove(absPath)
			_ = l.fs.Move(destPath, absPath, info)
			l.rollbackOperations(rollbackActions)
			return fmt.Errorf("failed to update tracking file for %s: %w", absPath, err)
		}

		// Add rollback action for this file
		rollbackAction := l.createRollbackAction(absPath, destPath, relativePath, info)
		rollbackActions = append(rollbackActions, rollbackAction)
	}

	// Phase 3: Git operations - add all files and create single commit
	for i, relativePath := range relativePaths {
		// For host-specific files, we need to add the relative path from repo root
		gitPath := relativePath
		if l.host != "" {
			gitPath = filepath.Join(l.host+".lnk", relativePath)
		}
		if err := l.git.Add(gitPath); err != nil {
			// Rollback all operations
			l.rollbackOperations(rollbackActions)
			return fmt.Errorf("failed to add %s to git: %w", absolutePaths[i], err)
		}
	}

	// Add .lnk file to the same commit
	if err := l.git.Add(l.getLnkFileName()); err != nil {
		// Rollback all operations
		l.rollbackOperations(rollbackActions)
		return fmt.Errorf("failed to add tracking file to git: %w", err)
	}

	// Commit all changes together
	commitMessage := fmt.Sprintf("lnk: added %d files", len(paths))
	if err := l.git.Commit(commitMessage); err != nil {
		// Rollback all operations
		l.rollbackOperations(rollbackActions)
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

// createRollbackAction creates a rollback function for a single file operation
func (l *Lnk) createRollbackAction(absPath, destPath, relativePath string, info os.FileInfo) func() error {
	return func() error {
		_ = os.Remove(absPath)
		_ = l.removeManagedItem(relativePath)
		return l.fs.Move(destPath, absPath, info)
	}
}

// rollbackOperations executes rollback actions in reverse order
func (l *Lnk) rollbackOperations(rollbackActions []func() error) {
	for i := len(rollbackActions) - 1; i >= 0; i-- {
		_ = rollbackActions[i]()
	}
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
		return nil, err
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
		return nil, fmt.Errorf("❌ Lnk repository not initialized\n   💡 Run \033[1mlnk init\033[0m first")
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

// FindBootstrapScript searches for a bootstrap script in the repository
func (l *Lnk) FindBootstrapScript() (string, error) {
	// Check if repository is initialized
	if !l.git.IsGitRepository() {
		return "", fmt.Errorf("❌ Lnk repository not initialized\n   💡 Run \033[1mlnk init\033[0m first")
	}

	// Look for bootstrap.sh - simple, opinionated choice
	scriptPath := filepath.Join(l.repoPath, "bootstrap.sh")
	if _, err := os.Stat(scriptPath); err == nil {
		return "bootstrap.sh", nil
	}

	return "", nil // No bootstrap script found
}

// RunBootstrapScript executes the bootstrap script
func (l *Lnk) RunBootstrapScript(scriptName string) error {
	scriptPath := filepath.Join(l.repoPath, scriptName)

	// Verify the script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("❌ Bootstrap script not found: \033[31m%s\033[0m", scriptName)
	}

	// Make sure it's executable
	if err := os.Chmod(scriptPath, 0755); err != nil {
		return fmt.Errorf("❌ Failed to make bootstrap script executable: %w", err)
	}

	// Run with bash (since we only support bootstrap.sh)
	cmd := exec.Command("bash", scriptPath)

	// Set working directory to the repository
	cmd.Dir = l.repoPath

	// Connect to stdout/stderr for user to see output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Run the script
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("❌ Bootstrap script failed with error: %w", err)
	}

	return nil
}

// walkDirectory walks through a directory and returns all regular files
func (l *Lnk) walkDirectory(dirPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories - we only want files
		if info.IsDir() {
			return nil
		}

		// Handle symlinks: include them as files if they point to regular files
		if info.Mode()&os.ModeSymlink != 0 {
			// For symlinks, we'll include them but the AddMultiple logic
			// will handle validation appropriately
			files = append(files, path)
			return nil
		}

		// Include regular files
		if info.Mode().IsRegular() {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", dirPath, err)
	}

	return files, nil
}

// ProgressCallback defines the signature for progress reporting callbacks
type ProgressCallback func(current, total int, currentFile string)

// AddRecursiveWithProgress adds directory contents individually with progress reporting
func (l *Lnk) AddRecursiveWithProgress(paths []string, progress ProgressCallback) error {
	var allFiles []string

	for _, path := range paths {
		// Get absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", path, err)
		}

		// Check if it's a directory
		info, err := os.Stat(absPath)
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", path, err)
		}

		if info.IsDir() {
			// Walk directory to get all files
			files, err := l.walkDirectory(absPath)
			if err != nil {
				return fmt.Errorf("failed to walk directory %s: %w", path, err)
			}
			allFiles = append(allFiles, files...)
		} else {
			// It's a regular file, add it directly
			allFiles = append(allFiles, absPath)
		}
	}

	// Use AddMultiple for batch processing
	if len(allFiles) == 0 {
		return fmt.Errorf("no files found to add")
	}

	// Apply progress threshold: only show progress for >10 files
	const progressThreshold = 10
	if len(allFiles) > progressThreshold && progress != nil {
		return l.addMultipleWithProgress(allFiles, progress)
	}

	// For small operations, use regular AddMultiple without progress
	return l.AddMultiple(allFiles)
}

// addMultipleWithProgress adds multiple files with progress reporting
func (l *Lnk) addMultipleWithProgress(paths []string, progress ProgressCallback) error {
	if len(paths) == 0 {
		return nil
	}

	// Phase 1: Validate all paths first (same as AddMultiple)
	var relativePaths []string
	var absolutePaths []string
	var infos []os.FileInfo

	for _, filePath := range paths {
		// Validate the file or directory
		if err := l.fs.ValidateFileForAdd(filePath); err != nil {
			return fmt.Errorf("validation failed for %s: %w", filePath, err)
		}

		// Get absolute path
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", filePath, err)
		}

		// Get relative path for tracking
		relativePath, err := getRelativePath(absPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", filePath, err)
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

		// Get file info
		info, err := os.Stat(absPath)
		if err != nil {
			return fmt.Errorf("failed to stat path %s: %w", filePath, err)
		}

		relativePaths = append(relativePaths, relativePath)
		absolutePaths = append(absolutePaths, absPath)
		infos = append(infos, info)
	}

	// Phase 2: Process all files with progress reporting
	var rollbackActions []func() error
	total := len(absolutePaths)

	for i, absPath := range absolutePaths {
		// Report progress
		if progress != nil {
			progress(i+1, total, filepath.Base(absPath))
		}

		relativePath := relativePaths[i]
		info := infos[i]

		// Generate repository path from relative path
		storagePath := l.getHostStoragePath()
		destPath := filepath.Join(storagePath, relativePath)

		// Ensure destination directory exists
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			// Rollback previous operations
			l.rollbackOperations(rollbackActions)
			return fmt.Errorf("failed to create destination directory: %w", err)
		}

		// Move to repository
		if err := l.fs.Move(absPath, destPath, info); err != nil {
			// Rollback previous operations
			l.rollbackOperations(rollbackActions)
			return fmt.Errorf("failed to move %s: %w", absPath, err)
		}

		// Create symlink
		if err := l.fs.CreateSymlink(destPath, absPath); err != nil {
			// Try to restore the file we just moved, then rollback others
			_ = l.fs.Move(destPath, absPath, info)
			l.rollbackOperations(rollbackActions)
			return fmt.Errorf("failed to create symlink for %s: %w", absPath, err)
		}

		// Add to tracking
		if err := l.addManagedItem(relativePath); err != nil {
			// Restore this file and rollback others
			_ = os.Remove(absPath)
			_ = l.fs.Move(destPath, absPath, info)
			l.rollbackOperations(rollbackActions)
			return fmt.Errorf("failed to update tracking file for %s: %w", absPath, err)
		}

		// Add rollback action for this file
		rollbackAction := l.createRollbackAction(absPath, destPath, relativePath, info)
		rollbackActions = append(rollbackActions, rollbackAction)
	}

	// Phase 3: Git operations - add all files and create single commit
	for i, relativePath := range relativePaths {
		// For host-specific files, we need to add the relative path from repo root
		gitPath := relativePath
		if l.host != "" {
			gitPath = filepath.Join(l.host+".lnk", relativePath)
		}
		if err := l.git.Add(gitPath); err != nil {
			// Rollback all operations
			l.rollbackOperations(rollbackActions)
			return fmt.Errorf("failed to add %s to git: %w", absolutePaths[i], err)
		}
	}

	// Add .lnk file to the same commit
	if err := l.git.Add(l.getLnkFileName()); err != nil {
		// Rollback all operations
		l.rollbackOperations(rollbackActions)
		return fmt.Errorf("failed to add tracking file to git: %w", err)
	}

	// Commit all changes together
	commitMessage := fmt.Sprintf("lnk: added %d files recursively", len(paths))
	if err := l.git.Commit(commitMessage); err != nil {
		// Rollback all operations
		l.rollbackOperations(rollbackActions)
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

// AddRecursive adds directory contents individually instead of the directory as a whole
func (l *Lnk) AddRecursive(paths []string) error {
	var allFiles []string

	for _, path := range paths {
		// Get absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", path, err)
		}

		// Check if it's a directory
		info, err := os.Stat(absPath)
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", path, err)
		}

		if info.IsDir() {
			// Walk directory to get all files
			files, err := l.walkDirectory(absPath)
			if err != nil {
				return fmt.Errorf("failed to walk directory %s: %w", path, err)
			}
			allFiles = append(allFiles, files...)
		} else {
			// It's a regular file, add it directly
			allFiles = append(allFiles, absPath)
		}
	}

	// Use AddMultiple for batch processing
	if len(allFiles) == 0 {
		return fmt.Errorf("no files found to add")
	}

	return l.AddMultiple(allFiles)
}

// PreviewAdd simulates an add operation and returns files that would be affected
func (l *Lnk) PreviewAdd(paths []string, recursive bool) ([]string, error) {
	var allFiles []string

	for _, path := range paths {
		// Get absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %s: %w", path, err)
		}

		// Check if it's a directory
		info, err := os.Stat(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to stat %s: %w", path, err)
		}

		if info.IsDir() && recursive {
			// Walk directory to get all files (same logic as AddRecursive)
			files, err := l.walkDirectory(absPath)
			if err != nil {
				return nil, fmt.Errorf("failed to walk directory %s: %w", path, err)
			}
			allFiles = append(allFiles, files...)
		} else {
			// It's a regular file or non-recursive directory, add it directly
			allFiles = append(allFiles, absPath)
		}
	}

	// Validate files (same validation as AddMultiple but without making changes)
	var validFiles []string
	for _, filePath := range allFiles {
		// Validate the file or directory
		if err := l.fs.ValidateFileForAdd(filePath); err != nil {
			return nil, fmt.Errorf("validation failed for %s: %w", filePath, err)
		}

		// Get relative path for tracking
		relativePath, err := getRelativePath(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get relative path for %s: %w", filePath, err)
		}

		// Check if this relative path is already managed
		managedItems, err := l.getManagedItems()
		if err != nil {
			return nil, fmt.Errorf("failed to get managed items: %w", err)
		}
		for _, item := range managedItems {
			if item == relativePath {
				return nil, fmt.Errorf("❌ File is already managed by lnk: \033[31m%s\033[0m", relativePath)
			}
		}

		validFiles = append(validFiles, filePath)
	}

	return validFiles, nil
}
