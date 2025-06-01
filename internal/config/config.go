package config

import (
	"context"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/yarlson/lnk/internal/errors"
	"github.com/yarlson/lnk/internal/fs"
	"github.com/yarlson/lnk/internal/models"
	"github.com/yarlson/lnk/internal/pathresolver"
)

// Config implements the service.ConfigManager interface
type Config struct {
	fileManager  *fs.FileManager
	pathResolver *pathresolver.Resolver
}

// New creates a new ConfigManager instance
func New(fileManager *fs.FileManager, pathResolver *pathresolver.Resolver) *Config {
	return &Config{
		fileManager:  fileManager,
		pathResolver: pathResolver,
	}
}

// LoadHostConfig loads the configuration for a specific host
func (cm *Config) LoadHostConfig(ctx context.Context, repoPath, host string) (*models.HostConfig, error) {
	managedFiles, err := cm.ListManagedFiles(ctx, repoPath, host)
	if err != nil {
		return nil, err
	}

	return &models.HostConfig{
		Name:         host,
		ManagedFiles: managedFiles,
		LastUpdate:   time.Now(),
	}, nil
}

// SaveHostConfig saves the configuration for a specific host
func (cm *Config) SaveHostConfig(ctx context.Context, repoPath string, config *models.HostConfig) error {
	// Convert managed files to relative paths for storage
	var relativePaths []string
	for _, file := range config.ManagedFiles {
		relativePaths = append(relativePaths, file.RelativePath)
	}

	// Sort for consistent ordering
	sort.Strings(relativePaths)

	return cm.writeManagedItems(ctx, repoPath, config.Name, relativePaths)
}

// AddManagedFileToHost adds a managed file to a host's configuration
func (cm *Config) AddManagedFileToHost(ctx context.Context, repoPath, host string, file models.ManagedFile) error {
	// Get current managed files
	managedFiles, err := cm.getManagedItems(ctx, repoPath, host)
	if err != nil {
		return err
	}

	// Check if already exists
	for _, item := range managedFiles {
		if item == file.RelativePath {
			return nil // Already managed
		}
	}

	// Add new item
	managedFiles = append(managedFiles, file.RelativePath)

	// Sort for consistent ordering
	sort.Strings(managedFiles)

	return cm.writeManagedItems(ctx, repoPath, host, managedFiles)
}

// RemoveManagedFileFromHost removes a managed file from a host's configuration
func (cm *Config) RemoveManagedFileFromHost(ctx context.Context, repoPath, host, relativePath string) error {
	// Get current managed files
	managedFiles, err := cm.getManagedItems(ctx, repoPath, host)
	if err != nil {
		return err
	}

	// Remove item
	var newManagedFiles []string
	for _, item := range managedFiles {
		if item != relativePath {
			newManagedFiles = append(newManagedFiles, item)
		}
	}

	return cm.writeManagedItems(ctx, repoPath, host, newManagedFiles)
}

// ListManagedFiles returns all files managed by a specific host
func (cm *Config) ListManagedFiles(ctx context.Context, repoPath, host string) ([]models.ManagedFile, error) {
	relativePaths, err := cm.getManagedItems(ctx, repoPath, host)
	if err != nil {
		return nil, err
	}

	var managedFiles []models.ManagedFile
	for _, relativePath := range relativePaths {
		// Get file storage path
		fileStoragePath, err := cm.pathResolver.GetFileStoragePathInRepo(repoPath, host, relativePath)
		if err != nil {
			return nil, errors.NewConfigNotFoundError(host).
				WithContext("relative_path", relativePath)
		}

		// Get original path (where symlink should be)
		originalPath, err := cm.pathResolver.GetAbsolutePathInHome(relativePath)
		if err != nil {
			return nil, errors.NewInvalidPathError(relativePath, "cannot convert to absolute path")
		}

		// Check if file exists and get info
		var isDirectory bool
		var mode os.FileMode
		if exists, err := cm.fileManager.Exists(ctx, fileStoragePath); err == nil && exists {
			if info, err := cm.fileManager.Stat(ctx, fileStoragePath); err == nil {
				isDirectory = info.IsDir()
				mode = info.Mode()
			}
		}

		managedFile := models.ManagedFile{
			OriginalPath: originalPath,
			RepoPath:     fileStoragePath,
			RelativePath: relativePath,
			Host:         host,
			IsDirectory:  isDirectory,
			Mode:         mode,
		}

		managedFiles = append(managedFiles, managedFile)
	}

	return managedFiles, nil
}

// GetManagedFile retrieves a specific managed file by relative path
func (cm *Config) GetManagedFile(ctx context.Context, repoPath, host, relativePath string) (*models.ManagedFile, error) {
	managedFiles, err := cm.ListManagedFiles(ctx, repoPath, host)
	if err != nil {
		return nil, err
	}

	for _, file := range managedFiles {
		if file.RelativePath == relativePath {
			return &file, nil
		}
	}

	return nil, errors.NewFileNotFoundError(relativePath)
}

// ConfigExists checks if a configuration file exists for the host
func (cm *Config) ConfigExists(ctx context.Context, repoPath, host string) (bool, error) {
	trackingFilePath, err := cm.pathResolver.GetTrackingFilePath(repoPath, host)
	if err != nil {
		return false, err
	}

	return cm.fileManager.Exists(ctx, trackingFilePath)
}

// getManagedItems returns the list of managed files and directories from .lnk file
// This is the core method that reads the plain text format
func (cm *Config) getManagedItems(ctx context.Context, repoPath, host string) ([]string, error) {
	trackingFilePath, err := cm.pathResolver.GetTrackingFilePath(repoPath, host)
	if err != nil {
		return nil, errors.NewConfigNotFoundError(host).
			WithContext("repo_path", repoPath)
	}

	// If .lnk file doesn't exist, return empty list
	exists, err := cm.fileManager.Exists(ctx, trackingFilePath)
	if err != nil {
		return nil, errors.NewFileSystemOperationError("check_exists", trackingFilePath, err)
	}
	if !exists {
		return []string{}, nil
	}

	content, err := cm.fileManager.ReadFile(ctx, trackingFilePath)
	if err != nil {
		return nil, errors.NewFileSystemOperationError("read", trackingFilePath, err)
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

// writeManagedItems writes the list of managed items to .lnk file
// This maintains the plain text line-by-line format for compatibility
func (cm *Config) writeManagedItems(ctx context.Context, repoPath, host string, items []string) error {
	trackingFilePath, err := cm.pathResolver.GetTrackingFilePath(repoPath, host)
	if err != nil {
		return errors.NewConfigNotFoundError(host).
			WithContext("repo_path", repoPath)
	}

	content := strings.Join(items, "\n")
	if len(items) > 0 {
		content += "\n"
	}

	if err := cm.fileManager.WriteFile(ctx, trackingFilePath, []byte(content), 0644); err != nil {
		return errors.NewFileSystemOperationError("write", trackingFilePath, err)
	}

	return nil
}
