// Package filemanager handles adding and removing files from lnk management.
package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/yarlson/lnk/internal/fs"
	"github.com/yarlson/lnk/internal/git"
	"github.com/yarlson/lnk/internal/lnkerror"
	"github.com/yarlson/lnk/internal/tracker"
)

// ProgressCallback defines the signature for progress reporting callbacks.
type ProgressCallback func(current, total int, currentFile string)

// Manager handles adding and removing files from lnk management.
type Manager struct {
	repoPath string
	host     string
	git      *git.Git
	fs       *fs.FileSystem
	tracker  *tracker.Tracker
}

// New creates a new file Manager.
func New(repoPath, host string, g *git.Git, f *fs.FileSystem, t *tracker.Tracker) *Manager {
	return &Manager{
		repoPath: repoPath,
		host:     host,
		git:      g,
		fs:       f,
		tracker:  t,
	}
}

// Add moves a file or directory to the repository and creates a symlink.
func (fm *Manager) Add(filePath string) error {
	if err := fm.fs.ValidateFileForAdd(filePath); err != nil {
		return err
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	relativePath, err := fs.GetRelativePath(absPath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	storagePath := fm.tracker.HostStoragePath()
	destPath := filepath.Join(storagePath, relativePath)

	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	managedItems, err := fm.tracker.GetManagedItems()
	if err != nil {
		return fmt.Errorf("failed to get managed items: %w", err)
	}
	if slices.Contains(managedItems, relativePath) {
		return lnkerror.WithPath(lnkerror.ErrAlreadyManaged, relativePath)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if err := fm.fs.Move(absPath, destPath, info); err != nil {
		return err
	}

	if err := fm.fs.CreateSymlink(destPath, absPath); err != nil {
		_ = fm.fs.Move(destPath, absPath, info)
		return err
	}

	if err := fm.tracker.AddManagedItem(relativePath); err != nil {
		_ = os.Remove(absPath)
		_ = fm.fs.Move(destPath, absPath, info)
		return fmt.Errorf("failed to update tracking file: %w", err)
	}

	gitPath := relativePath
	if fm.host != "" {
		gitPath = filepath.Join(fm.host+".lnk", relativePath)
	}
	if err := fm.git.Add(gitPath); err != nil {
		_ = os.Remove(absPath)
		_ = fm.tracker.RemoveManagedItem(relativePath)
		_ = fm.fs.Move(destPath, absPath, info)
		return err
	}

	if err := fm.git.Add(fm.tracker.LnkFileName()); err != nil {
		_ = os.Remove(absPath)
		_ = fm.tracker.RemoveManagedItem(relativePath)
		_ = fm.fs.Move(destPath, absPath, info)
		return err
	}

	basename := filepath.Base(relativePath)
	if err := fm.git.Commit(fmt.Sprintf("lnk: added %s", basename)); err != nil {
		_ = os.Remove(absPath)
		_ = fm.tracker.RemoveManagedItem(relativePath)
		_ = fm.fs.Move(destPath, absPath, info)
		return err
	}

	return nil
}

// validatedFile holds pre-validated file information for batch operations.
type validatedFile struct {
	absPath      string
	relativePath string
	info         os.FileInfo
}

// AddMultiple adds multiple files in a single transaction with optional progress reporting.
func (fm *Manager) AddMultiple(paths []string, progress ProgressCallback) error {
	if len(paths) == 0 {
		return nil
	}

	// Phase 1: Validate all paths.
	files, err := fm.validatePaths(paths)
	if err != nil {
		return err
	}

	// Phase 2: Process files (move, symlink, track) with optional progress.
	rollbackActions, err := fm.processFiles(files, progress)
	if err != nil {
		return err
	}

	// Phase 3: Git operations.
	if err := fm.commitFiles(files, rollbackActions, progress != nil); err != nil {
		return err
	}

	return nil
}

// validatePaths validates all paths and returns validated file info.
func (fm *Manager) validatePaths(paths []string) ([]validatedFile, error) {
	var files []validatedFile

	for _, filePath := range paths {
		if err := fm.fs.ValidateFileForAdd(filePath); err != nil {
			return nil, fmt.Errorf("validation failed for %s: %w", filePath, err)
		}

		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %s: %w", filePath, err)
		}

		relativePath, err := fs.GetRelativePath(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get relative path for %s: %w", filePath, err)
		}

		managedItems, err := fm.tracker.GetManagedItems()
		if err != nil {
			return nil, fmt.Errorf("failed to get managed items: %w", err)
		}
		if slices.Contains(managedItems, relativePath) {
			return nil, lnkerror.WithPath(lnkerror.ErrAlreadyManaged, relativePath)
		}

		info, err := os.Stat(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to stat path %s: %w", filePath, err)
		}

		files = append(files, validatedFile{
			absPath:      absPath,
			relativePath: relativePath,
			info:         info,
		})
	}

	return files, nil
}

// processFiles moves files to the repo, creates symlinks, and updates tracking.
func (fm *Manager) processFiles(files []validatedFile, progress ProgressCallback) ([]func() error, error) {
	var rollbackActions []func() error
	total := len(files)

	for i, f := range files {
		if progress != nil {
			progress(i+1, total, filepath.Base(f.absPath))
		}

		storagePath := fm.tracker.HostStoragePath()
		destPath := filepath.Join(storagePath, f.relativePath)

		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			fm.RollbackAll(rollbackActions)
			return nil, fmt.Errorf("failed to create destination directory: %w", err)
		}

		if err := fm.fs.Move(f.absPath, destPath, f.info); err != nil {
			fm.RollbackAll(rollbackActions)
			return nil, fmt.Errorf("failed to move %s: %w", f.absPath, err)
		}

		if err := fm.fs.CreateSymlink(destPath, f.absPath); err != nil {
			_ = fm.fs.Move(destPath, f.absPath, f.info)
			fm.RollbackAll(rollbackActions)
			return nil, fmt.Errorf("failed to create symlink for %s: %w", f.absPath, err)
		}

		if err := fm.tracker.AddManagedItem(f.relativePath); err != nil {
			_ = os.Remove(f.absPath)
			_ = fm.fs.Move(destPath, f.absPath, f.info)
			fm.RollbackAll(rollbackActions)
			return nil, fmt.Errorf("failed to update tracking file for %s: %w", f.absPath, err)
		}

		rollbackActions = append(rollbackActions, fm.CreateRollbackAction(f.absPath, destPath, f.relativePath, f.info))
	}

	return rollbackActions, nil
}

// commitFiles stages all files and creates a single git commit.
func (fm *Manager) commitFiles(files []validatedFile, rollbackActions []func() error, recursive bool) error {
	for _, f := range files {
		gitPath := f.relativePath
		if fm.host != "" {
			gitPath = filepath.Join(fm.host+".lnk", f.relativePath)
		}
		if err := fm.git.Add(gitPath); err != nil {
			fm.RollbackAll(rollbackActions)
			return fmt.Errorf("failed to add %s to git: %w", f.absPath, err)
		}
	}

	if err := fm.git.Add(fm.tracker.LnkFileName()); err != nil {
		fm.RollbackAll(rollbackActions)
		return fmt.Errorf("failed to add tracking file to git: %w", err)
	}

	suffix := "files"
	if recursive {
		suffix = "files recursively"
	}
	commitMessage := fmt.Sprintf("lnk: added %d %s", len(files), suffix)
	if err := fm.git.Commit(commitMessage); err != nil {
		fm.RollbackAll(rollbackActions)
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

// CreateRollbackAction creates a rollback function for a single file operation.
func (fm *Manager) CreateRollbackAction(absPath, destPath, relativePath string, info os.FileInfo) func() error {
	return func() error {
		_ = os.Remove(absPath)
		_ = fm.tracker.RemoveManagedItem(relativePath)
		return fm.fs.Move(destPath, absPath, info)
	}
}

// RollbackAll executes rollback actions in reverse order.
func (fm *Manager) RollbackAll(actions []func() error) {
	for i := len(actions) - 1; i >= 0; i-- {
		_ = actions[i]()
	}
}

// AddRecursiveWithProgress adds directory contents individually with optional progress.
func (fm *Manager) AddRecursiveWithProgress(paths []string, progress ProgressCallback) error {
	var allFiles []string

	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", path, err)
		}

		info, err := os.Stat(absPath)
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", path, err)
		}

		if info.IsDir() {
			files, err := fm.WalkDirectory(absPath)
			if err != nil {
				return fmt.Errorf("failed to walk directory %s: %w", path, err)
			}
			allFiles = append(allFiles, files...)
		} else {
			allFiles = append(allFiles, absPath)
		}
	}

	if len(allFiles) == 0 {
		return fmt.Errorf("no files found to add")
	}

	const progressThreshold = 10
	if len(allFiles) > progressThreshold && progress != nil {
		return fm.AddMultiple(allFiles, progress)
	}

	return fm.AddMultiple(allFiles, nil)
}

// PreviewAdd simulates an add operation and returns files that would be affected.
func (fm *Manager) PreviewAdd(paths []string, recursive bool) ([]string, error) {
	var allFiles []string

	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %s: %w", path, err)
		}

		info, err := os.Stat(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to stat %s: %w", path, err)
		}

		if info.IsDir() && recursive {
			files, err := fm.WalkDirectory(absPath)
			if err != nil {
				return nil, fmt.Errorf("failed to walk directory %s: %w", path, err)
			}
			allFiles = append(allFiles, files...)
		} else {
			allFiles = append(allFiles, absPath)
		}
	}

	var validFiles []string
	for _, filePath := range allFiles {
		if err := fm.fs.ValidateFileForAdd(filePath); err != nil {
			return nil, fmt.Errorf("validation failed for %s: %w", filePath, err)
		}

		relativePath, err := fs.GetRelativePath(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get relative path for %s: %w", filePath, err)
		}

		managedItems, err := fm.tracker.GetManagedItems()
		if err != nil {
			return nil, fmt.Errorf("failed to get managed items: %w", err)
		}
		if slices.Contains(managedItems, relativePath) {
			return nil, fmt.Errorf("\u274c File is already managed by lnk: \033[31m%s\033[0m", relativePath)
		}

		validFiles = append(validFiles, filePath)
	}

	return validFiles, nil
}

// Remove removes a symlink and restores the original file or directory.
func (fm *Manager) Remove(filePath string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if err := fm.fs.ValidateSymlinkForRemove(absPath, fm.repoPath); err != nil {
		return err
	}

	relativePath, err := fs.GetRelativePath(absPath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	managedItems, err := fm.tracker.GetManagedItems()
	if err != nil {
		return fmt.Errorf("failed to get managed items: %w", err)
	}

	if !slices.Contains(managedItems, relativePath) {
		return lnkerror.WithPath(lnkerror.ErrNotManaged, relativePath)
	}

	target, err := os.Readlink(absPath)
	if err != nil {
		return fmt.Errorf("failed to read symlink: %w", err)
	}

	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(absPath), target)
	}

	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("failed to stat target: %w", err)
	}

	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	if err := fm.tracker.RemoveManagedItem(relativePath); err != nil {
		return fmt.Errorf("failed to update tracking file: %w", err)
	}

	gitPath := relativePath
	if fm.host != "" {
		gitPath = filepath.Join(fm.host+".lnk", relativePath)
	}
	if err := fm.git.Remove(gitPath); err != nil {
		return err
	}

	if err := fm.git.Add(fm.tracker.LnkFileName()); err != nil {
		return err
	}

	basename := filepath.Base(relativePath)
	if err := fm.git.Commit(fmt.Sprintf("lnk: removed %s", basename)); err != nil {
		return err
	}

	if err := fm.fs.Move(target, absPath, info); err != nil {
		return err
	}

	return nil
}

// RemoveForce removes a file from lnk tracking even if the symlink no longer exists.
func (fm *Manager) RemoveForce(filePath string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	relativePath, err := fs.GetRelativePath(absPath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	managedItems, err := fm.tracker.GetManagedItems()
	if err != nil {
		return fmt.Errorf("failed to get managed items: %w", err)
	}

	if !slices.Contains(managedItems, relativePath) {
		return lnkerror.WithPath(lnkerror.ErrNotManaged, relativePath)
	}

	// Remove symlink if it exists (ignore errors - it may already be gone)
	_ = os.Remove(absPath)

	if err := fm.tracker.RemoveManagedItem(relativePath); err != nil {
		return fmt.Errorf("failed to update tracking file: %w", err)
	}

	gitPath := relativePath
	if fm.host != "" {
		gitPath = filepath.Join(fm.host+".lnk", relativePath)
	}

	// Remove from git (ignore errors - file may not be in git index)
	_ = fm.git.Remove(gitPath)

	if err := fm.git.Add(fm.tracker.LnkFileName()); err != nil {
		return err
	}

	basename := filepath.Base(relativePath)
	if err := fm.git.Commit(fmt.Sprintf("lnk: force removed %s", basename)); err != nil {
		return err
	}

	// Try to delete the repository copy if it exists
	repoFilePath := filepath.Join(fm.repoPath, gitPath)
	if _, err := os.Stat(repoFilePath); err == nil {
		if err := os.RemoveAll(repoFilePath); err != nil {
			return fmt.Errorf("failed to remove repository copy: %w", err)
		}
	}

	return nil
}

// WalkDirectory walks through a directory and returns all regular files.
func (fm *Manager) WalkDirectory(dirPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if info.Mode()&os.ModeSymlink != 0 {
			files = append(files, path)
			return nil
		}

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
