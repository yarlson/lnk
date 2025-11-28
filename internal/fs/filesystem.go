// Package fs provides file system operations for lnk.
package fs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/yarlson/lnk/internal/lnkerr"
)

// Sentinel errors for file system operations.
var (
	ErrFileNotExists   = errors.New("File or directory not found")
	ErrFileCheck       = errors.New("Unable to access file. Please check file permissions and try again.")
	ErrUnsupportedType = errors.New("Cannot manage this type of file")
	ErrNotManaged      = errors.New("File is not managed by lnk")
	ErrSymlinkRead     = errors.New("Unable to read symlink. The file may be corrupted or have invalid permissions.")
	ErrDirCreate       = errors.New("Failed to create directory. Please check permissions and available disk space.")
	ErrRelativePath    = errors.New("Unable to create symlink due to path configuration issues. Please check file locations.")
)

// FileSystem handles file system operations
type FileSystem struct{}

// New creates a new FileSystem instance
func New() *FileSystem {
	return &FileSystem{}
}

// ValidateFileForAdd validates that a file or directory can be added to lnk
func (fs *FileSystem) ValidateFileForAdd(filePath string) error {
	// Check if file exists and get its info
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return lnkerr.WithPath(ErrFileNotExists, filePath)
		}

		return lnkerr.WithPath(ErrFileCheck, filePath)
	}

	// Allow both regular files and directories
	if !info.Mode().IsRegular() && !info.IsDir() {
		return lnkerr.WithPathAndSuggestion(ErrUnsupportedType, filePath, "lnk can only manage regular files and directories")
	}

	return nil
}

// ValidateSymlinkForRemove validates that a symlink can be removed from lnk
func (fs *FileSystem) ValidateSymlinkForRemove(filePath, repoPath string) error {
	// Check if file exists and is a symlink
	info, err := os.Lstat(filePath) // Use Lstat to not follow symlinks
	if err != nil {
		if os.IsNotExist(err) {
			return lnkerr.WithPath(ErrFileNotExists, filePath)
		}

		return lnkerr.WithPath(ErrFileCheck, filePath)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return lnkerr.WithPathAndSuggestion(ErrNotManaged, filePath, "use 'lnk add' to manage this file first")
	}

	// Get symlink target and resolve to absolute path
	target, err := os.Readlink(filePath)
	if err != nil {
		return lnkerr.WithPath(ErrSymlinkRead, filePath)
	}

	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(filePath), target)
	}

	// Clean paths and check if target is inside the repository
	target = filepath.Clean(target)
	repoPath = filepath.Clean(repoPath)

	if !strings.HasPrefix(target, repoPath+string(filepath.Separator)) && target != repoPath {
		return lnkerr.WithPathAndSuggestion(ErrNotManaged, filePath, "use 'lnk add' to manage this file first")
	}

	return nil
}

// Move moves a file or directory from source to destination based on the file info
func (fs *FileSystem) Move(src, dst string, info os.FileInfo) error {
	if info.IsDir() {
		return fs.MoveDirectory(src, dst)
	}
	return fs.MoveFile(src, dst)
}

// MoveFile moves a file from source to destination
func (fs *FileSystem) MoveFile(src, dst string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return lnkerr.WithPath(ErrDirCreate, filepath.Dir(dst))
	}

	// Move the file
	return os.Rename(src, dst)
}

// CreateSymlink creates a relative symlink from target to linkPath
func (fs *FileSystem) CreateSymlink(target, linkPath string) error {
	// Calculate relative path from linkPath to target
	relTarget, err := filepath.Rel(filepath.Dir(linkPath), target)
	if err != nil {
		return lnkerr.Wrap(ErrRelativePath)
	}

	// Create the symlink
	return os.Symlink(relTarget, linkPath)
}

// MoveDirectory moves a directory from source to destination recursively
func (fs *FileSystem) MoveDirectory(src, dst string) error {
	// Ensure destination parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return lnkerr.WithPath(ErrDirCreate, filepath.Dir(dst))
	}

	// Move the directory
	return os.Rename(src, dst)
}
