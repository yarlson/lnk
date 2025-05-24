package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileSystem handles file system operations
type FileSystem struct{}

// New creates a new FileSystem instance
func New() *FileSystem {
	return &FileSystem{}
}

// ValidateFileForAdd validates that a file can be added to lnk
func (fs *FileSystem) ValidateFileForAdd(filePath string) error {
	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", filePath)
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if it's a directory
	if info.IsDir() {
		return fmt.Errorf("directories are not supported: %s", filePath)
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("only regular files are supported: %s", filePath)
	}

	return nil
}

// ValidateSymlinkForRemove validates that a symlink can be removed from lnk
func (fs *FileSystem) ValidateSymlinkForRemove(filePath, repoPath string) error {
	// Check if file exists
	info, err := os.Lstat(filePath) // Use Lstat to not follow symlinks
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", filePath)
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if it's a symlink
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("file is not managed by lnk: %s", filePath)
	}

	// Check if symlink points to the repository
	target, err := os.Readlink(filePath)
	if err != nil {
		return fmt.Errorf("failed to read symlink: %w", err)
	}

	// Convert relative path to absolute if needed
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(filePath), target)
	}

	// Clean the path to resolve any .. or . components
	target = filepath.Clean(target)
	repoPath = filepath.Clean(repoPath)

	// Check if target is inside the repository
	if !strings.HasPrefix(target, repoPath+string(filepath.Separator)) && target != repoPath {
		return fmt.Errorf("file is not managed by lnk: %s", filePath)
	}

	return nil
}

// MoveFile moves a file from source to destination
func (fs *FileSystem) MoveFile(src, dst string) error {
	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Move the file
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("failed to move file from %s to %s: %w", src, dst, err)
	}

	return nil
}

// CreateSymlink creates a relative symlink from target to linkPath
func (fs *FileSystem) CreateSymlink(target, linkPath string) error {
	// Calculate relative path from linkPath to target
	linkDir := filepath.Dir(linkPath)
	relTarget, err := filepath.Rel(linkDir, target)
	if err != nil {
		return fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Create the symlink
	if err := os.Symlink(relTarget, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}
