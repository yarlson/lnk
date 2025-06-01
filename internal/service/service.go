package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yarlson/lnk/internal/config"
	"github.com/yarlson/lnk/internal/errors"
	"github.com/yarlson/lnk/internal/fs"
	"github.com/yarlson/lnk/internal/git"
	"github.com/yarlson/lnk/internal/models"
	"github.com/yarlson/lnk/internal/pathresolver"
)

// FileManager handles file system operations
type FileManager interface {
	Exists(ctx context.Context, path string) (bool, error)
	Move(ctx context.Context, src, dst string) error
	CreateSymlink(ctx context.Context, target, linkPath string) error
	Remove(ctx context.Context, path string) error
	MkdirAll(ctx context.Context, path string, perm os.FileMode) error
	Readlink(ctx context.Context, path string) (string, error)
	Lstat(ctx context.Context, path string) (os.FileInfo, error)
	Stat(ctx context.Context, path string) (os.FileInfo, error)
}

// ConfigManager handles configuration persistence (reading and writing .lnk files)
type ConfigManager interface {
	AddManagedFileToHost(ctx context.Context, repoPath, host string, file models.ManagedFile) error
	RemoveManagedFileFromHost(ctx context.Context, repoPath, host, relativePath string) error
	ListManagedFiles(ctx context.Context, repoPath, host string) ([]models.ManagedFile, error)
	GetManagedFile(ctx context.Context, repoPath, host, relativePath string) (*models.ManagedFile, error)
}

// GitManager handles Git operations
type GitManager interface {
	Init(ctx context.Context, repoPath string) error
	Clone(ctx context.Context, repoPath, url string) error
	Add(ctx context.Context, repoPath string, files ...string) error
	Remove(ctx context.Context, repoPath string, files ...string) error
	Commit(ctx context.Context, repoPath, message string) error
	Push(ctx context.Context, repoPath string) error
	Pull(ctx context.Context, repoPath string) error
	Status(ctx context.Context, repoPath string) (*models.SyncStatus, error)
	IsRepository(ctx context.Context, repoPath string) (bool, error)
	HasChanges(ctx context.Context, repoPath string) (bool, error)
	IsLnkRepository(ctx context.Context, repoPath string) (bool, error)
}

// PathResolver handles path resolution and manipulation
type PathResolver interface {
	GetFileStoragePathInRepo(repoPath, host, relativePath string) (string, error)
	GetTrackingFilePath(repoPath, host string) (string, error)
	GetHomePath() (string, error)
	GetRelativePathFromHome(absPath string) (string, error)
	GetAbsolutePathInHome(relPath string) (string, error)
}

// Service encapsulates the business logic for lnk operations
type Service struct {
	fileManager   FileManager
	gitManager    GitManager // May be nil for some operations
	configManager ConfigManager
	pathResolver  PathResolver
	repoPath      string
}

// New creates a new Service instance with default dependencies
func New() (*Service, error) {
	// Initialize adapters
	fileManager := fs.New()
	gitManager := git.New()
	pathResolver := pathresolver.New()
	configManager := config.New(fileManager, pathResolver)

	// Get repository path
	repoPath, err := pathResolver.GetRepoStoragePath()
	if err != nil {
		return nil, errors.NewInvalidPathError("", "failed to determine repository storage path").
			WithContext("error", err.Error())
	}

	return &Service{
		fileManager:   fileManager,
		gitManager:    gitManager,
		configManager: configManager,
		pathResolver:  pathResolver,
		repoPath:      repoPath,
	}, nil
}

// NewLnkServiceWithDeps creates a new Service instance with provided dependencies (for testing)
func NewLnkServiceWithDeps(
	fileManager FileManager,
	gitManager GitManager,
	configManager ConfigManager,
	pathResolver PathResolver,
	repoPath string,
) *Service {
	return &Service{
		fileManager:   fileManager,
		gitManager:    gitManager,
		configManager: configManager,
		pathResolver:  pathResolver,
		repoPath:      repoPath,
	}
}

// ListManagedFiles returns the list of files managed by lnk for a specific host
// If host is empty, returns common configuration files
func (s *Service) ListManagedFiles(ctx context.Context, host string) ([]models.ManagedFile, error) {
	// Check if the repository exists
	exists, err := s.fileManager.Exists(ctx, s.repoPath)
	if err != nil {
		return nil, errors.NewFileSystemOperationError("check_repo_exists", s.repoPath, err)
	}

	if !exists {
		return nil, errors.NewRepoNotInitializedError(s.repoPath)
	}

	// Use the config manager to list managed files
	managedFiles, err := s.configManager.ListManagedFiles(ctx, s.repoPath, host)
	if err != nil {
		return nil, err // ConfigManager already returns properly typed errors
	}

	return managedFiles, nil
}

// GetStatus returns the Git repository status
// Returns an error if the repository is not initialized or GitManager is not available
func (s *Service) GetStatus(ctx context.Context) (*models.SyncStatus, error) {
	// Check if GitManager is available
	if s.gitManager == nil {
		return nil, errors.NewGitOperationError("get_status",
			fmt.Errorf("git manager not available"))
	}

	// Check if the repository exists
	exists, err := s.fileManager.Exists(ctx, s.repoPath)
	if err != nil {
		return nil, errors.NewFileSystemOperationError("check_repo_exists", s.repoPath, err)
	}

	if !exists {
		return nil, errors.NewRepoNotInitializedError(s.repoPath)
	}

	// Check if it's a Git repository
	isRepo, err := s.gitManager.IsRepository(ctx, s.repoPath)
	if err != nil {
		return nil, errors.NewGitOperationError("check_git_repo", err)
	}

	if !isRepo {
		return nil, errors.NewRepoNotInitializedError(s.repoPath).
			WithContext("reason", "directory exists but is not a git repository")
	}

	// Get Git status
	status, err := s.gitManager.Status(ctx, s.repoPath)
	if err != nil {
		return nil, err // GitManager already returns properly typed errors
	}

	return status, nil
}

// GetRepoPath returns the repository path
func (s *Service) GetRepoPath() string {
	return s.repoPath
}

// IsRepositoryInitialized checks if the lnk repository has been initialized
func (s *Service) IsRepositoryInitialized(ctx context.Context) (bool, error) {
	// Check if repository directory exists
	exists, err := s.fileManager.Exists(ctx, s.repoPath)
	if err != nil {
		return false, errors.NewFileSystemOperationError("check_repo_exists", s.repoPath, err)
	}

	if !exists {
		return false, nil
	}

	// Check if it's a Git repository (if GitManager is available)
	if s.gitManager != nil {
		isGitRepo, err := s.gitManager.IsRepository(ctx, s.repoPath)
		if err != nil {
			return false, errors.NewGitOperationError("check_git_repo", err)
		}
		return isGitRepo, nil
	}

	// If no GitManager, just check if the directory exists
	return true, nil
}

// InitializeRepository initializes a new lnk repository, optionally cloning from a remote URL
func (s *Service) InitializeRepository(ctx context.Context, remoteURL string) error {
	// Check if GitManager is available
	if s.gitManager == nil {
		return errors.NewGitOperationError("initialize_repository",
			fmt.Errorf("git manager not available"))
	}

	if remoteURL != "" {
		// Clone from remote
		return s.cloneRepository(ctx, remoteURL)
	}

	// Initialize empty repository
	return s.initEmptyRepository(ctx)
}

// cloneRepository clones a repository from the given URL
func (s *Service) cloneRepository(ctx context.Context, remoteURL string) error {
	// Clone using GitManager
	if err := s.gitManager.Clone(ctx, s.repoPath, remoteURL); err != nil {
		return errors.NewGitOperationError("clone_repository", err).
			WithContext("remote_url", remoteURL).
			WithContext("repo_path", s.repoPath)
	}

	return nil
}

// initEmptyRepository initializes an empty Git repository
func (s *Service) initEmptyRepository(ctx context.Context) error {
	// Check if repository directory already exists
	exists, err := s.fileManager.Exists(ctx, s.repoPath)
	if err != nil {
		return errors.NewFileSystemOperationError("check_repo_exists", s.repoPath, err)
	}

	if exists {
		// Check if it's already a Git repository
		isGitRepo, err := s.gitManager.IsRepository(ctx, s.repoPath)
		if err != nil {
			return errors.NewGitOperationError("check_git_repo", err)
		}

		if isGitRepo {
			// Check if it's a lnk repository
			isLnkRepo, err := s.gitManager.IsLnkRepository(ctx, s.repoPath)
			if err != nil {
				return errors.NewGitOperationError("check_lnk_repo", err)
			}

			if isLnkRepo {
				// It's already a lnk repository, init is idempotent
				return nil
			} else {
				// It's not a lnk repository, error to prevent data loss
				return errors.NewRepoNotInitializedError(s.repoPath).
					WithContext("reason", "directory contains an existing non-lnk Git repository")
			}
		}
	}

	// Create the repository directory if it doesn't exist
	if !exists {
		if err := s.fileManager.MkdirAll(ctx, s.repoPath, 0755); err != nil {
			return errors.NewFileSystemOperationError("create_repo_dir", s.repoPath, err)
		}
	}

	// Initialize Git repository
	if err := s.gitManager.Init(ctx, s.repoPath); err != nil {
		// Clean up directory if we created it
		if !exists {
			_ = s.fileManager.Remove(ctx, s.repoPath) // Ignore cleanup errors
		}
		return errors.NewGitOperationError("init_git_repo", err).
			WithContext("repo_path", s.repoPath)
	}

	return nil
}

// AddFile adds a file or directory to lnk management for the specified host
// This involves moving the file to the repository, creating a symlink, updating tracking, and committing to Git
func (s *Service) AddFile(ctx context.Context, filePath, host string) (*models.ManagedFile, error) {
	// Check if GitManager is available
	if s.gitManager == nil {
		return nil, errors.NewGitOperationError("add_file",
			fmt.Errorf("git manager not available"))
	}

	// Get absolute path
	absPath, err := s.pathResolver.GetAbsolutePathInHome(filePath)
	if err != nil {
		// If it fails, try as-is (might be already absolute)
		var pathErr error
		absPath, pathErr = filepath.Abs(filePath)
		if pathErr != nil {
			return nil, errors.NewFileSystemOperationError("resolve_path", filePath, err)
		}
	}

	// Validate that the file exists and is accessible (check this FIRST like the old implementation)
	exists, err := s.fileManager.Exists(ctx, absPath)
	if err != nil {
		return nil, errors.NewFileSystemOperationError("check_file_exists", absPath, err)
	}
	if !exists {
		return nil, errors.NewFileNotFoundError(absPath)
	}

	// Check if repository is initialized (after file existence check)
	initialized, err := s.IsRepositoryInitialized(ctx)
	if err != nil {
		return nil, err
	}
	if !initialized {
		return nil, errors.NewRepoNotInitializedError(s.repoPath)
	}

	// Get file information to determine if it's a directory
	fileInfo, err := s.fileManager.Stat(ctx, absPath)
	if err != nil {
		return nil, errors.NewFileSystemOperationError("stat_file", absPath, err)
	}

	// Get relative path for tracking
	relativePath, err := s.pathResolver.GetRelativePathFromHome(absPath)
	if err != nil {
		return nil, errors.NewFileSystemOperationError("get_relative_path", absPath, err)
	}

	// Check if file is already managed
	existingFile, err := s.configManager.GetManagedFile(ctx, s.repoPath, host, relativePath)
	if err == nil && existingFile != nil {
		return nil, errors.NewFileAlreadyManagedError(relativePath)
	}

	// Create managed file model
	managedFile := models.ManagedFile{
		OriginalPath: absPath,
		RelativePath: relativePath,
		Host:         host,
		IsDirectory:  fileInfo.IsDir(),
	}

	// Get storage path in repository
	storagePath, err := s.pathResolver.GetFileStoragePathInRepo(s.repoPath, host, relativePath)
	if err != nil {
		return nil, errors.NewFileSystemOperationError("get_storage_path", relativePath, err)
	}

	managedFile.RepoPath = storagePath

	// Execute the file addition with rollback support
	if err := s.executeFileAddition(ctx, &managedFile); err != nil {
		return nil, err
	}

	return &managedFile, nil
}

// executeFileAddition performs the actual file addition with rollback logic
func (s *Service) executeFileAddition(ctx context.Context, file *models.ManagedFile) error {
	var rollbackActions []func() error

	// Helper function to add rollback action
	addRollback := func(action func() error) {
		rollbackActions = append([]func() error{action}, rollbackActions...)
	}

	// Execute rollback if any step fails
	defer func() {
		if len(rollbackActions) > 0 {
			for _, action := range rollbackActions {
				_ = action() // Ignore rollback errors
			}
		}
	}()

	// Step 1: Create destination directory
	destDir := filepath.Dir(file.RepoPath)
	if err := s.fileManager.MkdirAll(ctx, destDir, 0755); err != nil {
		return errors.NewFileSystemOperationError("create_dest_dir", destDir, err)
	}

	// Step 2: Move file to repository
	if err := s.fileManager.Move(ctx, file.OriginalPath, file.RepoPath); err != nil {
		return errors.NewFileSystemOperationError("move_file", file.OriginalPath, err)
	}

	// Add rollback for move operation
	addRollback(func() error {
		return s.fileManager.Move(context.Background(), file.RepoPath, file.OriginalPath)
	})

	// Step 3: Create symlink
	if err := s.fileManager.CreateSymlink(ctx, file.RepoPath, file.OriginalPath); err != nil {
		return errors.NewFileSystemOperationError("create_symlink", file.OriginalPath, err)
	}

	// Add rollback for symlink creation
	addRollback(func() error {
		return s.fileManager.Remove(context.Background(), file.OriginalPath)
	})

	// Step 4: Add to config tracking
	if err := s.configManager.AddManagedFileToHost(ctx, s.repoPath, file.Host, *file); err != nil {
		return err // ConfigManager returns properly typed errors
	}

	// Add rollback for config update
	addRollback(func() error {
		return s.configManager.RemoveManagedFileFromHost(context.Background(),
			s.repoPath, file.Host, file.RelativePath)
	})

	// Step 5: Add file to Git
	gitPath := file.RelativePath
	if file.Host != "" {
		gitPath = filepath.Join(file.Host+".lnk", file.RelativePath)
	}

	if err := s.gitManager.Add(ctx, s.repoPath, gitPath); err != nil {
		return errors.NewGitOperationError("add_file_to_git", err)
	}

	// Step 6: Add config file to Git
	trackingFile, err := s.pathResolver.GetTrackingFilePath(s.repoPath, file.Host)
	if err != nil {
		return errors.NewFileSystemOperationError("get_tracking_file", "", err)
	}

	// Get relative path of tracking file from repo root
	trackingFileRel, err := filepath.Rel(s.repoPath, trackingFile)
	if err != nil {
		return errors.NewFileSystemOperationError("get_tracking_file_rel", trackingFile, err)
	}

	if err := s.gitManager.Add(ctx, s.repoPath, trackingFileRel); err != nil {
		return errors.NewGitOperationError("add_tracking_file_to_git", err)
	}

	// Step 7: Commit changes
	basename := filepath.Base(file.RelativePath)
	commitMessage := fmt.Sprintf("lnk: added %s", basename)
	if err := s.gitManager.Commit(ctx, s.repoPath, commitMessage); err != nil {
		return errors.NewGitOperationError("commit_changes", err)
	}

	// If we reach here, everything succeeded - clear rollback actions
	rollbackActions = nil
	return nil
}

// RemoveFile removes a file or directory from lnk management for the specified host
// This involves removing the symlink, restoring the original file, updating tracking, and committing to Git
func (s *Service) RemoveFile(ctx context.Context, filePath, host string) error {
	// Check if GitManager is available
	if s.gitManager == nil {
		return errors.NewGitOperationError("remove_file",
			fmt.Errorf("git manager not available"))
	}

	// Check if repository is initialized
	initialized, err := s.IsRepositoryInitialized(ctx)
	if err != nil {
		return err
	}
	if !initialized {
		return errors.NewRepoNotInitializedError(s.repoPath)
	}

	// Get absolute path
	absPath, err := s.pathResolver.GetAbsolutePathInHome(filePath)
	if err != nil {
		// If it fails, try as-is (might be already absolute)
		var pathErr error
		absPath, pathErr = filepath.Abs(filePath)
		if pathErr != nil {
			return errors.NewFileSystemOperationError("resolve_path", filePath, err)
		}
	}

	// Validate that this is a symlink
	linkInfo, err := s.fileManager.Lstat(ctx, absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.NewFileNotFoundError(absPath)
		}
		return errors.NewFileSystemOperationError("stat_symlink", absPath, err)
	}

	if linkInfo.Mode()&os.ModeSymlink == 0 {
		return errors.NewNotSymlinkError(absPath)
	}

	// Get symlink target
	target, err := s.fileManager.Readlink(ctx, absPath)
	if err != nil {
		return errors.NewFileSystemOperationError("read_symlink", absPath, err)
	}

	// Convert relative symlink target to absolute path
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(absPath), target)
	}

	// Validate that the target exists in our repository
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return errors.NewFileSystemOperationError("resolve_target", target, err)
	}

	repoPathAbs, err := filepath.Abs(s.repoPath)
	if err != nil {
		return errors.NewFileSystemOperationError("resolve_repo_path", s.repoPath, err)
	}

	if !strings.HasPrefix(targetAbs, repoPathAbs) {
		return errors.NewInvalidPathError(targetAbs, "symlink target is not in lnk repository")
	}

	// Get relative path for tracking
	relativePath, err := s.pathResolver.GetRelativePathFromHome(absPath)
	if err != nil {
		return errors.NewFileSystemOperationError("get_relative_path", absPath, err)
	}

	// Check if this file is actually managed
	managedFile, err := s.configManager.GetManagedFile(ctx, s.repoPath, host, relativePath)
	if err != nil || managedFile == nil {
		return errors.NewLnkError(errors.ErrorCodeFileNotFound, fmt.Sprintf("file is not managed by lnk: %s", relativePath))
	}

	// Get target file info to determine if it's a directory
	targetInfo, err := s.fileManager.Stat(ctx, targetAbs)
	if err != nil {
		return errors.NewFileSystemOperationError("stat_target", targetAbs, err)
	}

	// Execute the file removal with rollback support
	return s.executeFileRemoval(ctx, absPath, targetAbs, relativePath, host, targetInfo.IsDir())
}

// executeFileRemoval performs the actual file removal with rollback logic
func (s *Service) executeFileRemoval(ctx context.Context, symlinkPath, targetPath, relativePath, host string, isDirectory bool) error {
	var rollbackActions []func() error

	// Helper function to add rollback action
	addRollback := func(action func() error) {
		rollbackActions = append([]func() error{action}, rollbackActions...)
	}

	// Execute rollback if any step fails
	defer func() {
		if len(rollbackActions) > 0 {
			for _, action := range rollbackActions {
				_ = action() // Ignore rollback errors
			}
		}
	}()

	// Step 1: Remove the symlink
	if err := s.fileManager.Remove(ctx, symlinkPath); err != nil {
		return errors.NewFileSystemOperationError("remove_symlink", symlinkPath, err)
	}

	// Add rollback for symlink removal
	addRollback(func() error {
		return s.fileManager.CreateSymlink(context.Background(), targetPath, symlinkPath)
	})

	// Step 2: Move file back from repository to original location
	if err := s.fileManager.Move(ctx, targetPath, symlinkPath); err != nil {
		return errors.NewFileSystemOperationError("restore_file", targetPath, err)
	}

	// Add rollback for file restoration
	addRollback(func() error {
		return s.fileManager.Move(context.Background(), symlinkPath, targetPath)
	})

	// Step 3: Remove from config tracking
	if err := s.configManager.RemoveManagedFileFromHost(ctx, s.repoPath, host, relativePath); err != nil {
		return err // ConfigManager returns properly typed errors
	}

	// Add rollback for config update
	managedFile := models.ManagedFile{
		OriginalPath: symlinkPath,
		RelativePath: relativePath,
		RepoPath:     targetPath,
		Host:         host,
		IsDirectory:  isDirectory,
	}
	addRollback(func() error {
		return s.configManager.AddManagedFileToHost(context.Background(), s.repoPath, host, managedFile)
	})

	// Step 4: Remove file from Git
	gitPath := relativePath
	if host != "" {
		gitPath = filepath.Join(host+".lnk", relativePath)
	}

	if err := s.gitManager.Remove(ctx, s.repoPath, gitPath); err != nil {
		return err
	}

	// Step 5: Add config file to Git (to commit the tracking change)
	trackingFile, err := s.pathResolver.GetTrackingFilePath(s.repoPath, host)
	if err != nil {
		return errors.NewFileSystemOperationError("get_tracking_file", "", err)
	}

	// Get relative path of tracking file from repo root
	trackingFileRel, err := filepath.Rel(s.repoPath, trackingFile)
	if err != nil {
		return errors.NewFileSystemOperationError("get_tracking_file_rel", trackingFile, err)
	}

	if err := s.gitManager.Add(ctx, s.repoPath, trackingFileRel); err != nil {
		return errors.NewGitOperationError("add_tracking_file_to_git", err)
	}

	// Step 6: Commit changes
	basename := filepath.Base(relativePath)
	commitMessage := fmt.Sprintf("lnk: removed %s", basename)
	if err := s.gitManager.Commit(ctx, s.repoPath, commitMessage); err != nil {
		return errors.NewGitOperationError("commit_changes", err)
	}

	// If we reach here, everything succeeded - clear rollback actions
	rollbackActions = nil
	return nil
}

// PushChanges stages all changes and pushes to remote repository
func (s *Service) PushChanges(ctx context.Context, message string) error {
	// Check if GitManager is available
	if s.gitManager == nil {
		return errors.NewGitOperationError("push_changes",
			fmt.Errorf("git manager not available"))
	}

	// Check if repository is initialized
	initialized, err := s.IsRepositoryInitialized(ctx)
	if err != nil {
		return err
	}
	if !initialized {
		return errors.NewRepoNotInitializedError(s.repoPath)
	}

	// Check if there are any changes to commit
	hasChanges, err := s.gitManager.HasChanges(ctx, s.repoPath)
	if err != nil {
		return errors.NewGitOperationError("check_changes", err)
	}

	if hasChanges {
		// Add all changes (equivalent to git add .)
		if err := s.gitManager.Add(ctx, s.repoPath, "."); err != nil {
			return errors.NewGitOperationError("stage_changes", err)
		}

		// Create a sync commit
		if err := s.gitManager.Commit(ctx, s.repoPath, message); err != nil {
			return errors.NewGitOperationError("commit_changes", err)
		}
	}

	// Push to remote
	if err := s.gitManager.Push(ctx, s.repoPath); err != nil {
		return errors.NewGitOperationError("push_to_remote", err)
	}

	return nil
}

// PullChanges pulls changes from remote and restores symlinks for the specified host
func (s *Service) PullChanges(ctx context.Context, host string) ([]models.ManagedFile, error) {
	// Check if GitManager is available
	if s.gitManager == nil {
		return nil, errors.NewGitOperationError("pull_changes",
			fmt.Errorf("git manager not available"))
	}

	// Check if repository is initialized
	initialized, err := s.IsRepositoryInitialized(ctx)
	if err != nil {
		return nil, err
	}
	if !initialized {
		return nil, errors.NewRepoNotInitializedError(s.repoPath)
	}

	// Pull changes from remote
	if err := s.gitManager.Pull(ctx, s.repoPath); err != nil {
		return nil, errors.NewGitOperationError("pull_from_remote", err)
	}

	// Restore symlinks for the specified host
	restored, err := s.RestoreSymlinksForHost(ctx, host)
	if err != nil {
		return nil, err
	}

	return restored, nil
}

// RestoreSymlinksForHost restores symlinks for all managed files for the specified host
func (s *Service) RestoreSymlinksForHost(ctx context.Context, host string) ([]models.ManagedFile, error) {
	// Check if repository is initialized
	initialized, err := s.IsRepositoryInitialized(ctx)
	if err != nil {
		return nil, err
	}
	if !initialized {
		return nil, errors.NewRepoNotInitializedError(s.repoPath)
	}

	// Get list of managed files for this host
	managedFiles, err := s.configManager.ListManagedFiles(ctx, s.repoPath, host)
	if err != nil {
		return nil, err
	}

	var restored []models.ManagedFile
	homeDir, err := s.pathResolver.GetHomePath()
	if err != nil {
		return nil, errors.NewFileSystemOperationError("get_home_dir", "", err)
	}

	for _, managedFile := range managedFiles {
		// Determine symlink path (where the symlink should be created)
		symlinkPath := filepath.Join(homeDir, managedFile.RelativePath)

		// Determine repository file path (what the symlink should point to)
		repoFilePath, err := s.pathResolver.GetFileStoragePathInRepo(s.repoPath, host, managedFile.RelativePath)
		if err != nil {
			continue // Skip files with path resolution issues
		}

		// Check if repository file exists
		repoExists, err := s.fileManager.Exists(ctx, repoFilePath)
		if err != nil || !repoExists {
			continue // Skip missing files
		}

		// Check if symlink already exists and is correct
		if s.isValidSymlink(ctx, symlinkPath, repoFilePath) {
			continue // Skip files that are already correctly symlinked
		}

		// Ensure parent directory exists
		symlinkDir := filepath.Dir(symlinkPath)
		if err := s.fileManager.MkdirAll(ctx, symlinkDir, 0755); err != nil {
			continue // Skip files with directory creation issues
		}

		// Remove existing file/symlink if it exists
		exists, err := s.fileManager.Exists(ctx, symlinkPath)
		if err == nil && exists {
			if err := s.fileManager.Remove(ctx, symlinkPath); err != nil {
				continue // Skip files that can't be removed
			}
		}

		// Create symlink
		if err := s.fileManager.CreateSymlink(ctx, repoFilePath, symlinkPath); err != nil {
			continue // Skip files with symlink creation issues
		}

		// Update the managed file with current paths
		restoredFile := managedFile
		restoredFile.OriginalPath = symlinkPath
		restoredFile.RepoPath = repoFilePath
		restored = append(restored, restoredFile)
	}

	return restored, nil
}

// isValidSymlink checks if the given path is a symlink pointing to the expected target
func (s *Service) isValidSymlink(ctx context.Context, symlinkPath, expectedTarget string) bool {
	info, err := s.fileManager.Lstat(ctx, symlinkPath)
	if err != nil {
		return false
	}

	// Check if it's a symlink
	if info.Mode()&os.ModeSymlink == 0 {
		return false
	}

	// Check if it points to the correct target
	target, err := s.fileManager.Readlink(ctx, symlinkPath)
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
