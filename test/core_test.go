package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/yarlson/lnk/internal/core"
)

type CoreTestSuite struct {
	suite.Suite
	tempDir     string
	originalDir string
	lnk         *core.Lnk
}

func (suite *CoreTestSuite) SetupTest() {
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

func (suite *CoreTestSuite) TearDownTest() {
	// Return to original directory
	err := os.Chdir(suite.originalDir)
	suite.Require().NoError(err)

	// Clean up temp directory
	err = os.RemoveAll(suite.tempDir)
	suite.Require().NoError(err)
}

// Test core initialization functionality
func (suite *CoreTestSuite) TestCoreInit() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Check that the lnk directory was created
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	suite.DirExists(lnkDir)

	// Check that Git repo was initialized
	gitDir := filepath.Join(lnkDir, ".git")
	suite.DirExists(gitDir)
}

// Test core add/remove functionality with files
func (suite *CoreTestSuite) TestCoreFileOperations() {
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

	// Verify symlink and repo file
	info, err := os.Lstat(testFile)
	suite.Require().NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)

	repoFile := filepath.Join(suite.tempDir, "lnk", ".bashrc")
	suite.FileExists(repoFile)

	// Verify content is preserved
	repoContent, err := os.ReadFile(repoFile)
	suite.Require().NoError(err)
	suite.Equal(content, string(repoContent))

	// Test remove
	err = suite.lnk.Remove(testFile)
	suite.Require().NoError(err)

	// Verify symlink is gone and regular file is restored
	info, err = os.Lstat(testFile)
	suite.Require().NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink) // Not a symlink

	// Verify content is preserved
	restoredContent, err := os.ReadFile(testFile)
	suite.Require().NoError(err)
	suite.Equal(content, string(restoredContent))
}

// Test core add/remove functionality with directories
func (suite *CoreTestSuite) TestCoreDirectoryOperations() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create a directory with files
	testDir := filepath.Join(suite.tempDir, "testdir")
	err = os.MkdirAll(testDir, 0755)
	suite.Require().NoError(err)

	testFile := filepath.Join(testDir, "config.txt")
	content := "test config"
	err = os.WriteFile(testFile, []byte(content), 0644)
	suite.Require().NoError(err)

	// Add the directory
	err = suite.lnk.Add(testDir)
	suite.Require().NoError(err)

	// Verify directory is now a symlink
	info, err := os.Lstat(testDir)
	suite.Require().NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)

	// Verify directory exists in repo
	repoDir := filepath.Join(suite.tempDir, "lnk", "testdir")
	suite.DirExists(repoDir)

	// Remove the directory
	err = suite.lnk.Remove(testDir)
	suite.Require().NoError(err)

	// Verify symlink is gone and regular directory is restored
	info, err = os.Lstat(testDir)
	suite.Require().NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink) // Not a symlink
	suite.True(info.IsDir())                                // Is a directory

	// Verify content is preserved
	restoredContent, err := os.ReadFile(testFile)
	suite.Require().NoError(err)
	suite.Equal(content, string(restoredContent))
}

// Test .lnk file tracking functionality
func (suite *CoreTestSuite) TestLnkFileTracking() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add multiple items
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	testDir := filepath.Join(suite.tempDir, ".ssh")
	err = os.MkdirAll(testDir, 0700)
	suite.Require().NoError(err)
	configFile := filepath.Join(testDir, "config")
	err = os.WriteFile(configFile, []byte("Host example.com"), 0600)
	suite.Require().NoError(err)
	err = suite.lnk.Add(testDir)
	suite.Require().NoError(err)

	// Check .lnk file contains both entries
	lnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk")
	suite.FileExists(lnkFile)

	lnkContent, err := os.ReadFile(lnkFile)
	suite.Require().NoError(err)

	lines := strings.Split(strings.TrimSpace(string(lnkContent)), "\n")
	suite.Len(lines, 2)
	suite.Contains(lines, ".bashrc")
	suite.Contains(lines, ".ssh")

	// Remove one item and verify tracking is updated
	err = suite.lnk.Remove(testFile)
	suite.Require().NoError(err)

	lnkContent, err = os.ReadFile(lnkFile)
	suite.Require().NoError(err)

	lines = strings.Split(strings.TrimSpace(string(lnkContent)), "\n")
	suite.Len(lines, 1)
	suite.Contains(lines, ".ssh")
	suite.NotContains(lines, ".bashrc")
}

// Test XDG_CONFIG_HOME fallback
func (suite *CoreTestSuite) TestXDGConfigHomeFallback() {
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

// Test symlink restoration (pull functionality)
func (suite *CoreTestSuite) TestSymlinkRestoration() {
	_ = suite.lnk.Init()

	// Create a file in the repo directly (simulating a pull)
	repoFile := filepath.Join(suite.tempDir, "lnk", ".bashrc")
	content := "export PATH=$PATH:/usr/local/bin"
	err := os.WriteFile(repoFile, []byte(content), 0644)
	suite.Require().NoError(err)

	// Create .lnk file to track it
	lnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk")
	err = os.WriteFile(lnkFile, []byte(".bashrc\n"), 0644)
	suite.Require().NoError(err)

	// Get home directory for the test
	homeDir, err := os.UserHomeDir()
	suite.Require().NoError(err)

	targetFile := filepath.Join(homeDir, ".bashrc")

	// Clean up the test file after the test
	defer func() {
		_ = os.Remove(targetFile)
	}()

	// Test symlink restoration
	restored, err := suite.lnk.RestoreSymlinks()
	suite.Require().NoError(err)

	// Should have restored the symlink
	suite.Len(restored, 1)
	suite.Equal(".bashrc", restored[0])

	// Check that file is now a symlink
	info, err := os.Lstat(targetFile)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)
}

// Test error conditions
func (suite *CoreTestSuite) TestErrorConditions() {
	// Test add nonexistent file
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	err = suite.lnk.Add("/nonexistent/file")
	suite.Error(err)
	suite.Contains(err.Error(), "File does not exist")

	// Test remove unmanaged file
	testFile := filepath.Join(suite.tempDir, ".regularfile")
	err = os.WriteFile(testFile, []byte("content"), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Remove(testFile)
	suite.Error(err)
	suite.Contains(err.Error(), "File is not managed by lnk")

	// Test status without remote
	_, err = suite.lnk.Status()
	suite.Error(err)
	suite.Contains(err.Error(), "no remote configured")
}

// Test git operations
func (suite *CoreTestSuite) TestGitOperations() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add a file to create a commit
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	content := "export PATH=$PATH:/usr/local/bin"
	err = os.WriteFile(testFile, []byte(content), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Check that Git commit was made
	commits, err := suite.lnk.GetCommits()
	suite.Require().NoError(err)
	suite.Len(commits, 1)
	suite.Contains(commits[0], "lnk: added .bashrc")

	// Test add remote
	err = suite.lnk.AddRemote("origin", "https://github.com/test/dotfiles.git")
	suite.Require().NoError(err)

	// Test status with remote
	status, err := suite.lnk.Status()
	suite.Require().NoError(err)
	suite.Equal(1, status.Ahead)
	suite.Equal(0, status.Behind)
}

func TestCoreSuite(t *testing.T) {
	suite.Run(t, new(CoreTestSuite))
}
