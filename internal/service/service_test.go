package service

import (
	"context"
	stderrors "errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/yarlson/lnk/internal/errors"
	"github.com/yarlson/lnk/internal/models"
)

// Mock implementations for testing

type mockFileManager struct {
	existsFunc        func(ctx context.Context, path string) (bool, error)
	isDirectoryFunc   func(ctx context.Context, path string) (bool, error)
	moveFunc          func(ctx context.Context, src, dst string) error
	removeFunc        func(ctx context.Context, path string) error
	statFunc          func(ctx context.Context, path string) (os.FileInfo, error)
	createSymlinkFunc func(ctx context.Context, target, linkPath string) error
	mkdirAllFunc      func(ctx context.Context, path string, perm os.FileMode) error
	readlinkFunc      func(ctx context.Context, path string) (string, error)
	lstatFunc         func(ctx context.Context, path string) (os.FileInfo, error)
}

func (m *mockFileManager) Exists(ctx context.Context, path string) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, path)
	}
	return false, nil
}

func (m *mockFileManager) IsDirectory(ctx context.Context, path string) (bool, error) {
	if m.isDirectoryFunc != nil {
		return m.isDirectoryFunc(ctx, path)
	}
	return false, nil
}

func (m *mockFileManager) Move(ctx context.Context, src, dst string) error {
	if m.moveFunc != nil {
		return m.moveFunc(ctx, src, dst)
	}
	return nil
}

func (m *mockFileManager) CreateSymlink(ctx context.Context, target, linkPath string) error {
	if m.createSymlinkFunc != nil {
		return m.createSymlinkFunc(ctx, target, linkPath)
	}
	return nil
}

func (m *mockFileManager) Remove(ctx context.Context, path string) error {
	if m.removeFunc != nil {
		return m.removeFunc(ctx, path)
	}
	return nil
}

func (m *mockFileManager) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return nil, nil
}

func (m *mockFileManager) WriteFile(ctx context.Context, path string, data []byte, perm os.FileMode) error {
	return nil
}

func (m *mockFileManager) MkdirAll(ctx context.Context, path string, perm os.FileMode) error {
	if m.mkdirAllFunc != nil {
		return m.mkdirAllFunc(ctx, path, perm)
	}
	return nil
}

func (m *mockFileManager) Readlink(ctx context.Context, path string) (string, error) {
	if m.readlinkFunc != nil {
		return m.readlinkFunc(ctx, path)
	}
	return "", nil
}

func (m *mockFileManager) Lstat(ctx context.Context, path string) (os.FileInfo, error) {
	if m.lstatFunc != nil {
		return m.lstatFunc(ctx, path)
	}
	return nil, nil
}

func (m *mockFileManager) Stat(ctx context.Context, path string) (os.FileInfo, error) {
	if m.statFunc != nil {
		return m.statFunc(ctx, path)
	}
	return nil, nil
}

type mockGitManager struct {
	isRepositoryFunc    func(ctx context.Context, repoPath string) (bool, error)
	statusFunc          func(ctx context.Context, repoPath string) (*models.SyncStatus, error)
	isLnkRepositoryFunc func(ctx context.Context, repoPath string) (bool, error)
	initFunc            func(ctx context.Context, repoPath string) error
	cloneFunc           func(ctx context.Context, repoPath, url string) error
}

func (m *mockGitManager) Init(ctx context.Context, repoPath string) error {
	if m.initFunc != nil {
		return m.initFunc(ctx, repoPath)
	}
	return nil
}

func (m *mockGitManager) Clone(ctx context.Context, repoPath, url string) error {
	if m.cloneFunc != nil {
		return m.cloneFunc(ctx, repoPath, url)
	}
	return nil
}

func (m *mockGitManager) Add(ctx context.Context, repoPath string, files ...string) error {
	return nil
}

func (m *mockGitManager) Remove(ctx context.Context, repoPath string, files ...string) error {
	return nil
}

func (m *mockGitManager) Commit(ctx context.Context, repoPath, message string) error {
	return nil
}

func (m *mockGitManager) Push(ctx context.Context, repoPath string) error {
	return nil
}

func (m *mockGitManager) Pull(ctx context.Context, repoPath string) error {
	return nil
}

func (m *mockGitManager) Status(ctx context.Context, repoPath string) (*models.SyncStatus, error) {
	if m.statusFunc != nil {
		return m.statusFunc(ctx, repoPath)
	}
	return nil, nil
}

func (m *mockGitManager) IsRepository(ctx context.Context, repoPath string) (bool, error) {
	if m.isRepositoryFunc != nil {
		return m.isRepositoryFunc(ctx, repoPath)
	}
	return true, nil
}

func (m *mockGitManager) HasChanges(ctx context.Context, repoPath string) (bool, error) {
	return false, nil
}

func (m *mockGitManager) AddRemote(ctx context.Context, repoPath, name, url string) error {
	return nil
}

func (m *mockGitManager) GetRemoteURL(ctx context.Context, repoPath, name string) (string, error) {
	return "", nil
}

func (m *mockGitManager) IsLnkRepository(ctx context.Context, repoPath string) (bool, error) {
	if m.isLnkRepositoryFunc != nil {
		return m.isLnkRepositoryFunc(ctx, repoPath)
	}
	return true, nil
}

type mockConfigManager struct {
	listManagedFilesFunc          func(ctx context.Context, repoPath, host string) ([]models.ManagedFile, error)
	getManagedFileFunc            func(ctx context.Context, repoPath, host, relativePath string) (*models.ManagedFile, error)
	addManagedFileToHostFunc      func(ctx context.Context, repoPath, host string, file models.ManagedFile) error
	removeManagedFileFromHostFunc func(ctx context.Context, repoPath, host, relativePath string) error
}

func (m *mockConfigManager) LoadHostConfig(ctx context.Context, repoPath, host string) (*models.HostConfig, error) {
	return nil, nil
}

func (m *mockConfigManager) SaveHostConfig(ctx context.Context, repoPath string, config *models.HostConfig) error {
	return nil
}

func (m *mockConfigManager) AddManagedFileToHost(ctx context.Context, repoPath, host string, file models.ManagedFile) error {
	if m.addManagedFileToHostFunc != nil {
		return m.addManagedFileToHostFunc(ctx, repoPath, host, file)
	}
	return nil
}

func (m *mockConfigManager) RemoveManagedFileFromHost(ctx context.Context, repoPath, host, relativePath string) error {
	if m.removeManagedFileFromHostFunc != nil {
		return m.removeManagedFileFromHostFunc(ctx, repoPath, host, relativePath)
	}
	return nil
}

func (m *mockConfigManager) ListManagedFiles(ctx context.Context, repoPath, host string) ([]models.ManagedFile, error) {
	if m.listManagedFilesFunc != nil {
		return m.listManagedFilesFunc(ctx, repoPath, host)
	}
	return []models.ManagedFile{}, nil
}

func (m *mockConfigManager) GetManagedFile(ctx context.Context, repoPath, host, relativePath string) (*models.ManagedFile, error) {
	if m.getManagedFileFunc != nil {
		return m.getManagedFileFunc(ctx, repoPath, host, relativePath)
	}
	return nil, nil
}

func (m *mockConfigManager) ConfigExists(ctx context.Context, repoPath, host string) (bool, error) {
	return true, nil
}

type mockPathResolver struct {
	getAbsolutePathInHomeFunc    func(path string) (string, error)
	getRelativePathFromHomeFunc  func(absPath string) (string, error)
	getFileStoragePathInRepoFunc func(repoPath, host, relativePath string) (string, error)
	getTrackingFilePathFunc      func(repoPath, host string) (string, error)
	getHomePathFunc              func() (string, error)
}

func (m *mockPathResolver) GetRepoStoragePath() (string, error) {
	return "/test/repo", nil
}

func (m *mockPathResolver) GetFileStoragePathInRepo(repoPath, host, relativePath string) (string, error) {
	if m.getFileStoragePathInRepoFunc != nil {
		return m.getFileStoragePathInRepoFunc(repoPath, host, relativePath)
	}
	return "/test/repo/file", nil
}

func (m *mockPathResolver) GetTrackingFilePath(repoPath, host string) (string, error) {
	if m.getTrackingFilePathFunc != nil {
		return m.getTrackingFilePathFunc(repoPath, host)
	}
	return "/test/repo/.lnk", nil
}

func (m *mockPathResolver) GetHomePath() (string, error) {
	if m.getHomePathFunc != nil {
		return m.getHomePathFunc()
	}
	return "/home/user", nil
}

func (m *mockPathResolver) GetRelativePathFromHome(absPath string) (string, error) {
	if m.getRelativePathFromHomeFunc != nil {
		return m.getRelativePathFromHomeFunc(absPath)
	}
	return ".bashrc", nil
}

func (m *mockPathResolver) GetAbsolutePathInHome(path string) (string, error) {
	if m.getAbsolutePathInHomeFunc != nil {
		return m.getAbsolutePathInHomeFunc(path)
	}
	return "/home/user/.bashrc", nil
}

func (m *mockPathResolver) GetHostStoragePath(repoPath, host string) (string, error) {
	if host == "" {
		return repoPath, nil
	}
	return repoPath + "/" + host + ".lnk", nil
}

func (m *mockPathResolver) IsUnderHome(path string) (bool, error) {
	return true, nil
}

// Helper mock for os.FileInfo
type mockFileInfo struct {
	isDir bool
	mode  os.FileMode
}

func (m *mockFileInfo) Name() string { return "test" }
func (m *mockFileInfo) Size() int64  { return 0 }
func (m *mockFileInfo) Mode() os.FileMode {
	if m.mode != 0 {
		return m.mode
	}
	return 0644
}
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// Test Suite

type LnkServiceTestSuite struct {
	suite.Suite
	ctx      context.Context
	repoPath string
	host     string
}

func (suite *LnkServiceTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.repoPath = "/test/repo"
	suite.host = "testhost"
}

func (suite *LnkServiceTestSuite) TestListManagedFilesSuccess() {
	expectedFiles := []models.ManagedFile{
		{
			RelativePath: ".vimrc",
			Host:         suite.host,
			IsDirectory:  false,
		},
		{
			RelativePath: ".bashrc",
			Host:         suite.host,
			IsDirectory:  false,
		},
	}

	// Setup mocks
	fileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			if path == suite.repoPath {
				return true, nil
			}
			return false, nil
		},
	}

	configManager := &mockConfigManager{
		listManagedFilesFunc: func(ctx context.Context, repoPath, host string) ([]models.ManagedFile, error) {
			return expectedFiles, nil
		},
	}

	service := NewLnkServiceWithDeps(fileManager, nil, configManager, &mockPathResolver{}, suite.repoPath)

	// Test
	result, err := service.ListManagedFiles(suite.ctx, suite.host)
	suite.NoError(err)
	suite.Len(result, len(expectedFiles))

	for i, expected := range expectedFiles {
		suite.Equal(expected.RelativePath, result[i].RelativePath)
	}
}

func (suite *LnkServiceTestSuite) TestListManagedFilesRepoNotExists() {
	// Setup mocks
	fileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return false, nil // Repository doesn't exist
		},
	}

	service := NewLnkServiceWithDeps(fileManager, nil, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	// Test
	_, err := service.ListManagedFiles(suite.ctx, suite.host)
	suite.Error(err)

	// Check that it's the correct error type
	suite.True(errors.NewRepoNotInitializedError("").Is(err))
}

func (suite *LnkServiceTestSuite) TestListManagedFilesFileSystemError() {
	expectedError := stderrors.New("fs error")

	// Setup mocks
	fileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return false, expectedError
		},
	}

	service := NewLnkServiceWithDeps(fileManager, nil, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	// Test
	_, err := service.ListManagedFiles(suite.ctx, suite.host)
	suite.Error(err)

	// Check that it's wrapped as a FileSystemOperation error
	suite.True(errors.NewFileSystemOperationError("", "", nil).Is(err))
}

func (suite *LnkServiceTestSuite) TestListManagedFilesConfigManagerError() {
	expectedError := errors.NewConfigNotFoundError(suite.host)

	// Setup mocks
	fileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return true, nil // Repository exists
		},
	}

	configManager := &mockConfigManager{
		listManagedFilesFunc: func(ctx context.Context, repoPath, host string) ([]models.ManagedFile, error) {
			return nil, expectedError
		},
	}

	service := NewLnkServiceWithDeps(fileManager, nil, configManager, &mockPathResolver{}, suite.repoPath)

	// Test
	_, err := service.ListManagedFiles(suite.ctx, suite.host)
	suite.Error(err)
	suite.Equal(expectedError, err)
}

func (suite *LnkServiceTestSuite) TestIsRepositoryInitializedWithGitManager() {
	// Setup mocks
	fileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return true, nil
		},
	}

	gitManager := &mockGitManager{
		isRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return true, nil
		},
	}

	service := NewLnkServiceWithDeps(fileManager, gitManager, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	// Test
	isInit, err := service.IsRepositoryInitialized(suite.ctx)
	suite.NoError(err)
	suite.True(isInit)
}

func (suite *LnkServiceTestSuite) TestIsRepositoryInitializedWithoutGitManager() {
	// Setup mocks
	fileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return true, nil
		},
	}

	service := NewLnkServiceWithDeps(fileManager, nil, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	// Test
	isInit, err := service.IsRepositoryInitialized(suite.ctx)
	suite.NoError(err)
	suite.True(isInit)
}

func (suite *LnkServiceTestSuite) TestIsRepositoryInitializedDirectoryNotExists() {
	// Setup mocks
	fileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	service := NewLnkServiceWithDeps(fileManager, nil, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	// Test
	isInit, err := service.IsRepositoryInitialized(suite.ctx)
	suite.NoError(err)
	suite.False(isInit)
}

func (suite *LnkServiceTestSuite) TestGetStatusSuccess() {
	mockFileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return true, nil // Repository exists
		},
	}
	mockGitManager := &mockGitManager{
		isRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return true, nil // Is a git repository
		},
		statusFunc: func(ctx context.Context, repoPath string) (*models.SyncStatus, error) {
			return &models.SyncStatus{
				CurrentBranch: "main",
				Dirty:         false,
				HasRemote:     true,
				RemoteURL:     "https://github.com/test/repo.git",
				Ahead:         2,
				Behind:        1,
			}, nil
		},
	}

	service := NewLnkServiceWithDeps(mockFileManager, mockGitManager, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	status, err := service.GetStatus(suite.ctx)
	suite.NoError(err)
	suite.Equal("main", status.CurrentBranch)
	suite.Equal(2, status.Ahead)
	suite.Equal(1, status.Behind)
}

func (suite *LnkServiceTestSuite) TestGetStatusGitManagerNotAvailable() {
	// Create service without GitManager (nil)
	service := NewLnkServiceWithDeps(&mockFileManager{}, nil, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	_, err := service.GetStatus(suite.ctx)
	suite.Error(err)

	var lnkErr *errors.LnkError
	suite.True(stderrors.As(err, &lnkErr))
	suite.Equal(errors.ErrorCodeGitOperation, lnkErr.Code)
}

func (suite *LnkServiceTestSuite) TestGetStatusRepoNotExists() {
	mockFileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	service := NewLnkServiceWithDeps(mockFileManager, &mockGitManager{}, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	_, err := service.GetStatus(suite.ctx)
	suite.Error(err)

	var lnkErr *errors.LnkError
	suite.True(stderrors.As(err, &lnkErr))
	suite.Equal(errors.ErrorCodeRepoNotInitialized, lnkErr.Code)
}

func (suite *LnkServiceTestSuite) TestGetStatusNotAGitRepository() {
	mockFileManager := &mockFileManager{}
	mockGitManager := &mockGitManager{
		isRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return false, nil
		},
	}

	service := NewLnkServiceWithDeps(mockFileManager, mockGitManager, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	_, err := service.GetStatus(suite.ctx)
	suite.Error(err)

	var lnkErr *errors.LnkError
	suite.True(stderrors.As(err, &lnkErr))
	suite.Equal(errors.ErrorCodeRepoNotInitialized, lnkErr.Code)
}

func (suite *LnkServiceTestSuite) TestGetStatusGitStatusError() {
	// Mock file manager - repo exists
	mockFS := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return true, nil
		},
	}

	// Mock git manager - is repository, but status fails
	mockGit := &mockGitManager{
		isRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return true, nil
		},
		statusFunc: func(ctx context.Context, repoPath string) (*models.SyncStatus, error) {
			return nil, errors.NewGitOperationError("status", stderrors.New("git status failed"))
		},
	}

	service := NewLnkServiceWithDeps(mockFS, mockGit, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	_, err := service.GetStatus(suite.ctx)
	suite.Error(err)

	// Should be a git operation error
	var lnkErr *errors.LnkError
	suite.True(stderrors.As(err, &lnkErr))
	suite.Equal(errors.ErrorCodeGitOperation, lnkErr.Code)
}

func (suite *LnkServiceTestSuite) TestInitializeRepositoryEmptySuccess() {
	// Mock file manager - directory doesn't exist initially
	mockFS := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return false, nil // Directory doesn't exist
		},
	}

	service := NewLnkServiceWithDeps(mockFS, &mockGitManager{}, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	err := service.InitializeRepository(suite.ctx, "")
	suite.NoError(err)
}

func (suite *LnkServiceTestSuite) TestInitializeRepositoryCloneSuccess() {
	service := NewLnkServiceWithDeps(&mockFileManager{}, &mockGitManager{}, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	err := service.InitializeRepository(suite.ctx, "https://github.com/user/dotfiles.git")
	suite.NoError(err)
}

func (suite *LnkServiceTestSuite) TestInitializeRepositoryGitManagerNotAvailable() {
	service := NewLnkServiceWithDeps(&mockFileManager{}, nil, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	err := service.InitializeRepository(suite.ctx, "")
	suite.Error(err)

	var lnkErr *errors.LnkError
	suite.True(stderrors.As(err, &lnkErr))
	suite.Equal(errors.ErrorCodeGitOperation, lnkErr.Code)
}

func (suite *LnkServiceTestSuite) TestInitializeRepositoryExistingLnkRepo() {
	// Mock file manager - directory exists
	mockFS := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return true, nil
		},
	}

	// Mock git manager - existing lnk repository
	mockGit := &mockGitManager{
		isRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return true, nil
		},
	}

	service := NewLnkServiceWithDeps(mockFS, mockGit, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	err := service.InitializeRepository(suite.ctx, "")
	suite.NoError(err)
}

func (suite *LnkServiceTestSuite) TestInitializeRepositoryExistingNonLnkRepo() {
	// Mock file manager - directory exists
	mockFS := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return true, nil
		},
	}

	// Mock git manager - existing non-lnk repository
	mockGit := &mockGitManager{
		isRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return true, nil
		},
		isLnkRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return false, nil
		},
	}

	service := NewLnkServiceWithDeps(mockFS, mockGit, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	err := service.InitializeRepository(suite.ctx, "")
	suite.Error(err)

	var lnkErr *errors.LnkError
	suite.True(stderrors.As(err, &lnkErr))
	suite.Equal(errors.ErrorCodeRepoNotInitialized, lnkErr.Code)
}

func (suite *LnkServiceTestSuite) TestInitializeRepositoryCloneError() {
	// Mock git manager with clone error
	mockGit := &mockGitManager{
		cloneFunc: func(ctx context.Context, repoPath, url string) error {
			return stderrors.New("clone failed")
		},
	}

	service := NewLnkServiceWithDeps(&mockFileManager{}, mockGit, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	err := service.InitializeRepository(suite.ctx, "https://github.com/user/dotfiles.git")
	suite.Error(err)

	var lnkErr *errors.LnkError
	suite.True(stderrors.As(err, &lnkErr))
	suite.Equal(errors.ErrorCodeGitOperation, lnkErr.Code)
}

func (suite *LnkServiceTestSuite) TestAddFileSuccess() {
	// Mock file info
	mockFileInfo := &mockFileInfo{isDir: false}

	// Mock file manager
	mockFS := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			if path == suite.repoPath {
				return true, nil // Repository exists
			}
			if path == "/home/user/.vimrc" {
				return true, nil // File exists
			}
			return false, nil
		},
		statFunc: func(ctx context.Context, path string) (os.FileInfo, error) {
			return mockFileInfo, nil
		},
	}

	// Mock git manager
	mockGit := &mockGitManager{
		isRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return true, nil
		},
	}

	// Mock config manager
	mockConfig := &mockConfigManager{
		getManagedFileFunc: func(ctx context.Context, repoPath, host, relativePath string) (*models.ManagedFile, error) {
			return nil, errors.NewFileNotFoundError("not managed") // File not managed yet
		},
	}

	// Mock path resolver
	mockPath := &mockPathResolver{
		getAbsolutePathInHomeFunc: func(path string) (string, error) {
			if path == ".vimrc" {
				return "/home/user/.vimrc", nil
			}
			return "", stderrors.New("path not found")
		},
		getRelativePathFromHomeFunc: func(absPath string) (string, error) {
			if absPath == "/home/user/.vimrc" {
				return ".vimrc", nil
			}
			return "", stderrors.New("path not under home")
		},
		getFileStoragePathInRepoFunc: func(repoPath, host, relativePath string) (string, error) {
			return "/test/repo/.vimrc", nil
		},
	}

	service := NewLnkServiceWithDeps(mockFS, mockGit, mockConfig, mockPath, suite.repoPath)

	managedFile, err := service.AddFile(suite.ctx, ".vimrc", "")
	suite.NoError(err)
	suite.NotNil(managedFile)
	suite.Equal(".vimrc", managedFile.RelativePath)
	suite.False(managedFile.IsDirectory)
}

func (suite *LnkServiceTestSuite) TestAddFileGitManagerNotAvailable() {
	service := NewLnkServiceWithDeps(&mockFileManager{}, nil, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	_, err := service.AddFile(suite.ctx, ".vimrc", "")
	suite.Error(err)

	var lnkErr *errors.LnkError
	suite.True(stderrors.As(err, &lnkErr))
	suite.Equal(errors.ErrorCodeGitOperation, lnkErr.Code)
}

func (suite *LnkServiceTestSuite) TestAddFileRepoNotInitialized() {
	mockFS := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			if path == "/home/user/.vimrc" {
				return true, nil // File exists
			}
			return false, nil // Repository doesn't exist
		},
	}

	// Mock path resolver
	mockPath := &mockPathResolver{
		getAbsolutePathInHomeFunc: func(path string) (string, error) {
			if path == ".vimrc" {
				return "/home/user/.vimrc", nil
			}
			return "", stderrors.New("path not found")
		},
	}

	service := NewLnkServiceWithDeps(mockFS, &mockGitManager{}, &mockConfigManager{}, mockPath, suite.repoPath)

	_, err := service.AddFile(suite.ctx, ".vimrc", "")
	suite.Error(err)

	var lnkErr *errors.LnkError
	suite.True(stderrors.As(err, &lnkErr))
	suite.Equal(errors.ErrorCodeRepoNotInitialized, lnkErr.Code)
}

func (suite *LnkServiceTestSuite) TestAddFileFileNotExists() {
	mockFS := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			if path == suite.repoPath {
				return true, nil // Repository exists
			}
			return false, nil // File doesn't exist
		},
	}

	mockGit := &mockGitManager{
		isRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return true, nil
		},
	}

	service := NewLnkServiceWithDeps(mockFS, mockGit, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	_, err := service.AddFile(suite.ctx, ".vimrc", "")
	suite.Error(err)

	var lnkErr *errors.LnkError
	suite.True(stderrors.As(err, &lnkErr))
	suite.Equal(errors.ErrorCodeFileNotFound, lnkErr.Code)
}

func (suite *LnkServiceTestSuite) TestAddFileFileAlreadyManaged() {
	mockFS := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return true, nil // Both repo and file exist
		},
		statFunc: func(ctx context.Context, path string) (os.FileInfo, error) {
			return &mockFileInfo{isDir: false}, nil
		},
	}

	mockGit := &mockGitManager{
		isRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return true, nil
		},
	}

	mockConfig := &mockConfigManager{
		getManagedFileFunc: func(ctx context.Context, repoPath, host, relativePath string) (*models.ManagedFile, error) {
			// Return existing managed file
			return &models.ManagedFile{RelativePath: relativePath}, nil
		},
	}

	service := NewLnkServiceWithDeps(mockFS, mockGit, mockConfig, &mockPathResolver{}, suite.repoPath)

	_, err := service.AddFile(suite.ctx, ".vimrc", "")
	suite.Error(err)

	var lnkErr *errors.LnkError
	suite.True(stderrors.As(err, &lnkErr))
	suite.Equal(errors.ErrorCodeFileAlreadyManaged, lnkErr.Code)
}

func (suite *LnkServiceTestSuite) TestRemoveFileSuccess() {
	fileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return true, nil
		},
		lstatFunc: func(ctx context.Context, path string) (os.FileInfo, error) {
			return &mockFileInfo{mode: os.ModeSymlink}, nil
		},
		readlinkFunc: func(ctx context.Context, path string) (string, error) {
			return "/test/repo/.bashrc", nil
		},
		statFunc: func(ctx context.Context, path string) (os.FileInfo, error) {
			return &mockFileInfo{mode: 0644}, nil
		},
	}

	gitManager := &mockGitManager{
		isRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return true, nil
		},
	}

	configManager := &mockConfigManager{
		getManagedFileFunc: func(ctx context.Context, repoPath, host, relativePath string) (*models.ManagedFile, error) {
			return &models.ManagedFile{
				RelativePath: ".bashrc",
				Host:         "",
			}, nil
		},
	}

	service := NewLnkServiceWithDeps(fileManager, gitManager, configManager, &mockPathResolver{}, suite.repoPath)

	err := service.RemoveFile(suite.ctx, ".bashrc", "")
	suite.NoError(err)
}

func (suite *LnkServiceTestSuite) TestRemoveFileGitManagerNotAvailable() {
	service := NewLnkServiceWithDeps(&mockFileManager{}, nil, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	err := service.RemoveFile(suite.ctx, ".bashrc", "")
	suite.Error(err)
	suite.Contains(err.Error(), "git manager not available")
}

func (suite *LnkServiceTestSuite) TestRemoveFileRepoNotInitialized() {
	fileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	service := NewLnkServiceWithDeps(fileManager, &mockGitManager{}, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	err := service.RemoveFile(suite.ctx, ".bashrc", "")
	suite.Error(err)
	suite.Contains(err.Error(), "repository not initialized")
}

func (suite *LnkServiceTestSuite) TestRemoveFileFileNotSymlink() {
	fileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return true, nil
		},
		lstatFunc: func(ctx context.Context, path string) (os.FileInfo, error) {
			return &mockFileInfo{mode: 0644}, nil // Not a symlink
		},
	}

	gitManager := &mockGitManager{
		isRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return true, nil
		},
	}

	service := NewLnkServiceWithDeps(fileManager, gitManager, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	err := service.RemoveFile(suite.ctx, ".bashrc", "")
	suite.Error(err)
	suite.Contains(err.Error(), "not a symlink")
}

func (suite *LnkServiceTestSuite) TestRemoveFileFileNotManaged() {
	fileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return true, nil
		},
		lstatFunc: func(ctx context.Context, path string) (os.FileInfo, error) {
			return &mockFileInfo{mode: os.ModeSymlink}, nil
		},
		readlinkFunc: func(ctx context.Context, path string) (string, error) {
			return "/test/repo/.bashrc", nil
		},
	}

	gitManager := &mockGitManager{
		isRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return true, nil
		},
	}

	configManager := &mockConfigManager{
		getManagedFileFunc: func(ctx context.Context, repoPath, host, relativePath string) (*models.ManagedFile, error) {
			return nil, nil // File not managed
		},
	}

	service := NewLnkServiceWithDeps(fileManager, gitManager, configManager, &mockPathResolver{}, suite.repoPath)

	err := service.RemoveFile(suite.ctx, ".bashrc", "")
	suite.Error(err)
	suite.Contains(err.Error(), "not managed by lnk")
}

func (suite *LnkServiceTestSuite) TestPushChangesSuccess() {
	fileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return true, nil
		},
	}

	gitManager := &mockGitManager{
		isRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return true, nil
		},
	}

	service := NewLnkServiceWithDeps(fileManager, gitManager, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	err := service.PushChanges(suite.ctx, "test commit")
	suite.NoError(err)
}

func (suite *LnkServiceTestSuite) TestPushChangesGitManagerNotAvailable() {
	service := NewLnkServiceWithDeps(&mockFileManager{}, nil, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	err := service.PushChanges(suite.ctx, "test commit")
	suite.Error(err)
	suite.Contains(err.Error(), "git manager not available")
}

func (suite *LnkServiceTestSuite) TestPullChangesSuccess() {
	fileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return true, nil
		},
	}

	gitManager := &mockGitManager{
		isRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return true, nil
		},
	}

	configManager := &mockConfigManager{
		listManagedFilesFunc: func(ctx context.Context, repoPath, host string) ([]models.ManagedFile, error) {
			return []models.ManagedFile{}, nil
		},
	}

	service := NewLnkServiceWithDeps(fileManager, gitManager, configManager, &mockPathResolver{}, suite.repoPath)

	restored, err := service.PullChanges(suite.ctx, "")
	suite.NoError(err)
	suite.Len(restored, 0)
}

func (suite *LnkServiceTestSuite) TestPullChangesGitManagerNotAvailable() {
	service := NewLnkServiceWithDeps(&mockFileManager{}, nil, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	_, err := service.PullChanges(suite.ctx, "")
	suite.Error(err)
	suite.Contains(err.Error(), "git manager not available")
}

func (suite *LnkServiceTestSuite) TestRestoreSymlinksForHostSuccess() {
	fileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			if path == suite.repoPath {
				return true, nil
			}
			if path == "/test/repo/.bashrc" {
				return true, nil // Repository file exists
			}
			return false, nil // Symlink doesn't exist yet
		},
		lstatFunc: func(ctx context.Context, path string) (os.FileInfo, error) {
			return nil, os.ErrNotExist // Symlink doesn't exist
		},
	}

	gitManager := &mockGitManager{
		isRepositoryFunc: func(ctx context.Context, repoPath string) (bool, error) {
			return true, nil
		},
	}

	configManager := &mockConfigManager{
		listManagedFilesFunc: func(ctx context.Context, repoPath, host string) ([]models.ManagedFile, error) {
			return []models.ManagedFile{
				{
					RelativePath: ".bashrc",
					Host:         "",
				},
			}, nil
		},
	}

	pathResolver := &mockPathResolver{
		getFileStoragePathInRepoFunc: func(repoPath, host, relativePath string) (string, error) {
			return "/test/repo/.bashrc", nil
		},
	}

	service := NewLnkServiceWithDeps(fileManager, gitManager, configManager, pathResolver, suite.repoPath)

	restored, err := service.RestoreSymlinksForHost(suite.ctx, "")
	suite.NoError(err)
	suite.Len(restored, 1)
	if len(restored) > 0 {
		suite.Equal(".bashrc", restored[0].RelativePath)
	}
}

func (suite *LnkServiceTestSuite) TestRestoreSymlinksForHostRepoNotInitialized() {
	fileManager := &mockFileManager{
		existsFunc: func(ctx context.Context, path string) (bool, error) {
			return false, nil
		},
	}

	service := NewLnkServiceWithDeps(fileManager, &mockGitManager{}, &mockConfigManager{}, &mockPathResolver{}, suite.repoPath)

	_, err := service.RestoreSymlinksForHost(suite.ctx, "")
	suite.Error(err)
	suite.Contains(err.Error(), "repository not initialized")
}

func TestLnkServiceSuite(t *testing.T) {
	suite.Run(t, new(LnkServiceTestSuite))
}
