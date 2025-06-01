package fs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yarlson/lnk/internal/errors"
)

// FileManager implements the models.FileManager interface
type FileManager struct{}

// New creates a new FileManager instance
func New() *FileManager {
	return &FileManager{}
}

// Exists checks if a file or directory exists
func (fm *FileManager) Exists(ctx context.Context, path string) (bool, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.NewFileSystemOperationError("stat", path, err)
	}
	return true, nil
}

// IsDirectory checks if the path points to a directory
func (fm *FileManager) IsDirectory(ctx context.Context, path string) (bool, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, errors.NewFileNotFoundError(path)
		}
		return false, errors.NewFileSystemOperationError("stat", path, err)
	}
	return info.IsDir(), nil
}

// Move moves a file or directory from src to dst
func (fm *FileManager) Move(ctx context.Context, src, dst string) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := fm.MkdirAll(ctx, dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Check for context cancellation before move
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Move the file or directory
	if err := os.Rename(src, dst); err != nil {
		return errors.NewFileSystemOperationError("move", src, err).
			WithContext("destination", dst)
	}

	return nil
}

// CreateSymlink creates a symlink pointing from linkPath to target
func (fm *FileManager) CreateSymlink(ctx context.Context, target, linkPath string) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Calculate relative path from linkPath to target
	linkDir := filepath.Dir(linkPath)
	relTarget, err := filepath.Rel(linkDir, target)
	if err != nil {
		return errors.NewFileSystemOperationError("calculate_relative_path", linkPath, err).
			WithContext("target", target)
	}

	// Create the symlink
	if err := os.Symlink(relTarget, linkPath); err != nil {
		return errors.NewFileSystemOperationError("create_symlink", linkPath, err).
			WithContext("target", relTarget)
	}

	return nil
}

// Remove removes a file or directory
func (fm *FileManager) Remove(ctx context.Context, path string) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := os.RemoveAll(path); err != nil {
		return errors.NewFileSystemOperationError("remove", path, err)
	}

	return nil
}

// ReadFile reads the contents of a file
func (fm *FileManager) ReadFile(ctx context.Context, path string) ([]byte, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewFileNotFoundError(path)
		}
		return nil, errors.NewFileSystemOperationError("read", path, err)
	}

	return data, nil
}

// WriteFile writes data to a file with the given permissions
func (fm *FileManager) WriteFile(ctx context.Context, path string, data []byte, perm os.FileMode) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := fm.MkdirAll(ctx, dir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Check for context cancellation before write
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := os.WriteFile(path, data, perm); err != nil {
		return errors.NewFileSystemOperationError("write", path, err)
	}

	return nil
}

// MkdirAll creates a directory and all necessary parent directories
func (fm *FileManager) MkdirAll(ctx context.Context, path string, perm os.FileMode) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := os.MkdirAll(path, perm); err != nil {
		return errors.NewFileSystemOperationError("mkdir", path, err)
	}

	return nil
}

// Readlink returns the target of a symbolic link
func (fm *FileManager) Readlink(ctx context.Context, path string) (string, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	target, err := os.Readlink(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.NewFileNotFoundError(path)
		}
		return "", errors.NewFileSystemOperationError("readlink", path, err)
	}

	return target, nil
}

// Lstat returns file info without following symbolic links
func (fm *FileManager) Lstat(ctx context.Context, path string) (os.FileInfo, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewFileNotFoundError(path)
		}
		return nil, errors.NewFileSystemOperationError("lstat", path, err)
	}

	return info, nil
}

// Stat returns file info, following symbolic links
func (fm *FileManager) Stat(ctx context.Context, path string) (os.FileInfo, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewFileNotFoundError(path)
		}
		return nil, errors.NewFileSystemOperationError("stat", path, err)
	}

	return info, nil
}
