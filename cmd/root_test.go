package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

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

	// Set XDG_CONFIG_HOME to temp directory
	suite.T().Setenv("XDG_CONFIG_HOME", tempDir)

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
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	suite.DirExists(lnkDir)

	gitDir := filepath.Join(lnkDir, ".git")
	suite.DirExists(gitDir)
}

func (suite *CLITestSuite) TestInitWithRemote() {
	err := suite.runCommand("init", "-r", "https://github.com/user/dotfiles.git")
	// This will fail because we don't have a real remote, but that's expected
	suite.Error(err)
	suite.Contains(err.Error(), "git clone failed")
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
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	repoFile := filepath.Join(lnkDir, suite.tempDir, ".bashrc")
	suite.FileExists(repoFile)
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
	suite.Contains(err.Error(), "no remote configured")
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
			errContains: "File does not exist",
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
			outContains: "Moves a file to the lnk repository",
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
			args: []string{"add", ".bashrc"},
			setup: func() {
				testFile := filepath.Join(suite.tempDir, ".bashrc")
				_ = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
			},
			verify: func(output string) {
				suite.Contains(output, "Added .bashrc to lnk")
			},
		},
		{
			name: "add another file",
			args: []string{"add", ".vimrc"},
			setup: func() {
				testFile := filepath.Join(suite.tempDir, ".vimrc")
				_ = os.WriteFile(testFile, []byte("set number"), 0644)
			},
			verify: func(output string) {
				suite.Contains(output, "Added .vimrc to lnk")
			},
		},
		{
			name: "remove file",
			args: []string{"rm", ".vimrc"},
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
	testDir := filepath.Join(suite.tempDir, ".config")
	_ = os.MkdirAll(testDir, 0755)
	configFile := filepath.Join(testDir, "app.conf")
	_ = os.WriteFile(configFile, []byte("setting=value"), 0644)

	// Add the directory
	err := suite.runCommand("add", testDir)
	suite.NoError(err)

	// Check output
	output := suite.stdout.String()
	suite.Contains(output, "Added .config to lnk")

	// Verify directory is now a symlink
	info, err := os.Lstat(testDir)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)

	// Verify the directory exists in repo with preserved directory structure
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	repoDir := filepath.Join(lnkDir, suite.tempDir, ".config")
	suite.DirExists(repoDir)
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

	// Add a remote so status works
	lnkDir := filepath.Join(suite.tempDir, "lnk")
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

func TestCLISuite(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
}
