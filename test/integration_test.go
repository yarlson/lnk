package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// Verify the default branch is set to 'main'
	cmd := exec.Command("git", "symbolic-ref", "HEAD")
	cmd.Dir = lnkDir
	output, err := cmd.Output()
	suite.Require().NoError(err)
	suite.Equal("refs/heads/main", strings.TrimSpace(string(output)))
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

func (suite *LnkIntegrationTestSuite) TestInitWithRemote() {
	// Test that init with remote adds the origin remote
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	remoteURL := "https://github.com/user/dotfiles.git"
	err = suite.lnk.AddRemote("origin", remoteURL)
	suite.Require().NoError(err)

	// Verify the remote was added by checking git config
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = lnkDir

	output, err := cmd.Output()
	suite.Require().NoError(err)
	suite.Equal(remoteURL, strings.TrimSpace(string(output)))
}

func (suite *LnkIntegrationTestSuite) TestInitIdempotent() {
	// Test that running init multiple times is safe
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	lnkDir := filepath.Join(suite.tempDir, "lnk")

	// Add a file to the repo to ensure it's not lost
	testFile := filepath.Join(lnkDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	suite.Require().NoError(err)

	// Run init again - should be idempotent
	err = suite.lnk.Init()
	suite.Require().NoError(err)

	// File should still exist
	suite.FileExists(testFile)
	content, err := os.ReadFile(testFile)
	suite.Require().NoError(err)
	suite.Equal("test content", string(content))
}

func (suite *LnkIntegrationTestSuite) TestInitWithExistingRemote() {
	// Test init with remote when remote already exists (same URL)
	remoteURL := "https://github.com/user/dotfiles.git"

	// First init with remote
	err := suite.lnk.Init()
	suite.Require().NoError(err)
	err = suite.lnk.AddRemote("origin", remoteURL)
	suite.Require().NoError(err)

	// Init again with same remote should be idempotent
	err = suite.lnk.Init()
	suite.Require().NoError(err)
	err = suite.lnk.AddRemote("origin", remoteURL)
	suite.Require().NoError(err)

	// Verify remote is still correct
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = lnkDir
	output, err := cmd.Output()
	suite.Require().NoError(err)
	suite.Equal(remoteURL, strings.TrimSpace(string(output)))
}

func (suite *LnkIntegrationTestSuite) TestInitWithDifferentRemote() {
	// Test init with different remote when remote already exists
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add first remote
	err = suite.lnk.AddRemote("origin", "https://github.com/user/dotfiles.git")
	suite.Require().NoError(err)

	// Try to add different remote - should error
	err = suite.lnk.AddRemote("origin", "https://github.com/user/other-repo.git")
	suite.Error(err)
	suite.Contains(err.Error(), "already exists with different URL")
}

func (suite *LnkIntegrationTestSuite) TestInitWithNonLnkRepo() {
	// Test init when directory contains a non-lnk Git repository
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	err := os.MkdirAll(lnkDir, 0755)
	suite.Require().NoError(err)

	// Create a non-lnk git repo in the lnk directory
	cmd := exec.Command("git", "init")
	cmd.Dir = lnkDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// Add some content to make it look like a real repo
	testFile := filepath.Join(lnkDir, "important-file.txt")
	err = os.WriteFile(testFile, []byte("important data"), 0644)
	suite.Require().NoError(err)

	// Configure git and commit
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = lnkDir
	err = cmd.Run()
	suite.Require().NoError(err)

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = lnkDir
	err = cmd.Run()
	suite.Require().NoError(err)

	cmd = exec.Command("git", "add", "important-file.txt")
	cmd.Dir = lnkDir
	err = cmd.Run()
	suite.Require().NoError(err)

	cmd = exec.Command("git", "commit", "-m", "important commit")
	cmd.Dir = lnkDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// Now try to init lnk - should error to protect existing repo
	err = suite.lnk.Init()
	suite.Error(err)
	suite.Contains(err.Error(), "appears to contain an existing Git repository")

	// Verify the original file is still there
	suite.FileExists(testFile)
}

func TestLnkIntegrationSuite(t *testing.T) {
	suite.Run(t, new(LnkIntegrationTestSuite))
}
