package cmd

import (
	"bytes"
	"os"
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

	// Verify file exists in repo
	repoFile := filepath.Join(suite.tempDir, "lnk", ".bashrc")
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

	// Verify directory exists in repo
	repoDir := filepath.Join(suite.tempDir, "lnk", ".config")
	suite.DirExists(repoDir)
}

func TestCLISuite(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
}
