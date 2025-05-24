package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/yarlson/lnk/internal/core"
)

type LnkIntegrationTestSuite struct {
	suite.Suite
	tempDir     string
	originalDir string
	lnk         *core.Lnk
}

func (suite *LnkIntegrationTestSuite) SetupTest() {
	// Create temporary directory for each test
	tempDir, err := os.MkdirTemp("", "lnk-test-*")
	suite.Require().NoError(err)
	suite.tempDir = tempDir

	// Change to temp directory
	originalDir, err := os.Getwd()
	suite.Require().NoError(err)
	suite.originalDir = originalDir

	err = os.Chdir(tempDir)
	suite.Require().NoError(err)

	// Set XDG_CONFIG_HOME to temp directory
	suite.T().Setenv("XDG_CONFIG_HOME", tempDir)

	// Initialize Lnk instance
	suite.lnk = core.NewLnk()
}

func (suite *LnkIntegrationTestSuite) TearDownTest() {
	// Return to original directory
	err := os.Chdir(suite.originalDir)
	suite.Require().NoError(err)

	// Clean up temp directory
	err = os.RemoveAll(suite.tempDir)
	suite.Require().NoError(err)
}

func (suite *LnkIntegrationTestSuite) TestInit() {
	// Test that init creates the directory and Git repo
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Check that the lnk directory was created
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	suite.DirExists(lnkDir)

	// Check that Git repo was initialized
	gitDir := filepath.Join(lnkDir, ".git")
	suite.DirExists(gitDir)

	// Verify it's a non-bare repo
	configPath := filepath.Join(gitDir, "config")
	suite.FileExists(configPath)
}

func (suite *LnkIntegrationTestSuite) TestAddFile() {
	// Initialize first
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create a test file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	content := "export PATH=$PATH:/usr/local/bin"
	err = os.WriteFile(testFile, []byte(content), 0644)
	suite.Require().NoError(err)

	// Add the file
	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Check that the original file is now a symlink
	info, err := os.Lstat(testFile)
	suite.Require().NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)

	// Check that the file exists in the repo
	repoFile := filepath.Join(suite.tempDir, "lnk", ".bashrc")
	suite.FileExists(repoFile)

	// Check that the content is preserved
	repoContent, err := os.ReadFile(repoFile)
	suite.Require().NoError(err)
	suite.Equal(content, string(repoContent))

	// Check that symlink points to the correct location
	linkTarget, err := os.Readlink(testFile)
	suite.Require().NoError(err)
	expectedTarget, err := filepath.Rel(filepath.Dir(testFile), repoFile)
	suite.Require().NoError(err)
	suite.Equal(expectedTarget, linkTarget)

	// Check that Git commit was made
	commits, err := suite.lnk.GetCommits()
	suite.Require().NoError(err)
	suite.Len(commits, 1)
	suite.Contains(commits[0], "lnk: added .bashrc")
}

func (suite *LnkIntegrationTestSuite) TestRemoveFile() {
	// Initialize and add a file first
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	testFile := filepath.Join(suite.tempDir, ".vimrc")
	content := "set number"
	err = os.WriteFile(testFile, []byte(content), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Now remove the file
	err = suite.lnk.Remove(testFile)
	suite.Require().NoError(err)

	// Check that the symlink is gone and regular file is restored
	info, err := os.Lstat(testFile)
	suite.Require().NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink) // Not a symlink

	// Check that content is preserved
	restoredContent, err := os.ReadFile(testFile)
	suite.Require().NoError(err)
	suite.Equal(content, string(restoredContent))

	// Check that file is removed from repo
	repoFile := filepath.Join(suite.tempDir, "lnk", ".vimrc")
	suite.NoFileExists(repoFile)

	// Check that Git commit was made
	commits, err := suite.lnk.GetCommits()
	suite.Require().NoError(err)
	suite.Len(commits, 2) // add + remove
	suite.Contains(commits[0], "lnk: removed .vimrc")
	suite.Contains(commits[1], "lnk: added .vimrc")
}

func (suite *LnkIntegrationTestSuite) TestAddNonexistentFile() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	err = suite.lnk.Add("/nonexistent/file")
	suite.Error(err)
	suite.Contains(err.Error(), "file does not exist")
}

func (suite *LnkIntegrationTestSuite) TestAddDirectory() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create a directory
	testDir := filepath.Join(suite.tempDir, "testdir")
	err = os.MkdirAll(testDir, 0755)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testDir)
	suite.Error(err)
	suite.Contains(err.Error(), "directories are not supported")
}

func (suite *LnkIntegrationTestSuite) TestRemoveNonSymlink() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create a regular file (not managed by lnk)
	testFile := filepath.Join(suite.tempDir, ".regularfile")
	err = os.WriteFile(testFile, []byte("content"), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Remove(testFile)
	suite.Error(err)
	suite.Contains(err.Error(), "file is not managed by lnk")
}

func (suite *LnkIntegrationTestSuite) TestXDGConfigHomeFallback() {
	// Test fallback to ~/.config/lnk when XDG_CONFIG_HOME is not set
	suite.T().Setenv("XDG_CONFIG_HOME", "")

	homeDir := filepath.Join(suite.tempDir, "home")
	err := os.MkdirAll(homeDir, 0755)
	suite.Require().NoError(err)
	suite.T().Setenv("HOME", homeDir)

	lnk := core.NewLnk()
	err = lnk.Init()
	suite.Require().NoError(err)

	// Check that the lnk directory was created under ~/.config/lnk
	expectedDir := filepath.Join(homeDir, ".config", "lnk")
	suite.DirExists(expectedDir)
}

func TestLnkIntegrationSuite(t *testing.T) {
	suite.Run(t, new(LnkIntegrationTestSuite))
}
