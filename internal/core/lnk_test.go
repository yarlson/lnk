package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type CoreTestSuite struct {
	suite.Suite
	tempDir     string
	originalDir string
	lnk         *Lnk
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

	// Set HOME to temp directory for consistent relative path calculation
	suite.T().Setenv("HOME", tempDir)

	// Set XDG_CONFIG_HOME to temp directory
	suite.T().Setenv("XDG_CONFIG_HOME", tempDir)

	// Initialize Lnk instance
	suite.lnk = NewLnk()
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

	// The repository file will preserve the directory structure
	lnkDir := filepath.Join(suite.tempDir, "lnk")

	// Find the .bashrc file in the repository (it should be at the relative path from HOME)
	repoFile := filepath.Join(lnkDir, ".bashrc")
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

	// Check that the repository directory preserves the structure
	lnkDir := filepath.Join(suite.tempDir, "lnk")

	// The directory should be at the relative path from HOME
	repoDir := filepath.Join(lnkDir, "testdir")
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

	// The .lnk file now contains relative paths, not basenames
	// Check that the content contains references to .bashrc and .ssh
	content := string(lnkContent)
	suite.Contains(content, ".bashrc", ".lnk file should contain reference to .bashrc")
	suite.Contains(content, ".ssh", ".lnk file should contain reference to .ssh")

	// Remove one item and verify tracking is updated
	err = suite.lnk.Remove(testFile)
	suite.Require().NoError(err)

	lnkContent, err = os.ReadFile(lnkFile)
	suite.Require().NoError(err)

	lines = strings.Split(strings.TrimSpace(string(lnkContent)), "\n")
	suite.Len(lines, 1)

	content = string(lnkContent)
	suite.Contains(content, ".ssh", ".lnk file should still contain reference to .ssh")
	suite.NotContains(content, ".bashrc", ".lnk file should not contain reference to .bashrc after removal")
}

// Test XDG_CONFIG_HOME fallback
func (suite *CoreTestSuite) TestXDGConfigHomeFallback() {
	// Test fallback to ~/.config/lnk when XDG_CONFIG_HOME is not set
	suite.T().Setenv("XDG_CONFIG_HOME", "")

	homeDir := filepath.Join(suite.tempDir, "home")
	err := os.MkdirAll(homeDir, 0755)
	suite.Require().NoError(err)
	suite.T().Setenv("HOME", homeDir)

	lnk := NewLnk()
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
	suite.Contains(err.Error(), "File or directory not found")

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
	suite.Contains(err.Error(), "No remote repository is configured")
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

// Test edge case: files with same basename from different directories should be handled properly
func (suite *CoreTestSuite) TestSameBasenameFilesOverwrite() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create two directories with files having the same basename
	dirA := filepath.Join(suite.tempDir, "a")
	dirB := filepath.Join(suite.tempDir, "b")
	err = os.MkdirAll(dirA, 0755)
	suite.Require().NoError(err)
	err = os.MkdirAll(dirB, 0755)
	suite.Require().NoError(err)

	// Create files with same basename but different content
	fileA := filepath.Join(dirA, "config.json")
	fileB := filepath.Join(dirB, "config.json")
	contentA := `{"name": "config_a"}`
	contentB := `{"name": "config_b"}`

	err = os.WriteFile(fileA, []byte(contentA), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(fileB, []byte(contentB), 0644)
	suite.Require().NoError(err)

	// Add first file
	err = suite.lnk.Add(fileA)
	suite.Require().NoError(err)

	// Verify first file is managed correctly and preserves content
	info, err := os.Lstat(fileA)
	suite.Require().NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)

	symlinkContentA, err := os.ReadFile(fileA)
	suite.Require().NoError(err)
	suite.Equal(contentA, string(symlinkContentA), "First file should preserve its original content")

	// Add second file - this should work without overwriting the first
	err = suite.lnk.Add(fileB)
	suite.Require().NoError(err)

	// Verify second file is managed
	info, err = os.Lstat(fileB)
	suite.Require().NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)

	// CORRECT BEHAVIOR: Both files should preserve their original content
	symlinkContentA, err = os.ReadFile(fileA)
	suite.Require().NoError(err)
	symlinkContentB, err := os.ReadFile(fileB)
	suite.Require().NoError(err)

	suite.Equal(contentA, string(symlinkContentA), "First file should keep its original content")
	suite.Equal(contentB, string(symlinkContentB), "Second file should keep its original content")

	// Both files should be removable independently
	err = suite.lnk.Remove(fileA)
	suite.Require().NoError(err, "First file should be removable")

	// First file should be restored with correct content
	info, err = os.Lstat(fileA)
	suite.Require().NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink) // Not a symlink anymore

	restoredContentA, err := os.ReadFile(fileA)
	suite.Require().NoError(err)
	suite.Equal(contentA, string(restoredContentA), "Restored file should have original content")

	// Second file should still be manageable and removable
	err = suite.lnk.Remove(fileB)
	suite.Require().NoError(err, "Second file should also be removable without errors")

	// Second file should be restored with correct content
	info, err = os.Lstat(fileB)
	suite.Require().NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink) // Not a symlink anymore

	restoredContentB, err := os.ReadFile(fileB)
	suite.Require().NoError(err)
	suite.Equal(contentB, string(restoredContentB), "Second restored file should have original content")
}

// Test another variant: adding files with same basename should work correctly
func (suite *CoreTestSuite) TestSameBasenameSequentialAdd() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create subdirectories in different locations
	configDir := filepath.Join(suite.tempDir, "config")
	backupDir := filepath.Join(suite.tempDir, "backup")
	err = os.MkdirAll(configDir, 0755)
	suite.Require().NoError(err)
	err = os.MkdirAll(backupDir, 0755)
	suite.Require().NoError(err)

	// Create files with same basename (.bashrc)
	configBashrc := filepath.Join(configDir, ".bashrc")
	backupBashrc := filepath.Join(backupDir, ".bashrc")

	originalContent := "export PATH=/usr/local/bin:$PATH"
	backupContent := "export PATH=/opt/bin:$PATH"

	err = os.WriteFile(configBashrc, []byte(originalContent), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(backupBashrc, []byte(backupContent), 0644)
	suite.Require().NoError(err)

	// Add first .bashrc
	err = suite.lnk.Add(configBashrc)
	suite.Require().NoError(err)

	// Add second .bashrc - should work without overwriting the first
	err = suite.lnk.Add(backupBashrc)
	suite.Require().NoError(err)

	// Check .lnk tracking file should track both properly
	lnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk")
	lnkContent, err := os.ReadFile(lnkFile)
	suite.Require().NoError(err)

	// Both entries should be tracked and distinguishable
	content := string(lnkContent)
	suite.Contains(content, ".bashrc", "Both .bashrc files should be tracked")

	// Both files should maintain their distinct content
	content1, err := os.ReadFile(configBashrc)
	suite.Require().NoError(err)
	content2, err := os.ReadFile(backupBashrc)
	suite.Require().NoError(err)

	suite.Equal(originalContent, string(content1), "First file should keep original content")
	suite.Equal(backupContent, string(content2), "Second file should keep its distinct content")

	// Both should be removable independently
	err = suite.lnk.Remove(configBashrc)
	suite.Require().NoError(err, "First .bashrc should be removable")

	err = suite.lnk.Remove(backupBashrc)
	suite.Require().NoError(err, "Second .bashrc should be removable")
}

// Test dirty repository status detection
func (suite *CoreTestSuite) TestStatusDetectsDirtyRepo() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add and commit a file
	testFile := filepath.Join(suite.tempDir, "a")
	err = os.WriteFile(testFile, []byte("abc"), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Add a remote so status works
	err = suite.lnk.AddRemote("origin", "https://github.com/test/dotfiles.git")
	suite.Require().NoError(err)

	// Check status - should be clean but ahead of remote
	status, err := suite.lnk.Status()
	suite.Require().NoError(err)
	suite.Equal(1, status.Ahead)
	suite.Equal(0, status.Behind)
	suite.False(status.Dirty, "Repository should not be dirty after commit")

	// Now edit the managed file (simulating the issue scenario)
	err = os.WriteFile(testFile, []byte("def"), 0644)
	suite.Require().NoError(err)

	// Check status again - should detect dirty state
	status, err = suite.lnk.Status()
	suite.Require().NoError(err)
	suite.Equal(1, status.Ahead)
	suite.Equal(0, status.Behind)
	suite.True(status.Dirty, "Repository should be dirty after editing managed file")
}

// Test list functionality
func (suite *CoreTestSuite) TestListManagedItems() {
	// Test list without init - should fail
	_, err := suite.lnk.List()
	suite.Error(err)
	suite.Contains(err.Error(), "Lnk repository not initialized")

	// Initialize repository
	err = suite.lnk.Init()
	suite.Require().NoError(err)

	// Test list with no managed files
	items, err := suite.lnk.List()
	suite.Require().NoError(err)
	suite.Empty(items)

	// Add a file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	content := "export PATH=$PATH:/usr/local/bin"
	err = os.WriteFile(testFile, []byte(content), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Test list with one managed file
	items, err = suite.lnk.List()
	suite.Require().NoError(err)
	suite.Len(items, 1)
	suite.Contains(items[0], ".bashrc")

	// Add a directory
	testDir := filepath.Join(suite.tempDir, ".config")
	err = os.MkdirAll(testDir, 0755)
	suite.Require().NoError(err)
	configFile := filepath.Join(testDir, "app.conf")
	err = os.WriteFile(configFile, []byte("setting=value"), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testDir)
	suite.Require().NoError(err)

	// Test list with multiple managed items
	items, err = suite.lnk.List()
	suite.Require().NoError(err)
	suite.Len(items, 2)

	// Check that both items are present
	found := make(map[string]bool)
	for _, item := range items {
		if strings.Contains(item, ".bashrc") {
			found[".bashrc"] = true
		}
		if strings.Contains(item, ".config") {
			found[".config"] = true
		}
	}
	suite.True(found[".bashrc"], "Should contain .bashrc")
	suite.True(found[".config"], "Should contain .config")

	// Remove one item and verify list is updated
	err = suite.lnk.Remove(testFile)
	suite.Require().NoError(err)

	items, err = suite.lnk.List()
	suite.Require().NoError(err)
	suite.Len(items, 1)
	suite.Contains(items[0], ".config")
}

// Test multihost functionality
func (suite *CoreTestSuite) TestMultihostFileOperations() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create test files for different hosts
	testFile1 := filepath.Join(suite.tempDir, ".bashrc")
	content1 := "export PATH=$PATH:/usr/local/bin"
	err = os.WriteFile(testFile1, []byte(content1), 0644)
	suite.Require().NoError(err)

	testFile2 := filepath.Join(suite.tempDir, ".vimrc")
	content2 := "set number"
	err = os.WriteFile(testFile2, []byte(content2), 0644)
	suite.Require().NoError(err)

	// Add file to common configuration
	commonLnk := NewLnk()
	err = commonLnk.Add(testFile1)
	suite.Require().NoError(err)

	// Add file to host-specific configuration
	hostLnk := NewLnk(WithHost("workstation"))
	err = hostLnk.Add(testFile2)
	suite.Require().NoError(err)

	// Verify both files are symlinks
	info1, err := os.Lstat(testFile1)
	suite.Require().NoError(err)
	suite.Equal(os.ModeSymlink, info1.Mode()&os.ModeSymlink)

	info2, err := os.Lstat(testFile2)
	suite.Require().NoError(err)
	suite.Equal(os.ModeSymlink, info2.Mode()&os.ModeSymlink)

	// Verify common configuration tracking
	commonItems, err := commonLnk.List()
	suite.Require().NoError(err)
	suite.Len(commonItems, 1)
	suite.Contains(commonItems[0], ".bashrc")

	// Verify host-specific configuration tracking
	hostItems, err := hostLnk.List()
	suite.Require().NoError(err)
	suite.Len(hostItems, 1)
	suite.Contains(hostItems[0], ".vimrc")

	// Verify files are stored in correct locations
	lnkDir := filepath.Join(suite.tempDir, "lnk")

	// Common file should be in root
	commonFile := filepath.Join(lnkDir, ".lnk")
	suite.FileExists(commonFile)

	// Host-specific file should be in host subdirectory
	hostDir := filepath.Join(lnkDir, "workstation.lnk")
	suite.DirExists(hostDir)

	hostTrackingFile := filepath.Join(lnkDir, ".lnk.workstation")
	suite.FileExists(hostTrackingFile)

	// Test removal
	err = commonLnk.Remove(testFile1)
	suite.Require().NoError(err)

	err = hostLnk.Remove(testFile2)
	suite.Require().NoError(err)

	// Verify files are restored
	info1, err = os.Lstat(testFile1)
	suite.Require().NoError(err)
	suite.Equal(os.FileMode(0), info1.Mode()&os.ModeSymlink)

	info2, err = os.Lstat(testFile2)
	suite.Require().NoError(err)
	suite.Equal(os.FileMode(0), info2.Mode()&os.ModeSymlink)
}

// Test hostname detection
func (suite *CoreTestSuite) TestHostnameDetection() {
	hostname, err := GetCurrentHostname()
	suite.NoError(err)
	suite.NotEmpty(hostname)
}

// Test host-specific symlink restoration
func (suite *CoreTestSuite) TestMultihostSymlinkRestoration() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create files directly in host-specific storage (simulating a pull)
	hostLnk := NewLnk(WithHost("testhost"))

	// Ensure host storage directory exists
	hostStoragePath := hostLnk.getHostStoragePath()
	err = os.MkdirAll(hostStoragePath, 0755)
	suite.Require().NoError(err)

	// Create a file in host storage
	repoFile := filepath.Join(hostStoragePath, ".bashrc")
	content := "export HOST=testhost"
	err = os.WriteFile(repoFile, []byte(content), 0644)
	suite.Require().NoError(err)

	// Create host tracking file
	trackingFile := filepath.Join(suite.tempDir, "lnk", ".lnk.testhost")
	err = os.WriteFile(trackingFile, []byte(".bashrc\n"), 0644)
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
	restored, err := hostLnk.RestoreSymlinks()
	suite.Require().NoError(err)

	// Should have restored the symlink
	suite.Len(restored, 1)
	suite.Equal(".bashrc", restored[0])

	// Check that file is now a symlink
	info, err := os.Lstat(targetFile)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)
}

// Test that common and host-specific configurations don't interfere
func (suite *CoreTestSuite) TestMultihostIsolation() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create same file for common and host-specific
	testFile := filepath.Join(suite.tempDir, ".gitconfig")
	commonContent := "[user]\n\tname = Common User"
	err = os.WriteFile(testFile, []byte(commonContent), 0644)
	suite.Require().NoError(err)

	// Add to common
	commonLnk := NewLnk()
	err = commonLnk.Add(testFile)
	suite.Require().NoError(err)

	// Remove and recreate with different content
	err = commonLnk.Remove(testFile)
	suite.Require().NoError(err)

	hostContent := "[user]\n\tname = Work User"
	err = os.WriteFile(testFile, []byte(hostContent), 0644)
	suite.Require().NoError(err)

	// Add to host-specific
	hostLnk := NewLnk(WithHost("work"))
	err = hostLnk.Add(testFile)
	suite.Require().NoError(err)

	// Verify tracking files are separate
	commonItems, err := commonLnk.List()
	suite.Require().NoError(err)
	suite.Len(commonItems, 0) // Should be empty after removal

	hostItems, err := hostLnk.List()
	suite.Require().NoError(err)
	suite.Len(hostItems, 1)
	suite.Contains(hostItems[0], ".gitconfig")

	// Verify content is correct
	symlinkContent, err := os.ReadFile(testFile)
	suite.Require().NoError(err)
	suite.Equal(hostContent, string(symlinkContent))
}

// Test bootstrap script detection
func (suite *CoreTestSuite) TestFindBootstrapScript() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Test with no bootstrap script
	scriptPath, err := suite.lnk.FindBootstrapScript()
	suite.NoError(err)
	suite.Empty(scriptPath)

	// Test with bootstrap.sh
	bootstrapScript := filepath.Join(suite.tempDir, "lnk", "bootstrap.sh")
	err = os.WriteFile(bootstrapScript, []byte("#!/bin/bash\necho 'test'"), 0644)
	suite.Require().NoError(err)

	scriptPath, err = suite.lnk.FindBootstrapScript()
	suite.NoError(err)
	suite.Equal("bootstrap.sh", scriptPath)
}

// Test bootstrap script execution
func (suite *CoreTestSuite) TestRunBootstrapScript() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create a test script that creates a marker file
	bootstrapScript := filepath.Join(suite.tempDir, "lnk", "test.sh")
	markerFile := filepath.Join(suite.tempDir, "lnk", "bootstrap-executed.txt")
	scriptContent := fmt.Sprintf("#!/bin/bash\ntouch %s\necho 'Bootstrap executed'", markerFile)

	err = os.WriteFile(bootstrapScript, []byte(scriptContent), 0755)
	suite.Require().NoError(err)

	// Run the bootstrap script
	err = suite.lnk.RunBootstrapScript("test.sh")
	suite.NoError(err)

	// Verify the marker file was created
	suite.FileExists(markerFile)
}

// Test bootstrap script execution with error
func (suite *CoreTestSuite) TestRunBootstrapScriptWithError() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create a script that will fail
	bootstrapScript := filepath.Join(suite.tempDir, "lnk", "failing.sh")
	scriptContent := "#!/bin/bash\nexit 1"

	err = os.WriteFile(bootstrapScript, []byte(scriptContent), 0755)
	suite.Require().NoError(err)

	// Run the bootstrap script - should fail
	err = suite.lnk.RunBootstrapScript("failing.sh")
	suite.Error(err)
	suite.Contains(err.Error(), "Bootstrap script failed")
}

// Test running bootstrap on non-existent script
func (suite *CoreTestSuite) TestRunBootstrapScriptNotFound() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Try to run non-existent script
	err = suite.lnk.RunBootstrapScript("nonexistent.sh")
	suite.Error(err)
	suite.Contains(err.Error(), "Bootstrap script not found")
}

func (suite *CoreTestSuite) TestAddMultiple() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create multiple test files
	file1 := filepath.Join(suite.tempDir, "file1.txt")
	file2 := filepath.Join(suite.tempDir, "file2.txt")
	file3 := filepath.Join(suite.tempDir, "file3.txt")

	content1 := "content1"
	content2 := "content2"
	content3 := "content3"

	err = os.WriteFile(file1, []byte(content1), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(file2, []byte(content2), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(file3, []byte(content3), 0644)
	suite.Require().NoError(err)

	// Test AddMultiple method - should succeed
	paths := []string{file1, file2, file3}
	err = suite.lnk.AddMultiple(paths)
	suite.NoError(err, "AddMultiple should succeed")

	// Verify all files are now symlinks
	for _, file := range paths {
		info, err := os.Lstat(file)
		suite.NoError(err)
		suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink, "File should be a symlink: %s", file)
	}

	// Verify all files exist in storage
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	suite.FileExists(filepath.Join(lnkDir, "file1.txt"))
	suite.FileExists(filepath.Join(lnkDir, "file2.txt"))
	suite.FileExists(filepath.Join(lnkDir, "file3.txt"))

	// Verify .lnk file contains all entries
	lnkFile := filepath.Join(lnkDir, ".lnk")
	lnkContent, err := os.ReadFile(lnkFile)
	suite.NoError(err)
	suite.Equal("file1.txt\nfile2.txt\nfile3.txt\n", string(lnkContent))

	// Verify Git commit was created  
	commits, err := suite.lnk.GetCommits()
	suite.NoError(err)
	suite.T().Logf("Commits: %v", commits)
	// Should have at least 1 commit for the batch add
	suite.GreaterOrEqual(len(commits), 1)
	// The most recent commit should mention multiple files
	suite.Contains(commits[0], "added 3 files")
}

func (suite *CoreTestSuite) TestAddMultipleWithConflicts() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create test files
	file1 := filepath.Join(suite.tempDir, "file1.txt")
	file2 := filepath.Join(suite.tempDir, "file2.txt")
	file3 := filepath.Join(suite.tempDir, "file3.txt")

	err = os.WriteFile(file1, []byte("content1"), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(file2, []byte("content2"), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(file3, []byte("content3"), 0644)
	suite.Require().NoError(err)

	// Add file2 individually first
	err = suite.lnk.Add(file2)
	suite.Require().NoError(err)

	// Now try to add all three - should fail due to conflict with file2
	paths := []string{file1, file2, file3}
	err = suite.lnk.AddMultiple(paths)
	suite.Error(err, "AddMultiple should fail due to conflict")
	suite.Contains(err.Error(), "already managed")

	// Verify no partial changes were made
	// file1 and file3 should still be regular files, not symlinks
	info1, err := os.Lstat(file1)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info1.Mode()&os.ModeSymlink, "file1 should not be a symlink")

	info3, err := os.Lstat(file3)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info3.Mode()&os.ModeSymlink, "file3 should not be a symlink")

	// file2 should still be managed (was added before)
	info2, err := os.Lstat(file2)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info2.Mode()&os.ModeSymlink, "file2 should remain a symlink")
}

func (suite *CoreTestSuite) TestAddMultipleRollback() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create test files - one will be invalid to force rollback
	file1 := filepath.Join(suite.tempDir, "file1.txt")
	file2 := filepath.Join(suite.tempDir, "nonexistent.txt") // This doesn't exist
	file3 := filepath.Join(suite.tempDir, "file3.txt")

	err = os.WriteFile(file1, []byte("content1"), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(file3, []byte("content3"), 0644)
	suite.Require().NoError(err)
	// Note: file2 is intentionally not created

	// Try to add all files - should fail and rollback
	paths := []string{file1, file2, file3}
	err = suite.lnk.AddMultiple(paths)
	suite.Error(err, "AddMultiple should fail due to nonexistent file")

	// Verify rollback - no files should be symlinks
	info1, err := os.Lstat(file1)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info1.Mode()&os.ModeSymlink, "file1 should not be a symlink after rollback")

	info3, err := os.Lstat(file3)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info3.Mode()&os.ModeSymlink, "file3 should not be a symlink after rollback")

	// Verify no files in storage
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	suite.NoFileExists(filepath.Join(lnkDir, "file1.txt"))
	suite.NoFileExists(filepath.Join(lnkDir, "file3.txt"))

	// Verify .lnk file is empty or doesn't contain these files
	lnkFile := filepath.Join(lnkDir, ".lnk")
	if _, err := os.Stat(lnkFile); err == nil {
		lnkContent, err := os.ReadFile(lnkFile)
		suite.NoError(err)
		content := string(lnkContent)
		suite.NotContains(content, "file1.txt")
		suite.NotContains(content, "file3.txt")
	}
}

func (suite *CoreTestSuite) TestValidateMultiplePaths() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create a mix of valid and invalid paths
	validFile := filepath.Join(suite.tempDir, "valid.txt")
	err = os.WriteFile(validFile, []byte("content"), 0644)
	suite.Require().NoError(err)

	nonexistentFile := filepath.Join(suite.tempDir, "nonexistent.txt")
	// Don't create this file

	// Create a valid directory
	validDir := filepath.Join(suite.tempDir, "validdir")
	err = os.MkdirAll(validDir, 0755)
	suite.Require().NoError(err)

	// Test validation fails early with detailed error
	paths := []string{validFile, nonexistentFile, validDir}
	err = suite.lnk.AddMultiple(paths)
	suite.Error(err, "Should fail due to nonexistent file")
	suite.Contains(err.Error(), "validation failed")
	suite.Contains(err.Error(), "nonexistent.txt")

	// Verify no partial changes were made
	info, err := os.Lstat(validFile)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink, "valid file should not be a symlink")

	info, err = os.Lstat(validDir)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink, "valid directory should not be a symlink")
}

func (suite *CoreTestSuite) TestAtomicRollbackOnFailure() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create test files
	file1 := filepath.Join(suite.tempDir, "file1.txt")
	file2 := filepath.Join(suite.tempDir, "file2.txt")
	file3 := filepath.Join(suite.tempDir, "file3.txt")

	content1 := "original content 1"
	content2 := "original content 2"
	content3 := "original content 3"

	err = os.WriteFile(file1, []byte(content1), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(file2, []byte(content2), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(file3, []byte(content3), 0644)
	suite.Require().NoError(err)

	// Add file2 individually first to create a conflict
	err = suite.lnk.Add(file2)
	suite.Require().NoError(err)

	// Store original states
	info1Before, err := os.Lstat(file1)
	suite.Require().NoError(err)
	info3Before, err := os.Lstat(file3)
	suite.Require().NoError(err)

	// Try to add all files - should fail and rollback completely
	paths := []string{file1, file2, file3}
	err = suite.lnk.AddMultiple(paths)
	suite.Error(err, "Should fail due to conflict with file2")

	// Verify complete rollback
	info1After, err := os.Lstat(file1)
	suite.NoError(err)
	suite.Equal(info1Before.Mode(), info1After.Mode(), "file1 mode should be unchanged")
	
	info3After, err := os.Lstat(file3)
	suite.NoError(err)
	suite.Equal(info3Before.Mode(), info3After.Mode(), "file3 mode should be unchanged")

	// Verify original contents are preserved
	content1After, err := os.ReadFile(file1)
	suite.NoError(err)
	suite.Equal(content1, string(content1After), "file1 content should be preserved")

	content3After, err := os.ReadFile(file3)
	suite.NoError(err)
	suite.Equal(content3, string(content3After), "file3 content should be preserved")

	// file2 should still be managed (was added before)
	info2, err := os.Lstat(file2)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info2.Mode()&os.ModeSymlink, "file2 should remain a symlink")
}

func (suite *CoreTestSuite) TestDetailedErrorMessages() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Test with multiple types of errors
	validFile := filepath.Join(suite.tempDir, "valid.txt")
	err = os.WriteFile(validFile, []byte("content"), 0644)
	suite.Require().NoError(err)

	nonexistentFile := filepath.Join(suite.tempDir, "does-not-exist.txt")
	alreadyManagedFile := filepath.Join(suite.tempDir, "already-managed.txt")
	err = os.WriteFile(alreadyManagedFile, []byte("managed"), 0644)
	suite.Require().NoError(err)

	// Add one file first to create conflict
	err = suite.lnk.Add(alreadyManagedFile)
	suite.Require().NoError(err)

	// Test with nonexistent file
	paths := []string{validFile, nonexistentFile}
	err = suite.lnk.AddMultiple(paths)
	suite.Error(err, "Should fail due to nonexistent file")
	suite.Contains(err.Error(), "validation failed", "Error should mention validation failure")
	suite.Contains(err.Error(), "does-not-exist.txt", "Error should include specific filename")

	// Test with already managed file
	paths = []string{validFile, alreadyManagedFile}
	err = suite.lnk.AddMultiple(paths)
	suite.Error(err, "Should fail due to already managed file")
	suite.Contains(err.Error(), "already managed", "Error should mention already managed")
	suite.Contains(err.Error(), "already-managed.txt", "Error should include specific filename")
}

func TestCoreSuite(t *testing.T) {
	suite.Run(t, new(CoreTestSuite))
}
