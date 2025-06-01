package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/yarlson/lnk/internal/errors"
	"github.com/yarlson/lnk/internal/fs"
	"github.com/yarlson/lnk/internal/models"
	"github.com/yarlson/lnk/internal/pathresolver"
)

type ConfigTestSuite struct {
	suite.Suite
	tempDir       string
	configManager *Config
	fileManager   *fs.FileManager
	pathResolver  *pathresolver.Resolver
	ctx           context.Context
}

func (suite *ConfigTestSuite) SetupTest() {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "lnk_config_test_*")
	suite.Require().NoError(err)
	suite.tempDir = tempDir

	// Create file manager and path resolver
	suite.fileManager = fs.New()
	suite.pathResolver = pathresolver.New()

	// Create config manager
	suite.configManager = New(suite.fileManager, suite.pathResolver)

	// Create context
	suite.ctx = context.Background()
}

func (suite *ConfigTestSuite) TearDownTest() {
	err := os.RemoveAll(suite.tempDir)
	suite.Require().NoError(err)
}

func (suite *ConfigTestSuite) TestAddAndListManagedFiles() {
	repoPath := filepath.Join(suite.tempDir, "repo")
	host := "testhost"

	// Create a managed file
	managedFile := models.ManagedFile{
		RelativePath: ".vimrc",
		Host:         host,
		IsDirectory:  false,
	}

	// Add managed file
	err := suite.configManager.AddManagedFileToHost(suite.ctx, repoPath, host, managedFile)
	suite.NoError(err)

	// List managed files
	files, err := suite.configManager.ListManagedFiles(suite.ctx, repoPath, host)
	suite.NoError(err)

	suite.Len(files, 1)
	suite.Equal(".vimrc", files[0].RelativePath)
	suite.Equal(host, files[0].Host)

	// Verify tracking file was created
	trackingPath, err := suite.pathResolver.GetTrackingFilePath(repoPath, host)
	suite.NoError(err)

	exists, err := suite.fileManager.Exists(suite.ctx, trackingPath)
	suite.NoError(err)
	suite.True(exists)

	// Read tracking file content
	content, err := suite.fileManager.ReadFile(suite.ctx, trackingPath)
	suite.NoError(err)

	expectedContent := ".vimrc\n"
	suite.Equal(expectedContent, string(content))
}

func (suite *ConfigTestSuite) TestAddDuplicateFile() {
	repoPath := filepath.Join(suite.tempDir, "repo")
	host := "testhost"

	managedFile := models.ManagedFile{
		RelativePath: ".bashrc",
		Host:         host,
	}

	// Add file twice
	err := suite.configManager.AddManagedFileToHost(suite.ctx, repoPath, host, managedFile)
	suite.NoError(err)

	err = suite.configManager.AddManagedFileToHost(suite.ctx, repoPath, host, managedFile)
	suite.NoError(err)

	// Should still have only one file
	files, err := suite.configManager.ListManagedFiles(suite.ctx, repoPath, host)
	suite.NoError(err)
	suite.Len(files, 1)
}

func (suite *ConfigTestSuite) TestRemoveManagedFile() {
	repoPath := filepath.Join(suite.tempDir, "repo")
	host := "testhost"

	// Add two managed files
	file1 := models.ManagedFile{RelativePath: ".vimrc", Host: host}
	file2 := models.ManagedFile{RelativePath: ".bashrc", Host: host}

	err := suite.configManager.AddManagedFileToHost(suite.ctx, repoPath, host, file1)
	suite.NoError(err)

	err = suite.configManager.AddManagedFileToHost(suite.ctx, repoPath, host, file2)
	suite.NoError(err)

	// Remove one file
	err = suite.configManager.RemoveManagedFileFromHost(suite.ctx, repoPath, host, ".vimrc")
	suite.NoError(err)

	// Should have only one file left
	files, err := suite.configManager.ListManagedFiles(suite.ctx, repoPath, host)
	suite.NoError(err)

	suite.Len(files, 1)
	suite.Equal(".bashrc", files[0].RelativePath)
}

func (suite *ConfigTestSuite) TestLoadAndSaveHostConfig() {
	repoPath := filepath.Join(suite.tempDir, "repo")
	host := "workstation"

	// Create host config with managed files
	config := &models.HostConfig{
		Name: host,
		ManagedFiles: []models.ManagedFile{
			{RelativePath: ".vimrc", Host: host},
			{RelativePath: ".bashrc", Host: host},
		},
		LastUpdate: time.Now(),
	}

	// Save config
	err := suite.configManager.SaveHostConfig(suite.ctx, repoPath, config)
	suite.NoError(err)

	// Load config
	loadedConfig, err := suite.configManager.LoadHostConfig(suite.ctx, repoPath, host)
	suite.NoError(err)

	suite.Equal(host, loadedConfig.Name)
	suite.Len(loadedConfig.ManagedFiles, 2)

	// Check files are sorted
	suite.Equal(".bashrc", loadedConfig.ManagedFiles[0].RelativePath)
	suite.Equal(".vimrc", loadedConfig.ManagedFiles[1].RelativePath)
}

func (suite *ConfigTestSuite) TestGetManagedFile() {
	repoPath := filepath.Join(suite.tempDir, "repo")
	host := "testhost"

	managedFile := models.ManagedFile{
		RelativePath: ".gitconfig",
		Host:         host,
	}

	// Add managed file
	err := suite.configManager.AddManagedFileToHost(suite.ctx, repoPath, host, managedFile)
	suite.NoError(err)

	// Get specific managed file
	file, err := suite.configManager.GetManagedFile(suite.ctx, repoPath, host, ".gitconfig")
	suite.NoError(err)
	suite.Equal(".gitconfig", file.RelativePath)

	// Try to get non-existent file
	_, err = suite.configManager.GetManagedFile(suite.ctx, repoPath, host, ".nonexistent")
	suite.Error(err)
	suite.True(errors.NewFileNotFoundError("").Is(err))
}

func (suite *ConfigTestSuite) TestConfigExists() {
	repoPath := filepath.Join(suite.tempDir, "repo")
	host := "testhost"

	// Initially should not exist
	exists, err := suite.configManager.ConfigExists(suite.ctx, repoPath, host)
	suite.NoError(err)
	suite.False(exists)

	// Add a managed file
	managedFile := models.ManagedFile{RelativePath: ".vimrc", Host: host}
	err = suite.configManager.AddManagedFileToHost(suite.ctx, repoPath, host, managedFile)
	suite.NoError(err)

	// Now should exist
	exists, err = suite.configManager.ConfigExists(suite.ctx, repoPath, host)
	suite.NoError(err)
	suite.True(exists)
}

func (suite *ConfigTestSuite) TestEmptyConfig() {
	repoPath := filepath.Join(suite.tempDir, "repo")
	host := "emptyhost"

	// List files from non-existent config
	files, err := suite.configManager.ListManagedFiles(suite.ctx, repoPath, host)
	suite.NoError(err)
	suite.Len(files, 0)
}

func (suite *ConfigTestSuite) TestCommonAndHostConfigs() {
	repoPath := filepath.Join(suite.tempDir, "repo")

	// Add file to common config (empty host)
	commonFile := models.ManagedFile{RelativePath: ".bashrc", Host: ""}
	err := suite.configManager.AddManagedFileToHost(suite.ctx, repoPath, "", commonFile)
	suite.NoError(err)

	// Add file to host-specific config
	hostFile := models.ManagedFile{RelativePath: ".vimrc", Host: "workstation"}
	err = suite.configManager.AddManagedFileToHost(suite.ctx, repoPath, "workstation", hostFile)
	suite.NoError(err)

	// List common files
	commonFiles, err := suite.configManager.ListManagedFiles(suite.ctx, repoPath, "")
	suite.NoError(err)
	suite.Len(commonFiles, 1)
	suite.Equal(".bashrc", commonFiles[0].RelativePath)

	// List host files
	hostFiles, err := suite.configManager.ListManagedFiles(suite.ctx, repoPath, "workstation")
	suite.NoError(err)
	suite.Len(hostFiles, 1)
	suite.Equal(".vimrc", hostFiles[0].RelativePath)
}

func (suite *ConfigTestSuite) TestFileWithMetadata() {
	repoPath := filepath.Join(suite.tempDir, "repo")
	host := "testhost"

	// Create actual file in repository storage area
	hostStoragePath := filepath.Join(repoPath, host+".lnk")
	testFilePath := filepath.Join(hostStoragePath, ".vimrc")

	err := suite.fileManager.WriteFile(suite.ctx, testFilePath, []byte("set number"), 0644)
	suite.NoError(err)

	// Add managed file
	managedFile := models.ManagedFile{RelativePath: ".vimrc", Host: host}
	err = suite.configManager.AddManagedFileToHost(suite.ctx, repoPath, host, managedFile)
	suite.NoError(err)

	// List files should include metadata
	files, err := suite.configManager.ListManagedFiles(suite.ctx, repoPath, host)
	suite.NoError(err)
	suite.Len(files, 1)

	file := files[0]
	suite.False(file.IsDirectory)
	suite.NotZero(file.Mode)

	// Expected paths
	expectedRepoPath := testFilePath
	suite.Equal(expectedRepoPath, file.RepoPath)
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
