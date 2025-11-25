package fs

import (
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

// ValidateFileForAdd validates that a file or directory can be added to lnk
func (fs *FileSystem) ValidateFileForAdd(filePath string) error {
	// Check if file exists and get its info
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &FileNotExistsError{Path: filePath, Err: err}
		}

		return &FileCheckError{Err: err}
	}

	// Allow both regular files and directories
	if !info.Mode().IsRegular() && !info.IsDir() {
		return &UnsupportedFileTypeError{
			Path:       filePath,
			Suggestion: "lnk can only manage regular files and directories",
		}
	}

	return nil
}

// ValidateSymlinkForRemove validates that a symlink can be removed from lnk
func (fs *FileSystem) ValidateSymlinkForRemove(filePath, repoPath string) error {
	// Check if file exists and is a symlink
	info, err := os.Lstat(filePath) // Use Lstat to not follow symlinks
	if err != nil {
		if os.IsNotExist(err) {
			return &FileNotExistsError{Path: filePath, Err: err}
		}

		return &FileCheckError{Err: err}
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return &NotManagedByLnkError{
			Path:       filePath,
			Suggestion: "Use 'lnk add' to manage this file first",
		}
	}

	// Get symlink target and resolve to absolute path
	target, err := os.Readlink(filePath)
	if err != nil {
		return &SymlinkReadError{Err: err}
	}

	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(filePath), target)
	}

	// Clean paths and check if target is inside the repository
	target = filepath.Clean(target)
	repoPath = filepath.Clean(repoPath)

	if !strings.HasPrefix(target, repoPath+string(filepath.Separator)) && target != repoPath {
		return &NotManagedByLnkError{
			Path:       filePath,
			Suggestion: "Use 'lnk add' to manage this file first",
		}
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
		return &DirectoryCreationError{Path: filepath.Dir(dst), Err: err}
	}

	// Move the file
	return os.Rename(src, dst)
}

// CreateSymlink creates a relative symlink from target to linkPath
func (fs *FileSystem) CreateSymlink(target, linkPath string) error {
	// Calculate relative path from linkPath to target
	relTarget, err := filepath.Rel(filepath.Dir(linkPath), target)
	if err != nil {
		return &RelativePathCalculationError{Err: err}
	}

	// Create the symlink
	return os.Symlink(relTarget, linkPath)
}

// MoveDirectory moves a directory from source to destination recursively
func (fs *FileSystem) MoveDirectory(src, dst string) error {
	// Ensure destination parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return &DirectoryCreationError{Path: filepath.Dir(dst), Err: err}
	}

	// Move the directory
	return os.Rename(src, dst)
}
