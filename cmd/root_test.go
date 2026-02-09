package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type CLITestSuite struct {
	suite.Suite
	tempDir     string
	originalDir string
	stdout      *bytes.Buffer
	stderr      *bytes.Buffer
}

func (suite *CLITestSuite) SetupTest() {
	// Create temp directory and change to it
	tempDir, err := os.MkdirTemp("", "lnk-cli-test-*")
	suite.Require().NoError(err)
	suite.tempDir = tempDir

	originalDir, err := os.Getwd()
	suite.Require().NoError(err)
	suite.originalDir = originalDir

	err = os.Chdir(tempDir)
	suite.Require().NoError(err)

	// Set HOME to temp directory for consistent relative path calculation
	suite.T().Setenv("HOME", tempDir)

	// Set XDG_CONFIG_HOME to tempDir/.config for config files
	suite.T().Setenv("XDG_CONFIG_HOME", filepath.Join(tempDir, ".config"))

	// Capture output
	suite.stdout = &bytes.Buffer{}
	suite.stderr = &bytes.Buffer{}
}

func (suite *CLITestSuite) TearDownTest() {
	err := os.Chdir(suite.originalDir)
	suite.Require().NoError(err)
	err = os.RemoveAll(suite.tempDir)
	suite.Require().NoError(err)
}

func (suite *CLITestSuite) runCommand(args ...string) error {
	rootCmd := NewRootCommand()
	rootCmd.SetOut(suite.stdout)
	rootCmd.SetErr(suite.stderr)
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func (suite *CLITestSuite) TestInitCommand() {
	err := suite.runCommand("init")
	suite.NoError(err)

	// Check output
	output := suite.stdout.String()
	suite.Contains(output, "Initialized empty lnk repository")
	suite.Contains(output, "Location:")
	suite.Contains(output, "Next steps:")
	suite.Contains(output, "lnk add <file>")

	// Verify actual effect
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	suite.DirExists(lnkDir)

	gitDir := filepath.Join(lnkDir, ".git")
	suite.DirExists(gitDir)
}

func (suite *CLITestSuite) TestAddCommand() {
	// Initialize first
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Create test file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)

	// Test add command
	err = suite.runCommand("add", testFile)
	suite.NoError(err)

	// Check output
	output := suite.stdout.String()
	suite.Contains(output, "Added .bashrc to lnk")
	suite.Contains(output, "→")
	suite.Contains(output, "sync to remote")

	// Verify symlink was created
	info, err := os.Lstat(testFile)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)

	// Verify the file exists in repo with preserved directory structure
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	repoFile := filepath.Join(lnkDir, ".bashrc")
	suite.FileExists(repoFile)

	// Verify content is preserved in storage
	storedContent, err := os.ReadFile(repoFile)
	suite.NoError(err)
	suite.Equal("export PATH=/usr/local/bin:$PATH", string(storedContent))

	// Verify .lnk file contains the correct entry
	lnkFile := filepath.Join(lnkDir, ".lnk")
	lnkContent, err := os.ReadFile(lnkFile)
	suite.NoError(err)
	suite.Equal(".bashrc\n", string(lnkContent))
}

func (suite *CLITestSuite) TestRemoveCommand() {
	// Setup: init and add a file
	_ = suite.runCommand("init")
	testFile := filepath.Join(suite.tempDir, ".vimrc")
	_ = os.WriteFile(testFile, []byte("set number"), 0644)
	_ = suite.runCommand("add", testFile)
	suite.stdout.Reset()

	// Test remove command
	err := suite.runCommand("rm", testFile)
	suite.NoError(err)

	// Check output
	output := suite.stdout.String()
	suite.Contains(output, "Removed .vimrc from lnk")
	suite.Contains(output, "→")
	suite.Contains(output, "Original file restored")

	// Verify symlink is gone and regular file is restored
	info, err := os.Lstat(testFile)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink) // Not a symlink

	// Verify content is preserved
	content, err := os.ReadFile(testFile)
	suite.NoError(err)
	suite.Equal("set number", string(content))
}

func (suite *CLITestSuite) TestStatusCommand() {
	// Initialize first
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Test status without remote - should fail
	err = suite.runCommand("status")
	suite.Error(err)
	suite.Contains(err.Error(), "No remote repository is configured")
}

func (suite *CLITestSuite) TestListCommand() {
	// Test list without init - should fail
	err := suite.runCommand("list")
	suite.Error(err)
	suite.Contains(err.Error(), "Lnk repository not initialized")

	// Initialize first
	err = suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Test list with no managed files
	err = suite.runCommand("list")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "No files currently managed by lnk")
	suite.Contains(output, "lnk add <file>")
	suite.stdout.Reset()

	// Add a file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Test list with one managed file
	err = suite.runCommand("list")
	suite.NoError(err)
	output = suite.stdout.String()
	suite.Contains(output, "Files managed by lnk")
	suite.Contains(output, "1 item")
	suite.Contains(output, ".bashrc")
	suite.stdout.Reset()

	// Add another file
	testFile2 := filepath.Join(suite.tempDir, ".vimrc")
	err = os.WriteFile(testFile2, []byte("set number"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile2)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Test list with multiple managed files
	err = suite.runCommand("list")
	suite.NoError(err)
	output = suite.stdout.String()
	suite.Contains(output, "Files managed by lnk")
	suite.Contains(output, "2 items")
	suite.Contains(output, ".bashrc")
	suite.Contains(output, ".vimrc")

	// Verify both files exist in storage with correct content
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")

	bashrcStorage := filepath.Join(lnkDir, ".bashrc")
	suite.FileExists(bashrcStorage)
	bashrcContent, err := os.ReadFile(bashrcStorage)
	suite.NoError(err)
	suite.Equal("export PATH=/usr/local/bin:$PATH", string(bashrcContent))

	vimrcStorage := filepath.Join(lnkDir, ".vimrc")
	suite.FileExists(vimrcStorage)
	vimrcContent, err := os.ReadFile(vimrcStorage)
	suite.NoError(err)
	suite.Equal("set number", string(vimrcContent))

	// Verify .lnk file contains both entries (sorted)
	lnkFile := filepath.Join(lnkDir, ".lnk")
	lnkContent, err := os.ReadFile(lnkFile)
	suite.NoError(err)
	suite.Equal(".bashrc\n.vimrc\n", string(lnkContent))
}

func (suite *CLITestSuite) TestErrorHandling() {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		outContains string
	}{
		{
			name:        "add nonexistent file",
			args:        []string{"add", "/nonexistent/file"},
			wantErr:     true,
			errContains: "File or directory not found",
		},
		{
			name:        "status without init",
			args:        []string{"status"},
			wantErr:     true,
			errContains: "Lnk repository not initialized",
		},
		{
			name:        "help command",
			args:        []string{"--help"},
			wantErr:     false,
			outContains: "Lnk - Git-native dotfiles management",
		},
		{
			name:        "version command",
			args:        []string{"--version"},
			wantErr:     false,
			outContains: "lnk version",
		},
		{
			name:        "init help",
			args:        []string{"init", "--help"},
			wantErr:     false,
			outContains: "Creates the lnk directory",
		},
		{
			name:        "add help",
			args:        []string{"add", "--help"},
			wantErr:     false,
			outContains: "Moves files to the lnk repository",
		},
		{
			name:        "list help",
			args:        []string{"list", "--help"},
			wantErr:     false,
			outContains: "Display all files and directories",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.stdout.Reset()
			suite.stderr.Reset()

			err := suite.runCommand(tt.args...)

			if tt.wantErr {
				suite.Error(err, "Expected error for %s", tt.name)
				if tt.errContains != "" {
					suite.Contains(err.Error(), tt.errContains, "Wrong error message for %s", tt.name)
				}
			} else {
				suite.NoError(err, "Unexpected error for %s", tt.name)
			}

			if tt.outContains != "" {
				output := suite.stdout.String()
				suite.Contains(output, tt.outContains, "Expected output not found for %s", tt.name)
			}
		})
	}
}

func (suite *CLITestSuite) TestCompleteWorkflow() {
	// Test realistic user workflow
	steps := []struct {
		name   string
		args   []string
		setup  func()
		verify func(output string)
	}{
		{
			name: "initialize repository",
			args: []string{"init"},
			verify: func(output string) {
				suite.Contains(output, "Initialized empty lnk repository")
			},
		},
		{
			name: "add config file",
			args: []string{"add", filepath.Join(suite.tempDir, ".bashrc")},
			setup: func() {
				testFile := filepath.Join(suite.tempDir, ".bashrc")
				_ = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
			},
			verify: func(output string) {
				suite.Contains(output, "Added .bashrc to lnk")

				// Verify storage and .lnk file
				lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
				bashrcStorage := filepath.Join(lnkDir, ".bashrc")
				suite.FileExists(bashrcStorage)

				storedContent, err := os.ReadFile(bashrcStorage)
				suite.NoError(err)
				suite.Equal("export PATH=/usr/local/bin:$PATH", string(storedContent))

				lnkFile := filepath.Join(lnkDir, ".lnk")
				lnkContent, err := os.ReadFile(lnkFile)
				suite.NoError(err)
				suite.Equal(".bashrc\n", string(lnkContent))
			},
		},
		{
			name: "add another file",
			args: []string{"add", filepath.Join(suite.tempDir, ".vimrc")},
			setup: func() {
				testFile := filepath.Join(suite.tempDir, ".vimrc")
				_ = os.WriteFile(testFile, []byte("set number"), 0644)
			},
			verify: func(output string) {
				suite.Contains(output, "Added .vimrc to lnk")

				// Verify storage and .lnk file now contains both files
				lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
				vimrcStorage := filepath.Join(lnkDir, ".vimrc")
				suite.FileExists(vimrcStorage)

				storedContent, err := os.ReadFile(vimrcStorage)
				suite.NoError(err)
				suite.Equal("set number", string(storedContent))

				lnkFile := filepath.Join(lnkDir, ".lnk")
				lnkContent, err := os.ReadFile(lnkFile)
				suite.NoError(err)
				suite.Equal(".bashrc\n.vimrc\n", string(lnkContent))
			},
		},
		{
			name: "remove file",
			args: []string{"rm", filepath.Join(suite.tempDir, ".vimrc")},
			verify: func(output string) {
				suite.Contains(output, "Removed .vimrc from lnk")
			},
		},
	}

	for _, step := range steps {
		suite.Run(step.name, func() {
			if step.setup != nil {
				step.setup()
			}

			suite.stdout.Reset()
			suite.stderr.Reset()

			err := suite.runCommand(step.args...)
			suite.NoError(err, "Step %s failed: %v", step.name, err)

			output := suite.stdout.String()
			if step.verify != nil {
				step.verify(output)
			}
		})
	}
}

func (suite *CLITestSuite) TestRemoveUnmanagedFile() {
	// Initialize repository
	_ = suite.runCommand("init")

	// Create a regular file (not managed by lnk)
	testFile := filepath.Join(suite.tempDir, ".regularfile")
	_ = os.WriteFile(testFile, []byte("content"), 0644)

	// Try to remove it
	err := suite.runCommand("rm", testFile)
	suite.Error(err)
	suite.Contains(err.Error(), "File is not managed by lnk")
}

func (suite *CLITestSuite) TestAddDirectory() {
	// Initialize repository
	_ = suite.runCommand("init")
	suite.stdout.Reset()

	// Create a directory with files
	testDir := filepath.Join(suite.tempDir, ".ssh")
	_ = os.MkdirAll(testDir, 0755)
	configFile := filepath.Join(testDir, "config")
	_ = os.WriteFile(configFile, []byte("Host example.com"), 0644)

	// Add the directory
	err := suite.runCommand("add", testDir)
	suite.NoError(err)

	// Check output
	output := suite.stdout.String()
	suite.Contains(output, "Added .ssh to lnk")

	// Verify directory is now a symlink
	info, err := os.Lstat(testDir)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)

	// Verify the directory exists in repo with preserved directory structure
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	repoDir := filepath.Join(lnkDir, ".ssh")
	suite.DirExists(repoDir)

	// Verify directory content is preserved
	repoConfigFile := filepath.Join(repoDir, "config")
	suite.FileExists(repoConfigFile)
	storedContent, err := os.ReadFile(repoConfigFile)
	suite.NoError(err)
	suite.Equal("Host example.com", string(storedContent))

	// Verify .lnk file contains the directory entry
	lnkFile := filepath.Join(lnkDir, ".lnk")
	lnkContent, err := os.ReadFile(lnkFile)
	suite.NoError(err)
	suite.Equal(".ssh\n", string(lnkContent))
}

func (suite *CLITestSuite) TestSameBasenameFilesBug() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

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
	err = suite.runCommand("add", fileA)
	suite.NoError(err)
	suite.stdout.Reset()

	// Verify first file content is preserved
	content, err := os.ReadFile(fileA)
	suite.NoError(err)
	suite.Equal(contentA, string(content), "First file should preserve its original content")

	// Add second file with same basename - this should work correctly
	err = suite.runCommand("add", fileB)
	suite.NoError(err, "Adding second file with same basename should work")

	// CORRECT BEHAVIOR: Both files should preserve their original content
	contentAfterAddA, err := os.ReadFile(fileA)
	suite.NoError(err)
	contentAfterAddB, err := os.ReadFile(fileB)
	suite.NoError(err)

	suite.Equal(contentA, string(contentAfterAddA), "First file should keep its original content")
	suite.Equal(contentB, string(contentAfterAddB), "Second file should keep its original content")

	// Verify both files exist in storage with correct paths and content
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")

	storageFileA := filepath.Join(lnkDir, "a", "config.json")
	suite.FileExists(storageFileA)
	storedContentA, err := os.ReadFile(storageFileA)
	suite.NoError(err)
	suite.Equal(contentA, string(storedContentA))

	storageFileB := filepath.Join(lnkDir, "b", "config.json")
	suite.FileExists(storageFileB)
	storedContentB, err := os.ReadFile(storageFileB)
	suite.NoError(err)
	suite.Equal(contentB, string(storedContentB))

	// Verify .lnk file contains both entries with correct relative paths
	lnkFile := filepath.Join(lnkDir, ".lnk")
	lnkContent, err := os.ReadFile(lnkFile)
	suite.NoError(err)
	suite.Equal("a/config.json\nb/config.json\n", string(lnkContent))

	// Both files should be removable independently
	suite.stdout.Reset()
	err = suite.runCommand("rm", fileA)
	suite.NoError(err, "First file should be removable")

	// Verify output shows removal
	output := suite.stdout.String()
	suite.Contains(output, "Removed config.json from lnk")

	// Verify first file is restored with correct content
	restoredContentA, err := os.ReadFile(fileA)
	suite.NoError(err)
	suite.Equal(contentA, string(restoredContentA), "Restored first file should have original content")

	// Second file should still be removable without errors
	suite.stdout.Reset()
	err = suite.runCommand("rm", fileB)
	suite.NoError(err, "Second file should also be removable without errors")

	// Verify second file is restored with correct content
	restoredContentB, err := os.ReadFile(fileB)
	suite.NoError(err)
	suite.Equal(contentB, string(restoredContentB), "Restored second file should have original content")
}

func (suite *CLITestSuite) TestStatusDirtyRepo() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add and commit a file
	testFile := filepath.Join(suite.tempDir, "a")
	err = os.WriteFile(testFile, []byte("abc"), 0644)
	suite.Require().NoError(err)

	err = suite.runCommand("add", testFile)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Verify file is stored correctly
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	storageFile := filepath.Join(lnkDir, "a")
	suite.FileExists(storageFile)
	storedContent, err := os.ReadFile(storageFile)
	suite.NoError(err)
	suite.Equal("abc", string(storedContent))

	// Verify .lnk file contains the entry
	lnkFile := filepath.Join(lnkDir, ".lnk")
	lnkContent, err := os.ReadFile(lnkFile)
	suite.NoError(err)
	suite.Equal("a\n", string(lnkContent))

	// Add a remote so status works
	cmd := exec.Command("git", "remote", "add", "origin", "https://github.com/test/dotfiles.git")
	cmd.Dir = lnkDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// Status should show clean but ahead
	err = suite.runCommand("status")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "1 commit ahead")
	suite.NotContains(output, "uncommitted changes")
	suite.stdout.Reset()

	// Now edit the managed file (simulating the issue scenario)
	err = os.WriteFile(testFile, []byte("def"), 0644)
	suite.Require().NoError(err)

	// Status should now detect dirty state and NOT say "up to date"
	err = suite.runCommand("status")
	suite.NoError(err)
	output = suite.stdout.String()
	suite.Contains(output, "Repository has uncommitted changes")
	suite.NotContains(output, "Repository is up to date")
	suite.Contains(output, "lnk push")
}

func (suite *CLITestSuite) TestMultihostCommands() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Create test files
	testFile1 := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile1, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)

	testFile2 := filepath.Join(suite.tempDir, ".vimrc")
	err = os.WriteFile(testFile2, []byte("set number"), 0644)
	suite.Require().NoError(err)

	// Add file to common configuration
	err = suite.runCommand("add", testFile1)
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "Added .bashrc to lnk")
	suite.NotContains(output, "host:")
	suite.stdout.Reset()

	// Add file to host-specific configuration
	err = suite.runCommand("add", "--host", "workstation", testFile2)
	suite.NoError(err)
	output = suite.stdout.String()
	suite.Contains(output, "Added .vimrc to lnk (host: workstation)")
	suite.Contains(output, "workstation.lnk")
	suite.stdout.Reset()

	// Verify storage paths and .lnk files for both common and host-specific
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")

	// Verify common file storage and tracking
	commonStorage := filepath.Join(lnkDir, ".bashrc")
	suite.FileExists(commonStorage)
	commonContent, err := os.ReadFile(commonStorage)
	suite.NoError(err)
	suite.Equal("export PATH=/usr/local/bin:$PATH", string(commonContent))

	commonLnkFile := filepath.Join(lnkDir, ".lnk")
	commonLnkContent, err := os.ReadFile(commonLnkFile)
	suite.NoError(err)
	suite.Equal(".bashrc\n", string(commonLnkContent))

	// Verify host-specific file storage and tracking
	hostStorage := filepath.Join(lnkDir, "workstation.lnk", ".vimrc")
	suite.FileExists(hostStorage)
	hostContent, err := os.ReadFile(hostStorage)
	suite.NoError(err)
	suite.Equal("set number", string(hostContent))

	hostLnkFile := filepath.Join(lnkDir, ".lnk.workstation")
	hostLnkContent, err := os.ReadFile(hostLnkFile)
	suite.NoError(err)
	suite.Equal(".vimrc\n", string(hostLnkContent))

	// Test list command - common only
	err = suite.runCommand("list")
	suite.NoError(err)
	output = suite.stdout.String()
	suite.Contains(output, "Files managed by lnk (common)")
	suite.Contains(output, ".bashrc")
	suite.NotContains(output, ".vimrc")
	suite.stdout.Reset()

	// Test list command - specific host
	err = suite.runCommand("list", "--host", "workstation")
	suite.NoError(err)
	output = suite.stdout.String()
	suite.Contains(output, "Files managed by lnk (host: workstation)")
	suite.Contains(output, ".vimrc")
	suite.NotContains(output, ".bashrc")
	suite.stdout.Reset()

	// Test list command - all configurations
	err = suite.runCommand("list", "--all")
	suite.NoError(err)
	output = suite.stdout.String()
	suite.Contains(output, "All configurations managed by lnk")
	suite.Contains(output, "Common configuration")
	suite.Contains(output, "Host: workstation")
	suite.Contains(output, ".bashrc")
	suite.Contains(output, ".vimrc")
	suite.stdout.Reset()

	// Test remove from host-specific
	err = suite.runCommand("rm", "--host", "workstation", testFile2)
	suite.NoError(err)
	output = suite.stdout.String()
	suite.Contains(output, "Removed .vimrc from lnk (host: workstation)")
	suite.stdout.Reset()

	// Test remove from common
	err = suite.runCommand("rm", testFile1)
	suite.NoError(err)
	output = suite.stdout.String()
	suite.Contains(output, "Removed .bashrc from lnk")
	suite.NotContains(output, "host:")
	suite.stdout.Reset()

	// Verify files are restored
	info1, err := os.Lstat(testFile1)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info1.Mode()&os.ModeSymlink)

	info2, err := os.Lstat(testFile2)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info2.Mode()&os.ModeSymlink)
}

func (suite *CLITestSuite) TestMultihostErrorHandling() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Try to remove from non-existent host config
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)

	err = suite.runCommand("rm", "--host", "nonexistent", testFile)
	suite.Error(err)
	suite.Contains(err.Error(), "File is not managed by lnk")

	// Try to list non-existent host config
	err = suite.runCommand("list", "--host", "nonexistent")
	suite.NoError(err) // Should not error, just show empty
	output := suite.stdout.String()
	suite.Contains(output, "No files currently managed by lnk (host: nonexistent)")
}

func (suite *CLITestSuite) TestBootstrapCommand() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Test bootstrap command with no script
	err = suite.runCommand("bootstrap")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "No bootstrap script found")
	suite.Contains(output, "bootstrap.sh")
	suite.stdout.Reset()

	// Create a bootstrap script
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	bootstrapScript := filepath.Join(lnkDir, "bootstrap.sh")
	scriptContent := `#!/bin/bash
echo "Bootstrap script executed!"
echo "Working directory: $(pwd)"
touch bootstrap-ran.txt
`
	err = os.WriteFile(bootstrapScript, []byte(scriptContent), 0755)
	suite.Require().NoError(err)

	// Test bootstrap command with script
	err = suite.runCommand("bootstrap")
	suite.NoError(err)
	output = suite.stdout.String()
	suite.Contains(output, "Running bootstrap script")
	suite.Contains(output, "bootstrap.sh")
	suite.Contains(output, "Bootstrap completed successfully")

	// Verify script actually ran
	markerFile := filepath.Join(lnkDir, "bootstrap-ran.txt")
	suite.FileExists(markerFile)
}

func (suite *CLITestSuite) TestInitWithBootstrap() {
	// Create a temporary remote repository with bootstrap script
	remoteDir := filepath.Join(suite.tempDir, "remote")
	err := os.MkdirAll(remoteDir, 0755)
	suite.Require().NoError(err)

	// Initialize git repo in remote with main branch
	cmd := exec.Command("git", "init", "--bare", "--initial-branch=main")
	cmd.Dir = remoteDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// Create a working repo to populate the remote
	workingDir := filepath.Join(suite.tempDir, "working")
	err = os.MkdirAll(workingDir, 0755)
	suite.Require().NoError(err)

	cmd = exec.Command("git", "clone", remoteDir, workingDir)
	err = cmd.Run()
	suite.Require().NoError(err)

	// Add a bootstrap script to the working repo
	bootstrapScript := filepath.Join(workingDir, "bootstrap.sh")
	scriptContent := `#!/bin/bash
echo "Remote bootstrap script executed!"
touch remote-bootstrap-ran.txt
`
	err = os.WriteFile(bootstrapScript, []byte(scriptContent), 0755)
	suite.Require().NoError(err)

	// Add a dummy config file
	configFile := filepath.Join(workingDir, ".bashrc")
	err = os.WriteFile(configFile, []byte("echo 'Hello from remote!'"), 0644)
	suite.Require().NoError(err)

	// Add .lnk file to track the config
	lnkFile := filepath.Join(workingDir, ".lnk")
	err = os.WriteFile(lnkFile, []byte(".bashrc\n"), 0644)
	suite.Require().NoError(err)

	// Commit and push to remote
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = workingDir
	err = cmd.Run()
	suite.Require().NoError(err)

	cmd = exec.Command("git", "-c", "user.email=test@example.com", "-c", "user.name=Test User", "commit", "-m", "Add bootstrap and config")
	cmd.Dir = workingDir
	err = cmd.Run()
	suite.Require().NoError(err)

	cmd = exec.Command("git", "push", "origin", "main")
	cmd.Dir = workingDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// Now test init with remote and automatic bootstrap
	err = suite.runCommand("init", "-r", remoteDir)
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "Cloned from:")
	suite.Contains(output, "Looking for bootstrap script")
	suite.Contains(output, "Found bootstrap script:")
	suite.Contains(output, "bootstrap.sh")
	suite.Contains(output, "Running bootstrap script")
	suite.Contains(output, "Bootstrap completed successfully")

	// Verify bootstrap actually ran
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	markerFile := filepath.Join(lnkDir, "remote-bootstrap-ran.txt")
	suite.FileExists(markerFile)
}

func (suite *CLITestSuite) TestInitWithBootstrapDisabled() {
	// Create a temporary remote repository with bootstrap script
	remoteDir := filepath.Join(suite.tempDir, "remote")
	err := os.MkdirAll(remoteDir, 0755)
	suite.Require().NoError(err)

	// Initialize git repo in remote with main branch
	cmd := exec.Command("git", "init", "--bare", "--initial-branch=main")
	cmd.Dir = remoteDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// Create a working repo to populate the remote
	workingDir := filepath.Join(suite.tempDir, "working")
	err = os.MkdirAll(workingDir, 0755)
	suite.Require().NoError(err)

	cmd = exec.Command("git", "clone", remoteDir, workingDir)
	err = cmd.Run()
	suite.Require().NoError(err)

	// Add a bootstrap script
	bootstrapScript := filepath.Join(workingDir, "bootstrap.sh")
	scriptContent := `#!/bin/bash
echo "This should not run!"
touch should-not-exist.txt
`
	err = os.WriteFile(bootstrapScript, []byte(scriptContent), 0755)
	suite.Require().NoError(err)

	// Commit and push
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = workingDir
	err = cmd.Run()
	suite.Require().NoError(err)

	cmd = exec.Command("git", "-c", "user.email=test@example.com", "-c", "user.name=Test User", "commit", "-m", "Add bootstrap")
	cmd.Dir = workingDir
	err = cmd.Run()
	suite.Require().NoError(err)

	cmd = exec.Command("git", "push", "origin", "main")
	cmd.Dir = workingDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// Test init with --no-bootstrap flag
	err = suite.runCommand("init", "-r", remoteDir, "--no-bootstrap")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "Cloned from:")
	suite.NotContains(output, "Looking for bootstrap script")
	suite.NotContains(output, "Running bootstrap script")

	// Verify bootstrap did NOT run
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	markerFile := filepath.Join(lnkDir, "should-not-exist.txt")
	suite.NoFileExists(markerFile)
}

func (suite *CLITestSuite) TestAddCommandMultipleFiles() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Create multiple test files
	testFile1 := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile1, []byte("export PATH1"), 0644)
	suite.Require().NoError(err)

	testFile2 := filepath.Join(suite.tempDir, ".vimrc")
	err = os.WriteFile(testFile2, []byte("set number"), 0644)
	suite.Require().NoError(err)

	testFile3 := filepath.Join(suite.tempDir, ".gitconfig")
	err = os.WriteFile(testFile3, []byte("[user]\n  name = test"), 0644)
	suite.Require().NoError(err)

	// Test add command with multiple files - should succeed
	err = suite.runCommand("add", testFile1, testFile2, testFile3)
	suite.NoError(err, "Adding multiple files should succeed")

	// Check output shows all files were added
	output := suite.stdout.String()
	suite.Contains(output, "Added 3 items to lnk")
	suite.Contains(output, ".bashrc")
	suite.Contains(output, ".vimrc")
	suite.Contains(output, ".gitconfig")

	// Verify all files are now symlinks
	for _, file := range []string{testFile1, testFile2, testFile3} {
		info, err := os.Lstat(file)
		suite.NoError(err)
		suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)
	}

	// Verify all files exist in storage
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	suite.FileExists(filepath.Join(lnkDir, ".bashrc"))
	suite.FileExists(filepath.Join(lnkDir, ".vimrc"))
	suite.FileExists(filepath.Join(lnkDir, ".gitconfig"))

	// Verify .lnk file contains all entries
	lnkFile := filepath.Join(lnkDir, ".lnk")
	lnkContent, err := os.ReadFile(lnkFile)
	suite.NoError(err)
	suite.Equal(".bashrc\n.gitconfig\n.vimrc\n", string(lnkContent))
}

func (suite *CLITestSuite) TestAddCommandMixedTypes() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Create a file
	testFile := filepath.Join(suite.tempDir, ".vimrc")
	err = os.WriteFile(testFile, []byte("set number"), 0644)
	suite.Require().NoError(err)

	// Create a directory with content
	testDir := filepath.Join(suite.tempDir, ".config", "git")
	err = os.MkdirAll(testDir, 0755)
	suite.Require().NoError(err)
	configFile := filepath.Join(testDir, "config")
	err = os.WriteFile(configFile, []byte("[user]"), 0644)
	suite.Require().NoError(err)

	// Test add command with mixed files and directories - should succeed
	err = suite.runCommand("add", testFile, testDir)
	suite.NoError(err, "Adding mixed files and directories should succeed")

	// Check output shows both items were added
	output := suite.stdout.String()
	suite.Contains(output, "Added 2 items to lnk")
	suite.Contains(output, ".vimrc")
	suite.Contains(output, "git")

	// Verify both are now symlinks
	info1, err := os.Lstat(testFile)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info1.Mode()&os.ModeSymlink)

	info2, err := os.Lstat(testDir)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info2.Mode()&os.ModeSymlink)

	// Verify storage
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	suite.FileExists(filepath.Join(lnkDir, ".vimrc"))
	suite.DirExists(filepath.Join(lnkDir, ".config", "git"))
	suite.FileExists(filepath.Join(lnkDir, ".config", "git", "config"))
}

func (suite *CLITestSuite) TestAddCommandRecursiveFlag() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Create a directory with nested files
	testDir := filepath.Join(suite.tempDir, ".config", "zed")
	err = os.MkdirAll(testDir, 0755)
	suite.Require().NoError(err)

	// Create nested files
	settingsFile := filepath.Join(testDir, "settings.json")
	err = os.WriteFile(settingsFile, []byte(`{"theme": "dark"}`), 0644)
	suite.Require().NoError(err)

	keymapFile := filepath.Join(testDir, "keymap.json")
	err = os.WriteFile(keymapFile, []byte(`{"ctrl+s": "save"}`), 0644)
	suite.Require().NoError(err)

	// Create a subdirectory with files
	themesDir := filepath.Join(testDir, "themes")
	err = os.MkdirAll(themesDir, 0755)
	suite.Require().NoError(err)

	themeFile := filepath.Join(themesDir, "custom.json")
	err = os.WriteFile(themeFile, []byte(`{"colors": {}}`), 0644)
	suite.Require().NoError(err)

	// Test recursive flag - should process directory contents individually
	err = suite.runCommand("add", "--recursive", testDir)
	suite.NoError(err, "Adding directory recursively should succeed")

	// Check output shows multiple files were processed
	output := suite.stdout.String()
	suite.Contains(output, "Added") // Should show some success message

	// Verify individual files are now symlinks (not the directory itself)
	info, err := os.Lstat(settingsFile)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink, "settings.json should be a symlink")

	info, err = os.Lstat(keymapFile)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink, "keymap.json should be a symlink")

	info, err = os.Lstat(themeFile)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink, "custom.json should be a symlink")

	// The directory itself should NOT be a symlink
	info, err = os.Lstat(testDir)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink, "Directory should not be a symlink")

	// Verify files exist individually in storage
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	suite.FileExists(filepath.Join(lnkDir, ".config", "zed", "settings.json"))
	suite.FileExists(filepath.Join(lnkDir, ".config", "zed", "keymap.json"))
	suite.FileExists(filepath.Join(lnkDir, ".config", "zed", "themes", "custom.json"))
}

func (suite *CLITestSuite) TestAddCommandRecursiveMultipleDirs() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Create two directories with files
	dir1 := filepath.Join(suite.tempDir, "dir1")
	dir2 := filepath.Join(suite.tempDir, "dir2")
	err = os.MkdirAll(dir1, 0755)
	suite.Require().NoError(err)
	err = os.MkdirAll(dir2, 0755)
	suite.Require().NoError(err)

	// Create files in each directory
	file1 := filepath.Join(dir1, "file1.txt")
	file2 := filepath.Join(dir2, "file2.txt")
	err = os.WriteFile(file1, []byte("content1"), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(file2, []byte("content2"), 0644)
	suite.Require().NoError(err)

	// Test recursive flag with multiple directories
	err = suite.runCommand("add", "--recursive", dir1, dir2)
	suite.NoError(err, "Adding multiple directories recursively should succeed")

	// Verify both files are symlinks
	info, err := os.Lstat(file1)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink, "file1.txt should be a symlink")

	info, err = os.Lstat(file2)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink, "file2.txt should be a symlink")

	// Verify directories are not symlinks
	info, err = os.Lstat(dir1)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink, "dir1 should not be a symlink")

	info, err = os.Lstat(dir2)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink, "dir2 should not be a symlink")
}

// Task 3.1: Dry-Run Mode Tests

func (suite *CLITestSuite) TestDryRunFlag() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.NoError(err)
	initOutput := suite.stdout.String()
	suite.Contains(initOutput, "Initialized")
	suite.stdout.Reset()

	// Create test files
	testFile1 := filepath.Join(suite.tempDir, "test1.txt")
	testFile2 := filepath.Join(suite.tempDir, "test2.txt")
	suite.Require().NoError(os.WriteFile(testFile1, []byte("content1"), 0644))
	suite.Require().NoError(os.WriteFile(testFile2, []byte("content2"), 0644))

	// Run add with dry-run flag (should not exist yet)
	err = suite.runCommand("add", "--dry-run", testFile1, testFile2)
	suite.NoError(err, "Dry-run command should succeed")
	output := suite.stdout.String()

	// Basic check that some output was produced (flag exists but behavior TBD)
	suite.NotEmpty(output, "Should produce some output")

	// Verify files were NOT actually added (no symlinks created)
	info, err := os.Lstat(testFile1)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink, "File should not be a symlink in dry-run")

	info, err = os.Lstat(testFile2)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink, "File should not be a symlink in dry-run")

	// Verify lnk list shows no managed files
	suite.stdout.Reset()
	err = suite.runCommand("list")
	suite.NoError(err)
	listOutput := suite.stdout.String()
	suite.NotContains(listOutput, "test1.txt", "Files should not be managed after dry-run")
	suite.NotContains(listOutput, "test2.txt", "Files should not be managed after dry-run")
}

func (suite *CLITestSuite) TestDryRunOutput() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.NoError(err)
	initOutput := suite.stdout.String()
	suite.Contains(initOutput, "Initialized")
	suite.stdout.Reset()

	// Create test files
	testFile1 := filepath.Join(suite.tempDir, "test1.txt")
	testFile2 := filepath.Join(suite.tempDir, "test2.txt")
	suite.Require().NoError(os.WriteFile(testFile1, []byte("content1"), 0644))
	suite.Require().NoError(os.WriteFile(testFile2, []byte("content2"), 0644))

	// Run add with dry-run flag
	err = suite.runCommand("add", "--dry-run", testFile1, testFile2)
	suite.NoError(err, "Dry-run command should succeed")
	output := suite.stdout.String()

	// Verify dry-run shows preview of what would be added
	suite.Contains(output, "Would add", "Should show dry-run preview")
	suite.Contains(output, "test1.txt", "Should show first file")
	suite.Contains(output, "test2.txt", "Should show second file")
	suite.Contains(output, "2 files", "Should show file count")

	// Should contain helpful instructions
	suite.Contains(output, "run without --dry-run", "Should provide next steps")
}

func (suite *CLITestSuite) TestDryRunRecursive() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.NoError(err)
	initOutput := suite.stdout.String()
	suite.Contains(initOutput, "Initialized")
	suite.stdout.Reset()

	// Create directory structure with multiple files
	configDir := filepath.Join(suite.tempDir, ".config", "test-app")
	err = os.MkdirAll(configDir, 0755)
	suite.Require().NoError(err)

	// Create files in directory
	for i := 1; i <= 15; i++ {
		file := filepath.Join(configDir, fmt.Sprintf("config%d.json", i))
		suite.Require().NoError(os.WriteFile(file, []byte(fmt.Sprintf("config %d", i)), 0644))
	}

	// Run recursive add with dry-run
	err = suite.runCommand("add", "--dry-run", "--recursive", configDir)
	suite.NoError(err, "Dry-run recursive command should succeed")
	output := suite.stdout.String()

	// Verify dry-run shows all files that would be added
	suite.Contains(output, "Would add", "Should show dry-run preview")
	suite.Contains(output, "15 files", "Should show correct file count")
	suite.Contains(output, "recursively", "Should indicate recursive mode")

	// Should show some of the files
	suite.Contains(output, "config1.json", "Should show first file")
	suite.Contains(output, "config15.json", "Should show last file")

	// Verify no actual changes were made
	for i := 1; i <= 15; i++ {
		file := filepath.Join(configDir, fmt.Sprintf("config%d.json", i))
		info, err := os.Lstat(file)
		suite.NoError(err)
		suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink, "File should not be symlink after dry-run")
	}
}

// Task 3.2: Enhanced Output and Messaging Tests

func (suite *CLITestSuite) TestEnhancedSuccessOutput() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.NoError(err)
	suite.stdout.Reset()

	// Create multiple test files
	testFiles := []string{
		filepath.Join(suite.tempDir, "config1.txt"),
		filepath.Join(suite.tempDir, "config2.txt"),
		filepath.Join(suite.tempDir, "config3.txt"),
	}

	for i, file := range testFiles {
		suite.Require().NoError(os.WriteFile(file, []byte(fmt.Sprintf("content %d", i+1)), 0644))
	}

	// Add multiple files
	args := append([]string{"add"}, testFiles...)
	err = suite.runCommand(args...)
	suite.NoError(err)
	output := suite.stdout.String()

	// Should have enhanced formatting with consistent indentation
	suite.Contains(output, "🔗", "Should use link icons")
	suite.Contains(output, "config1.txt", "Should show first file")
	suite.Contains(output, "config2.txt", "Should show second file")
	suite.Contains(output, "config3.txt", "Should show third file")

	// Should show organized file list
	suite.Contains(output, "   ", "Should have consistent indentation")

	// Should include summary information
	suite.Contains(output, "3 items", "Should show total count")
}

func (suite *CLITestSuite) TestOperationSummary() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.NoError(err)
	suite.stdout.Reset()

	// Create directory with files for recursive operation
	configDir := filepath.Join(suite.tempDir, ".config", "test-app")
	err = os.MkdirAll(configDir, 0755)
	suite.Require().NoError(err)

	// Create files in directory
	for i := 1; i <= 5; i++ {
		file := filepath.Join(configDir, fmt.Sprintf("file%d.json", i))
		suite.Require().NoError(os.WriteFile(file, []byte(fmt.Sprintf("content %d", i)), 0644))
	}

	// Add recursively
	err = suite.runCommand("add", "--recursive", configDir)
	suite.NoError(err)
	output := suite.stdout.String()

	// Should show operation summary
	suite.Contains(output, "recursively", "Should indicate operation type")
	suite.Contains(output, "5", "Should show correct file count")

	// Should include contextual help message
	suite.Contains(output, "lnk push", "Should suggest next steps")
	suite.Contains(output, "sync to remote", "Should explain next step purpose")

	// Should show operation completion confirmation
	suite.Contains(output, "✨", "Should use success emoji")
	suite.Contains(output, "Added", "Should confirm operation completed")
}

// Task 3.3: Documentation and Help Updates Tests

func (suite *CLITestSuite) TestUpdatedHelpText() {
	// Test main help
	err := suite.runCommand("help")
	suite.NoError(err)
	helpOutput := suite.stdout.String()
	suite.stdout.Reset()

	// Should mention bulk operations
	suite.Contains(helpOutput, "multiple files", "Help should mention multiple file support")

	// Test add command help
	err = suite.runCommand("add", "--help")
	suite.NoError(err)
	addHelpOutput := suite.stdout.String()

	// Should include new flags
	suite.Contains(addHelpOutput, "--recursive", "Help should include recursive flag")
	suite.Contains(addHelpOutput, "--dry-run", "Help should include dry-run flag")

	// Should include examples
	suite.Contains(addHelpOutput, "Examples:", "Help should include usage examples")
	suite.Contains(addHelpOutput, "lnk add ~/.bashrc ~/.vimrc", "Help should show multiple file example")
	suite.Contains(addHelpOutput, "lnk add --recursive ~/.config", "Help should show recursive example")
	suite.Contains(addHelpOutput, "lnk add --dry-run", "Help should show dry-run example")

	// Should describe what each flag does
	suite.Contains(addHelpOutput, "directory contents individually", "Should explain recursive flag")
	suite.Contains(addHelpOutput, "without making changes", "Should explain dry-run flag")
}

// Task 3.1: Tests for force flag functionality
func (suite *CLITestSuite) TestInitCmd_ForceFlag_BypassesSafetyCheck() {
	// Setup: Create .lnk file to simulate existing content
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	err := os.MkdirAll(lnkDir, 0755)
	suite.Require().NoError(err)

	// Initialize git repo first
	cmd := exec.Command("git", "init")
	cmd.Dir = lnkDir
	err = cmd.Run()
	suite.Require().NoError(err)

	lnkFile := filepath.Join(lnkDir, ".lnk")
	err = os.WriteFile(lnkFile, []byte(".bashrc\n"), 0644)
	suite.Require().NoError(err)

	// Create a dummy remote directory for testing
	remoteDir := filepath.Join(suite.tempDir, "remote")
	err = os.MkdirAll(remoteDir, 0755)
	suite.Require().NoError(err)
	cmd = exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// Execute init command with --force flag
	err = suite.runCommand("init", "-r", remoteDir, "--force")
	suite.NoError(err, "Force flag should bypass safety check")

	// Verify output shows warning
	output := suite.stdout.String()
	suite.Contains(output, "force", "Should show force warning")
}

func (suite *CLITestSuite) TestInitCmd_NoForceFlag_RespectsSafetyCheck() {
	// Setup: Create .lnk file to simulate existing content
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	err := os.MkdirAll(lnkDir, 0755)
	suite.Require().NoError(err)

	// Initialize git repo first
	cmd := exec.Command("git", "init")
	cmd.Dir = lnkDir
	err = cmd.Run()
	suite.Require().NoError(err)

	lnkFile := filepath.Join(lnkDir, ".lnk")
	err = os.WriteFile(lnkFile, []byte(".bashrc\n"), 0644)
	suite.Require().NoError(err)

	// Create a dummy remote directory for testing
	remoteDir := filepath.Join(suite.tempDir, "remote")
	err = os.MkdirAll(remoteDir, 0755)
	suite.Require().NoError(err)
	cmd = exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// Execute init command without --force flag - should fail
	err = suite.runCommand("init", "-r", remoteDir)
	suite.Error(err, "Should respect safety check without force flag")
	suite.Contains(err.Error(), "already contains managed files")
}

func (suite *CLITestSuite) TestInitCmd_ForceFlag_ShowsWarning() {
	// Setup: Create .lnk file to simulate existing content
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	err := os.MkdirAll(lnkDir, 0755)
	suite.Require().NoError(err)

	// Initialize git repo first
	cmd := exec.Command("git", "init")
	cmd.Dir = lnkDir
	err = cmd.Run()
	suite.Require().NoError(err)

	lnkFile := filepath.Join(lnkDir, ".lnk")
	err = os.WriteFile(lnkFile, []byte(".bashrc\n"), 0644)
	suite.Require().NoError(err)

	// Create a dummy remote directory for testing
	remoteDir := filepath.Join(suite.tempDir, "remote")
	err = os.MkdirAll(remoteDir, 0755)
	suite.Require().NoError(err)
	cmd = exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// Execute init command with --force flag
	err = suite.runCommand("init", "-r", remoteDir, "--force")
	suite.NoError(err, "Force flag should bypass safety check")

	// Verify output shows appropriate warning
	output := suite.stdout.String()
	suite.Contains(output, "⚠️", "Should show warning emoji")
	suite.Contains(output, "overwrite", "Should warn about overwriting")
}

// Task 4.1: Integration tests for end-to-end workflows
func (suite *CLITestSuite) TestE2E_InitAddInit_PreventDataLoss() {
	// Run: lnk init
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Create and add test file
	testFile := filepath.Join(suite.tempDir, ".testfile")
	err = os.WriteFile(testFile, []byte("important content"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile)
	suite.Require().NoError(err)

	// Create dummy remote for testing
	remoteDir := filepath.Join(suite.tempDir, "remote")
	err = os.MkdirAll(remoteDir, 0755)
	suite.Require().NoError(err)
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// Run: lnk init -r <remote> → should FAIL
	err = suite.runCommand("init", "-r", remoteDir)
	suite.Error(err, "Should prevent data loss")
	suite.Contains(err.Error(), "already contains managed files")

	// Verify testfile still exists and is managed
	suite.FileExists(testFile)
	info, err := os.Lstat(testFile)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink, "File should still be symlink")
}

func (suite *CLITestSuite) TestE2E_FreshInit_Success() {
	// Create dummy remote for testing
	remoteDir := filepath.Join(suite.tempDir, "remote")
	err := os.MkdirAll(remoteDir, 0755)
	suite.Require().NoError(err)
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// Fresh init with remote should succeed
	err = suite.runCommand("init", "-r", remoteDir)
	suite.NoError(err, "Fresh init should succeed")

	// Verify repository was created
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	suite.DirExists(lnkDir)
	gitDir := filepath.Join(lnkDir, ".git")
	suite.DirExists(gitDir)

	// Verify success message
	output := suite.stdout.String()
	suite.Contains(output, "Initialized lnk repository")
	suite.Contains(output, "Cloned from:")
}

func (suite *CLITestSuite) TestE2E_ForceInit_OverwritesContent() {
	// Setup: init and add content first
	err := suite.runCommand("init")
	suite.Require().NoError(err)

	testFile := filepath.Join(suite.tempDir, ".testfile")
	err = os.WriteFile(testFile, []byte("original content"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Create dummy remote for testing
	remoteDir := filepath.Join(suite.tempDir, "remote")
	err = os.MkdirAll(remoteDir, 0755)
	suite.Require().NoError(err)
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// Force init should succeed and show warning
	err = suite.runCommand("init", "-r", remoteDir, "--force")
	suite.NoError(err, "Force init should succeed")

	// Verify warning was shown
	output := suite.stdout.String()
	suite.Contains(output, "⚠️", "Should show warning")
	suite.Contains(output, "overwrite", "Should warn about overwriting")
	suite.Contains(output, "Initialized lnk repository")
}

func (suite *CLITestSuite) TestE2E_ErrorMessage_SuggestsCorrectCommand() {
	// Setup: init and add content first
	err := suite.runCommand("init")
	suite.Require().NoError(err)

	testFile := filepath.Join(suite.tempDir, ".testfile")
	err = os.WriteFile(testFile, []byte("important content"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile)
	suite.Require().NoError(err)

	// Try init with remote - should fail with helpful message
	err = suite.runCommand("init", "-r", "https://github.com/test/dotfiles.git")
	suite.Error(err, "Should fail with helpful error")

	// Verify error message suggests correct alternative
	suite.Contains(err.Error(), "already contains managed files", "Should explain the problem")
	suite.Contains(err.Error(), "lnk pull", "Should suggest pull command")
	suite.Contains(err.Error(), "instead of", "Should explain the alternative")
	suite.Contains(err.Error(), "lnk init -r", "Should show the problematic command")
}

// Task 6.1: Regression tests to ensure existing functionality unchanged
func (suite *CLITestSuite) TestRegression_FreshInit_UnchangedBehavior() {
	// Test that fresh init (no existing content) works exactly as before
	err := suite.runCommand("init")
	suite.NoError(err, "Fresh init should work unchanged")

	// Verify same output format and behavior
	output := suite.stdout.String()
	suite.Contains(output, "Initialized empty lnk repository")
	suite.Contains(output, "Location:")

	// Verify repository structure is created correctly
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	suite.DirExists(lnkDir)
	gitDir := filepath.Join(lnkDir, ".git")
	suite.DirExists(gitDir)
}

func (suite *CLITestSuite) TestRegression_ExistingWorkflows_StillWork() {
	// Test that all existing workflows continue to function

	// 1. Normal init → add → list → remove workflow
	err := suite.runCommand("init")
	suite.NoError(err, "Init should work")
	suite.stdout.Reset()

	// Create and add a file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)

	err = suite.runCommand("add", testFile)
	suite.NoError(err, "Add should work")
	suite.stdout.Reset()

	// List files
	err = suite.runCommand("list")
	suite.NoError(err, "List should work")
	output := suite.stdout.String()
	suite.Contains(output, ".bashrc", "Should list added file")
	suite.stdout.Reset()

	// Remove file
	err = suite.runCommand("rm", testFile)
	suite.NoError(err, "Remove should work")

	// Verify file is restored as regular file
	info, err := os.Lstat(testFile)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink, "File should be regular after remove")
}

func (suite *CLITestSuite) TestRegression_GitOperations_Unaffected() {
	// Test that Git operations continue to work normally
	err := suite.runCommand("init")
	suite.NoError(err)

	// Add a file to create commits
	testFile := filepath.Join(suite.tempDir, ".vimrc")
	err = os.WriteFile(testFile, []byte("set number"), 0644)
	suite.Require().NoError(err)

	err = suite.runCommand("add", testFile)
	suite.NoError(err)

	// Verify Git repository structure and commits are normal
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")

	// Check that commits are created normally
	cmd := exec.Command("git", "log", "--oneline", "--format=%s")
	cmd.Dir = lnkDir
	output, err := cmd.Output()
	suite.NoError(err, "Git log should work")

	commits := string(output)
	suite.Contains(commits, "lnk: added .vimrc", "Should have normal commit message")

	// Check that git status works
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = lnkDir
	statusOutput, err := cmd.Output()
	suite.NoError(err, "Git status should work")
	suite.Empty(strings.TrimSpace(string(statusOutput)), "Working directory should be clean")
}

func (suite *CLITestSuite) TestRegression_PerformanceImpact_Minimal() {
	// Test that the new safety checks don't significantly impact performance

	// Simple performance check: ensure a single init completes quickly
	start := time.Now()
	err := suite.runCommand("init")
	elapsed := time.Since(start)

	suite.NoError(err, "Init should succeed")
	suite.Less(elapsed, 2*time.Second, "Init should complete quickly")

	// Test safety check performance on existing repository
	suite.stdout.Reset()
	start = time.Now()
	err = suite.runCommand("init", "-r", "dummy-url")
	elapsed = time.Since(start)

	// Should fail quickly due to safety check (not hang)
	suite.Error(err, "Should fail due to safety check")
	suite.Less(elapsed, 1*time.Second, "Safety check should be fast")
}

// Task 7.1: Tests for help documentation
func (suite *CLITestSuite) TestInitCommand_HelpText_MentionsForceFlag() {
	err := suite.runCommand("init", "--help")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "--force", "Help should mention force flag")
	suite.Contains(output, "overwrite", "Help should explain force behavior")
}

func (suite *CLITestSuite) TestInitCommand_HelpText_ExplainsDataProtection() {
	err := suite.runCommand("init", "--help")
	suite.NoError(err)
	output := suite.stdout.String()

	// Should explain what the command does
	suite.Contains(output, "Creates", "Should explain what init does")
	suite.Contains(output, "lnk directory", "Should mention lnk directory")

	// Should warn about the force flag risks
	suite.Contains(output, "WARNING", "Should warn about force flag risks")
	suite.Contains(output, "overwrite existing content", "Should mention overwrite risk")
}

// TestPushPullWithDifferentBranches tests push/pull operations with different default branch names
func (suite *CLITestSuite) TestPushPullWithDifferentBranches() {
	testCases := []struct {
		name        string
		branchName  string
		setupRemote func(remoteDir string) error
	}{
		{
			name:       "master branch",
			branchName: "master",
			setupRemote: func(remoteDir string) error {
				cmd := exec.Command("git", "init", "--bare", "--initial-branch=master")
				cmd.Dir = remoteDir
				return cmd.Run()
			},
		},
		{
			name:       "main branch",
			branchName: "main",
			setupRemote: func(remoteDir string) error {
				cmd := exec.Command("git", "init", "--bare", "--initial-branch=main")
				cmd.Dir = remoteDir
				return cmd.Run()
			},
		},
		{
			name:       "custom branch",
			branchName: "develop",
			setupRemote: func(remoteDir string) error {
				cmd := exec.Command("git", "init", "--bare", "--initial-branch=develop")
				cmd.Dir = remoteDir
				return cmd.Run()
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create a separate temp directory for this test case
			testDir, err := os.MkdirTemp("", "lnk-push-pull-test-*")
			suite.Require().NoError(err)
			defer func() { _ = os.RemoveAll(testDir) }()

			// Save current dir and change to test dir
			originalDir, err := os.Getwd()
			suite.Require().NoError(err)
			defer func() { _ = os.Chdir(originalDir) }()

			err = os.Chdir(testDir)
			suite.Require().NoError(err)

			// Set HOME to test directory
			suite.T().Setenv("HOME", testDir)
			suite.T().Setenv("XDG_CONFIG_HOME", testDir)

			// Create remote repository
			remoteDir := filepath.Join(testDir, "remote.git")
			err = os.MkdirAll(remoteDir, 0755)
			suite.Require().NoError(err)

			err = tc.setupRemote(remoteDir)
			suite.Require().NoError(err)

			// Initialize lnk with remote
			err = suite.runCommand("init", "--remote", remoteDir)
			suite.Require().NoError(err)

			// Switch to the test branch if not main/master (since init creates main by default)
			if tc.branchName != "main" {
				lnkDir := filepath.Join(testDir, "lnk")
				cmd := exec.Command("git", "checkout", "-b", tc.branchName)
				cmd.Dir = lnkDir
				_, err = cmd.CombinedOutput()
				suite.Require().NoError(err)
			}

			// Add a test file
			testFile := filepath.Join(testDir, ".testrc")
			err = os.WriteFile(testFile, []byte("test config"), 0644)
			suite.Require().NoError(err)

			err = suite.runCommand("add", testFile)
			suite.Require().NoError(err)

			// Test push operation
			err = suite.runCommand("push", "test push with "+tc.branchName)
			suite.Require().NoError(err, "Push should work with %s branch", tc.branchName)

			// Create another test directory to simulate pulling from another machine
			pullTestDir, err := os.MkdirTemp("", "lnk-pull-test-*")
			suite.Require().NoError(err)
			defer func() { _ = os.RemoveAll(pullTestDir) }()

			err = os.Chdir(pullTestDir)
			suite.Require().NoError(err)

			// Set HOME for pull test
			suite.T().Setenv("HOME", pullTestDir)
			suite.T().Setenv("XDG_CONFIG_HOME", pullTestDir)

			// Clone and test pull
			err = suite.runCommand("init", "--remote", remoteDir)
			suite.Require().NoError(err)

			err = suite.runCommand("pull")
			suite.Require().NoError(err, "Pull should work with %s branch", tc.branchName)

			// Verify the file was pulled correctly
			lnkDir := filepath.Join(pullTestDir, "lnk")
			pulledFile := filepath.Join(lnkDir, ".testrc")
			suite.FileExists(pulledFile, "File should exist after pull with %s branch", tc.branchName)

			content, err := os.ReadFile(pulledFile)
			suite.Require().NoError(err)
			suite.Equal("test config", string(content), "File content should match after pull with %s branch", tc.branchName)
		})
	}
}

func (suite *CLITestSuite) TestDiffCommand_NotInitialized() {
	// Test diff without init - should fail
	err := suite.runCommand("diff")
	suite.Error(err)
	suite.Contains(err.Error(), "Lnk repository not initialized")
}

func (suite *CLITestSuite) TestDiffCommand_NoChanges() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add a file so the repo has commits
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Test diff with no uncommitted changes
	err = suite.runCommand("diff")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "No uncommitted changes")
	suite.Contains(output, "dotfiles are clean")
}

func (suite *CLITestSuite) TestDiffCommand_WithChanges() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add a file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Modify the managed file (it's now a symlink into the repo)
	// The symlink points into the lnk repo, so writing to the symlink modifies the repo file
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH\nexport EDITOR=vim"), 0644)
	suite.Require().NoError(err)

	// Test diff with uncommitted changes
	err = suite.runCommand("diff")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "EDITOR=vim", "Diff should show the changed content")
	suite.Contains(output, ".bashrc", "Diff should reference the changed file")
}

func (suite *CLITestSuite) TestDoctorCommand_NotInitialized() {
	err := suite.runCommand("doctor")
	suite.Error(err)
	suite.Contains(err.Error(), "Lnk repository not initialized")
}

func (suite *CLITestSuite) TestDoctorCommand_AllValid() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add a file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Run doctor — everything should be healthy
	err = suite.runCommand("doctor")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "Repository is healthy")
}

func (suite *CLITestSuite) TestDoctorCommand_RemovesInvalidEntries() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add a file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add a second file, then delete it from the repo to create an invalid entry
	testFile2 := filepath.Join(suite.tempDir, ".vimrc")
	err = os.WriteFile(testFile2, []byte("set number"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile2)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Delete the stored file from repo (simulating manual deletion / corruption)
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	repoVimrc := filepath.Join(lnkDir, ".vimrc")
	err = os.Remove(repoVimrc)
	suite.Require().NoError(err)

	// Run doctor
	err = suite.runCommand("doctor")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "Removed 1 invalid entry")
	suite.Contains(output, ".vimrc")
	suite.Contains(output, "lnk push")

	// Verify .lnk file was cleaned
	lnkFile := filepath.Join(lnkDir, ".lnk")
	lnkContent, err := os.ReadFile(lnkFile)
	suite.NoError(err)
	suite.Equal(".bashrc\n", string(lnkContent))

	// Verify git commit was made
	gitCmd := exec.Command("git", "log", "--oneline", "--format=%s", "-1")
	gitCmd.Dir = lnkDir
	commitOutput, err := gitCmd.Output()
	suite.NoError(err)
	suite.Contains(string(commitOutput), "lnk: cleaned 1 invalid")
}

func (suite *CLITestSuite) TestDoctorCommand_WithHost() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add a host-specific file
	testFile := filepath.Join(suite.tempDir, ".vimrc")
	err = os.WriteFile(testFile, []byte("set number"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", "--host", "work", testFile)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add a second host-specific file, then delete it from repo
	testFile2 := filepath.Join(suite.tempDir, ".zshrc")
	err = os.WriteFile(testFile2, []byte("# zsh"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", "--host", "work", testFile2)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Delete the stored file from repo
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	repoZshrc := filepath.Join(lnkDir, "work.lnk", ".zshrc")
	err = os.Remove(repoZshrc)
	suite.Require().NoError(err)

	// Run doctor with --host flag
	err = suite.runCommand("doctor", "--host", "work")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "host: work")
	suite.Contains(output, ".zshrc")
	suite.Contains(output, "invalid")

	// Verify .lnk.work was cleaned
	hostLnkFile := filepath.Join(lnkDir, ".lnk.work")
	hostContent, err := os.ReadFile(hostLnkFile)
	suite.NoError(err)
	suite.Equal(".vimrc\n", string(hostContent))
}

func (suite *CLITestSuite) TestDoctorCommand_EmptyRepo() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Run doctor on repo with no managed files
	err = suite.runCommand("doctor")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "Repository is healthy")
}

func (suite *CLITestSuite) TestDoctorCommand_HelpText() {
	err := suite.runCommand("doctor", "--help")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "Scans the lnk repository")
	suite.Contains(output, "--host")
	suite.Contains(output, "--dry-run")
}

func (suite *CLITestSuite) TestDoctorCommand_DryRun_AllValid() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add a file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Run doctor --dry-run — everything should be healthy
	err = suite.runCommand("doctor", "--dry-run")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "Repository is healthy")
}

func (suite *CLITestSuite) TestDoctorCommand_DryRun_ShowsInvalidEntries() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add two files
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile)
	suite.Require().NoError(err)

	testFile2 := filepath.Join(suite.tempDir, ".vimrc")
	err = os.WriteFile(testFile2, []byte("set number"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile2)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Delete .vimrc from repo to create invalid entry
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	repoVimrc := filepath.Join(lnkDir, ".vimrc")
	err = os.Remove(repoVimrc)
	suite.Require().NoError(err)

	// Run doctor --dry-run
	err = suite.runCommand("doctor", "--dry-run")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "Would remove 1 invalid entry")
	suite.Contains(output, ".vimrc")
	suite.Contains(output, "run without --dry-run")

	// Verify NO actual changes were made — .lnk file still contains both entries
	lnkFile := filepath.Join(lnkDir, ".lnk")
	lnkContent, err := os.ReadFile(lnkFile)
	suite.NoError(err)
	suite.Contains(string(lnkContent), ".bashrc")
	suite.Contains(string(lnkContent), ".vimrc")
}

func (suite *CLITestSuite) TestDoctorCommand_DryRun_WithHost() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add a host-specific file
	testFile := filepath.Join(suite.tempDir, ".vimrc")
	err = os.WriteFile(testFile, []byte("set number"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", "--host", "work", testFile)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Delete from host storage
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	repoVimrc := filepath.Join(lnkDir, "work.lnk", ".vimrc")
	err = os.Remove(repoVimrc)
	suite.Require().NoError(err)

	// Run doctor --dry-run --host work
	err = suite.runCommand("doctor", "--dry-run", "--host", "work")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "host: work")
	suite.Contains(output, ".vimrc")
	suite.Contains(output, "invalid")
	suite.Contains(output, "run without --dry-run")

	// Verify NO actual changes — .lnk.work still contains entry
	hostLnkFile := filepath.Join(lnkDir, ".lnk.work")
	hostContent, err := os.ReadFile(hostLnkFile)
	suite.NoError(err)
	suite.Contains(string(hostContent), ".vimrc")
}

func (suite *CLITestSuite) TestDoctorCommand_DryRun_EmptyRepo() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Run doctor --dry-run on repo with no managed files
	err = suite.runCommand("doctor", "--dry-run")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "Repository is healthy")
}

func (suite *CLITestSuite) TestDoctorCommand_DryRun_MultipleInvalid() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add three files
	files := []string{".bashrc", ".vimrc", ".zshrc"}
	for _, name := range files {
		testFile := filepath.Join(suite.tempDir, name)
		err = os.WriteFile(testFile, []byte("content"), 0644)
		suite.Require().NoError(err)
		err = suite.runCommand("add", testFile)
		suite.Require().NoError(err)
	}
	suite.stdout.Reset()

	// Delete two from repo
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	err = os.Remove(filepath.Join(lnkDir, ".vimrc"))
	suite.Require().NoError(err)
	err = os.Remove(filepath.Join(lnkDir, ".zshrc"))
	suite.Require().NoError(err)

	// Run doctor --dry-run
	err = suite.runCommand("doctor", "--dry-run")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "Would remove 2 invalid entries")
	suite.Contains(output, ".vimrc")
	suite.Contains(output, ".zshrc")
	suite.NotContains(output, "Fixed") // Should NOT use "Fixed" in dry-run
}

func (suite *CLITestSuite) TestDoctorCommand_BrokenSymlinks() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add a file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Remove the symlink to create a broken symlink scenario
	err = os.Remove(testFile)
	suite.Require().NoError(err)

	// Run doctor
	err = suite.runCommand("doctor")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "Restored 1 broken symlink")
	suite.Contains(output, ".bashrc")

	// Verify symlink was restored
	info, err := os.Lstat(testFile)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)
}

func (suite *CLITestSuite) TestDoctorCommand_DryRun_BrokenSymlinks() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add a file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Remove the symlink
	err = os.Remove(testFile)
	suite.Require().NoError(err)

	// Run doctor --dry-run
	err = suite.runCommand("doctor", "--dry-run")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "Would fix 1 broken symlink")
	suite.Contains(output, ".bashrc")

	// Verify NO actual fix — symlink should still be missing
	_, err = os.Lstat(testFile)
	suite.True(os.IsNotExist(err))
}

func (suite *CLITestSuite) TestDoctorCommand_OrphanedFiles() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add a file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Create an orphaned file in the repo
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	orphanFile := filepath.Join(lnkDir, "orphan.txt")
	err = os.WriteFile(orphanFile, []byte("orphaned"), 0644)
	suite.Require().NoError(err)

	// Stage and commit the orphan so git knows about it
	gitCmd := exec.Command("git", "add", "orphan.txt")
	gitCmd.Dir = lnkDir
	err = gitCmd.Run()
	suite.Require().NoError(err)
	gitCmd = exec.Command("git", "-c", "user.email=test@test.com", "-c", "user.name=Test", "commit", "-m", "add orphan")
	gitCmd.Dir = lnkDir
	err = gitCmd.Run()
	suite.Require().NoError(err)

	// Run doctor
	err = suite.runCommand("doctor")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "Removed 1 orphaned file")
	suite.Contains(output, "orphan.txt")

	// Verify orphan was deleted
	_, err = os.Stat(orphanFile)
	suite.True(os.IsNotExist(err))
}

func (suite *CLITestSuite) TestDoctorCommand_DryRun_OrphanedFiles() {
	// Initialize repository
	err := suite.runCommand("init")
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Add a file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.runCommand("add", testFile)
	suite.Require().NoError(err)
	suite.stdout.Reset()

	// Create an orphaned file in the repo
	lnkDir := filepath.Join(suite.tempDir, ".config", "lnk")
	orphanFile := filepath.Join(lnkDir, "orphan.txt")
	err = os.WriteFile(orphanFile, []byte("orphaned"), 0644)
	suite.Require().NoError(err)

	// Run doctor --dry-run
	err = suite.runCommand("doctor", "--dry-run")
	suite.NoError(err)
	output := suite.stdout.String()
	suite.Contains(output, "Would remove 1 orphaned file")
	suite.Contains(output, "orphan.txt")

	// Verify NO actual deletion — orphan still exists
	suite.FileExists(orphanFile)
}

func TestCLISuite(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
}
