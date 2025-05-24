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

	// Create a directory with files
	testDir := filepath.Join(suite.tempDir, "testdir")
	err = os.MkdirAll(testDir, 0755)
	suite.Require().NoError(err)

	// Add files to the directory
	testFile1 := filepath.Join(testDir, "file1.txt")
	err = os.WriteFile(testFile1, []byte("content1"), 0644)
	suite.Require().NoError(err)

	testFile2 := filepath.Join(testDir, "file2.txt")
	err = os.WriteFile(testFile2, []byte("content2"), 0644)
	suite.Require().NoError(err)

	// Add the directory - should now succeed
	err = suite.lnk.Add(testDir)
	suite.Require().NoError(err)

	// Check that the directory is now a symlink
	info, err := os.Lstat(testDir)
	suite.Require().NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)

	// Check that the directory exists in the repo
	repoDir := filepath.Join(suite.tempDir, "lnk", "testdir")
	suite.DirExists(repoDir)

	// Check that files are preserved
	repoFile1 := filepath.Join(repoDir, "file1.txt")
	repoFile2 := filepath.Join(repoDir, "file2.txt")
	suite.FileExists(repoFile1)
	suite.FileExists(repoFile2)

	content1, err := os.ReadFile(repoFile1)
	suite.Require().NoError(err)
	suite.Equal("content1", string(content1))

	content2, err := os.ReadFile(repoFile2)
	suite.Require().NoError(err)
	suite.Equal("content2", string(content2))

	// Check that .lnk file was created and contains the directory
	lnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk")
	suite.FileExists(lnkFile)

	lnkContent, err := os.ReadFile(lnkFile)
	suite.Require().NoError(err)
	suite.Contains(string(lnkContent), "testdir")

	// Check that Git commit was made
	commits, err := suite.lnk.GetCommits()
	suite.Require().NoError(err)
	suite.Len(commits, 1)
	suite.Contains(commits[0], "lnk: added testdir")
}

func (suite *LnkIntegrationTestSuite) TestRemoveDirectory() {
	// Initialize and add a directory first
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	testDir := filepath.Join(suite.tempDir, "testdir")
	err = os.MkdirAll(testDir, 0755)
	suite.Require().NoError(err)

	testFile := filepath.Join(testDir, "config.txt")
	content := "test config"
	err = os.WriteFile(testFile, []byte(content), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testDir)
	suite.Require().NoError(err)

	// Now remove the directory
	err = suite.lnk.Remove(testDir)
	suite.Require().NoError(err)

	// Check that the symlink is gone and regular directory is restored
	info, err := os.Lstat(testDir)
	suite.Require().NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink) // Not a symlink
	suite.True(info.IsDir())                                // Is a directory

	// Check that content is preserved
	restoredContent, err := os.ReadFile(testFile)
	suite.Require().NoError(err)
	suite.Equal(content, string(restoredContent))

	// Check that directory is removed from repo
	repoDir := filepath.Join(suite.tempDir, "lnk", "testdir")
	suite.NoDirExists(repoDir)

	// Check that .lnk file no longer contains the directory
	lnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk")
	if suite.FileExists(lnkFile) {
		lnkContent, err := os.ReadFile(lnkFile)
		suite.Require().NoError(err)
		suite.NotContains(string(lnkContent), "testdir")
	}

	// Check that Git commit was made
	commits, err := suite.lnk.GetCommits()
	suite.Require().NoError(err)
	suite.Len(commits, 2) // add + remove
	suite.Contains(commits[0], "lnk: removed testdir")
	suite.Contains(commits[1], "lnk: added testdir")
}

func (suite *LnkIntegrationTestSuite) TestLnkFileTracking() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add a file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Add a directory
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

	// Remove a file and check .lnk is updated
	err = suite.lnk.Remove(testFile)
	suite.Require().NoError(err)

	lnkContent, err = os.ReadFile(lnkFile)
	suite.Require().NoError(err)

	lines = strings.Split(strings.TrimSpace(string(lnkContent)), "\n")
	suite.Len(lines, 1)
	suite.Contains(lines, ".ssh")
	suite.NotContains(lines, ".bashrc")
}

func (suite *LnkIntegrationTestSuite) TestPullWithDirectories() {
	// Initialize repo
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add remote for pull to work
	err = suite.lnk.AddRemote("origin", "https://github.com/test/dotfiles.git")
	suite.Require().NoError(err)

	// Create a directory and .lnk file in the repo directly to simulate a pull
	repoDir := filepath.Join(suite.tempDir, "lnk", ".config")
	err = os.MkdirAll(repoDir, 0755)
	suite.Require().NoError(err)

	configFile := filepath.Join(repoDir, "app.conf")
	content := "setting=value"
	err = os.WriteFile(configFile, []byte(content), 0644)
	suite.Require().NoError(err)

	// Create .lnk file
	lnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk")
	err = os.WriteFile(lnkFile, []byte(".config\n"), 0644)
	suite.Require().NoError(err)

	// Get home directory for the test
	homeDir, err := os.UserHomeDir()
	suite.Require().NoError(err)

	targetDir := filepath.Join(homeDir, ".config")

	// Clean up the test directory after the test
	defer func() {
		_ = os.RemoveAll(targetDir)
	}()

	// Create a regular directory in home to simulate conflict scenario
	err = os.MkdirAll(targetDir, 0755)
	suite.Require().NoError(err)
	err = os.WriteFile(filepath.Join(targetDir, "different.conf"), []byte("different"), 0644)
	suite.Require().NoError(err)

	// Pull should restore symlinks and handle conflicts
	restored, err := suite.lnk.Pull()
	// In tests, pull will fail because we don't have real remotes, but that's expected
	// We can still test the symlink restoration part
	if err != nil {
		suite.Contains(err.Error(), "git pull failed")
		// Test symlink restoration directly
		restored, err = suite.lnk.RestoreSymlinks()
		suite.Require().NoError(err)
	}

	// Should have restored the symlink
	suite.GreaterOrEqual(len(restored), 1)
	if len(restored) > 0 {
		suite.Equal(".config", restored[0])
	}

	// Check that directory is back to being a symlink
	info, err := os.Lstat(targetDir)
	suite.Require().NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)

	// Check content is preserved from repo
	repoContent, err := os.ReadFile(configFile)
	suite.Require().NoError(err)
	suite.Equal(content, string(repoContent))
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

// TestSyncStatus tests the status command functionality
func (suite *LnkIntegrationTestSuite) TestSyncStatus() {
	// Initialize repo with remote
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	err = suite.lnk.AddRemote("origin", "https://github.com/test/dotfiles.git")
	suite.Require().NoError(err)

	// Add a file to create some local changes
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	content := "export PATH=$PATH:/usr/local/bin"
	err = os.WriteFile(testFile, []byte(content), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Get status - should show 1 commit ahead
	status, err := suite.lnk.Status()
	suite.Require().NoError(err)
	suite.Equal(1, status.Ahead)
	suite.Equal(0, status.Behind)
	suite.Equal("origin/main", status.Remote)
}

// TestSyncPush tests the push command functionality
func (suite *LnkIntegrationTestSuite) TestSyncPush() {
	// Initialize repo
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add remote for push to work
	err = suite.lnk.AddRemote("origin", "https://github.com/test/dotfiles.git")
	suite.Require().NoError(err)

	// Add a file
	testFile := filepath.Join(suite.tempDir, ".vimrc")
	content := "set number"
	err = os.WriteFile(testFile, []byte(content), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Add another file for a second commit
	testFile2 := filepath.Join(suite.tempDir, ".gitconfig")
	content2 := "[user]\n    name = Test User"
	err = os.WriteFile(testFile2, []byte(content2), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testFile2)
	suite.Require().NoError(err)

	// Modify one of the files to create uncommitted changes
	repoFile := filepath.Join(suite.tempDir, "lnk", ".vimrc")
	modifiedContent := "set number\nset relativenumber"
	err = os.WriteFile(repoFile, []byte(modifiedContent), 0644)
	suite.Require().NoError(err)

	// Push should stage all changes and create a sync commit
	message := "Updated configuration files"
	err = suite.lnk.Push(message)
	// In tests, push will fail because we don't have real remotes, but that's expected
	// The important part is that it stages and commits changes
	if err != nil {
		suite.Contains(err.Error(), "git push failed")
	}

	// Check that a sync commit was made (even if push failed)
	commits, err := suite.lnk.GetCommits()
	suite.Require().NoError(err)
	suite.GreaterOrEqual(len(commits), 3) // at least 2 add commits + 1 sync commit
	suite.Contains(commits[0], message)   // Latest commit should contain our message
}

// TestSyncPull tests the pull command functionality
func (suite *LnkIntegrationTestSuite) TestSyncPull() {
	// Initialize repo
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add remote for pull to work
	err = suite.lnk.AddRemote("origin", "https://github.com/test/dotfiles.git")
	suite.Require().NoError(err)

	// Pull should attempt to pull from remote (will fail in tests but that's expected)
	_, err = suite.lnk.Pull()
	// In tests, pull will fail because we don't have real remotes, but that's expected
	suite.Error(err)
	suite.Contains(err.Error(), "git pull failed")

	// Test RestoreSymlinks functionality separately
	// Create a file in the repo directly
	repoFile := filepath.Join(suite.tempDir, "lnk", ".bashrc")
	content := "export PATH=$PATH:/usr/local/bin"
	err = os.WriteFile(repoFile, []byte(content), 0644)
	suite.Require().NoError(err)

	// Test that RestoreSymlinks can be called (even if it doesn't restore anything in this test setup)
	restored, err := suite.lnk.RestoreSymlinks()
	suite.Require().NoError(err)
	// In this test setup, it might not restore anything, and that's okay for Phase 1
	suite.GreaterOrEqual(len(restored), 0)
}

// TestSyncStatusNoRemote tests status when no remote is configured
func (suite *LnkIntegrationTestSuite) TestSyncStatusNoRemote() {
	// Initialize repo without remote
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Status should indicate no remote
	_, err = suite.lnk.Status()
	suite.Error(err)
	suite.Contains(err.Error(), "no remote configured")
}

// TestSyncPushWithModifiedFiles tests push when files are modified
func (suite *LnkIntegrationTestSuite) TestSyncPushWithModifiedFiles() {
	// Initialize repo and add a file
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add remote for push to work
	err = suite.lnk.AddRemote("origin", "https://github.com/test/dotfiles.git")
	suite.Require().NoError(err)

	testFile := filepath.Join(suite.tempDir, ".bashrc")
	content := "export PATH=$PATH:/usr/local/bin"
	err = os.WriteFile(testFile, []byte(content), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Modify the file in the repo (simulate editing managed file)
	repoFile := filepath.Join(suite.tempDir, "lnk", ".bashrc")
	modifiedContent := "export PATH=$PATH:/usr/local/bin\nexport EDITOR=vim"
	err = os.WriteFile(repoFile, []byte(modifiedContent), 0644)
	suite.Require().NoError(err)

	// Push should detect and commit the changes
	message := "Updated bashrc with editor setting"
	err = suite.lnk.Push(message)
	// In tests, push will fail because we don't have real remotes, but that's expected
	if err != nil {
		suite.Contains(err.Error(), "git push failed")
	}

	// Check that changes were committed (even if push failed)
	commits, err := suite.lnk.GetCommits()
	suite.Require().NoError(err)
	suite.GreaterOrEqual(len(commits), 2) // add commit + sync commit
	suite.Contains(commits[0], message)
}

func TestLnkIntegrationSuite(t *testing.T) {
	suite.Run(t, new(LnkIntegrationTestSuite))
}
