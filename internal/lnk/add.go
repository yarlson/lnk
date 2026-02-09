package lnk

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yarlson/lnk/internal/lnkerror"
)

// ProgressCallback defines the signature for progress reporting callbacks
type ProgressCallback func(current, total int, currentFile string)

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
			return lnkerror.WithPath(ErrAlreadyManaged, relativePath)
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
				return lnkerror.WithPath(ErrAlreadyManaged, relativePath)
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
				return lnkerror.WithPath(ErrAlreadyManaged, relativePath)
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
	return l.AddRecursiveWithProgress(paths, nil)
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
