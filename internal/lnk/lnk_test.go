package lnk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/yarlson/lnk/internal/fs"
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

func TestCoreSuite(t *testing.T) {
	suite.Run(t, new(CoreTestSuite))
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

// Test hostname detection
func (suite *CoreTestSuite) TestHostnameDetection() {
	hostname, err := GetCurrentHostname()
	suite.NoError(err)
	suite.NotEmpty(hostname)
}

// TestGetRelativePath tests path conversion with various scenarios
func (suite *CoreTestSuite) TestGetRelativePath() {
	tests := []struct {
		name       string
		path       string
		homeDir    string
		want       string
		wantErr    bool
		errMessage string
	}{
		{
			name:    "file in home root",
			path:    "/home/user/.bashrc",
			homeDir: "/home/user",
			want:    ".bashrc",
			wantErr: false,
		},
		{
			name:    "file in home subdirectory",
			path:    "/home/user/.config/app/config.json",
			homeDir: "/home/user",
			want:    ".config/app/config.json",
			wantErr: false,
		},
		{
			name:    "file outside home",
			path:    "/etc/config",
			homeDir: "/home/user",
			want:    "etc/config",
			wantErr: false,
		},
		{
			name:    "path with trailing slash",
			path:    "/home/user/.config/",
			homeDir: "/home/user",
			want:    ".config",
			wantErr: false,
		},
		{
			name:    "path equal to home",
			path:    "/home/user",
			homeDir: "/home/user",
			want:    ".",
			wantErr: false,
		},
		{
			name:       "empty path",
			path:       "",
			homeDir:    "/home/user",
			want:       "",
			wantErr:    true,
			errMessage: "failed to get relative path",
		},
		{
			name:       "relative path (should error)",
			path:       ".bashrc",
			homeDir:    "/home/user",
			want:       "",
			wantErr:    true,
			errMessage: "failed to get relative path",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Set HOME env for the test
			suite.T().Setenv("HOME", tt.homeDir)

			// Call GetRelativePath (now in filemanager package)
			got, err := fs.GetRelativePath(tt.path)

			// Verify error expectation
			if tt.wantErr {
				suite.Error(err, "Expected error for test case: %s", tt.name)
				if tt.errMessage != "" {
					suite.Contains(err.Error(), tt.errMessage, "Error message mismatch for: %s", tt.name)
				}
			} else {
				suite.NoError(err, "Unexpected error for test case: %s", tt.name)
				suite.Equal(tt.want, got, "Path mismatch for test case: %s", tt.name)
			}
		})
	}
}

// TestGetRepoPath tests repository path resolution
func (suite *CoreTestSuite) TestGetRepoPath() {
	tests := []struct {
		name       string
		setupEnv   func()
		wantSuffix string
	}{
		{
			name: "with XDG_CONFIG_HOME set",
			setupEnv: func() {
				suite.T().Setenv("XDG_CONFIG_HOME", "/custom/config")
			},
			wantSuffix: "/custom/config/lnk",
		},
		{
			name: "without XDG_CONFIG_HOME defaults to HOME/.config",
			setupEnv: func() {
				suite.T().Setenv("XDG_CONFIG_HOME", "")
				suite.T().Setenv("HOME", suite.tempDir)
			},
			wantSuffix: "/.config/lnk",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupEnv()
			path := GetRepoPath()
			suite.Contains(path, tt.wantSuffix)
		})
	}
}

// Task 1.1: Tests for HasUserContent() method
func (suite *CoreTestSuite) TestHasUserContent_WithCommonTracker_ReturnsTrue() {
	// Initialize lnk repository
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create .lnk file to simulate existing content
	lnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk")
	err = os.WriteFile(lnkFile, []byte(".bashrc\n"), 0644)
	suite.Require().NoError(err)

	// Call HasUserContent()
	hasContent := suite.lnk.HasUserContent()
	suite.True(hasContent, "Should detect common tracker file")
}

func (suite *CoreTestSuite) TestHasUserContent_WithHostTracker_ReturnsTrue() {
	// Initialize lnk repository
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create host-specific lnk instance
	hostLnk := NewLnk(WithHost("testhost"))

	// Create .lnk.hostname file to simulate host-specific content
	lnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk.testhost")
	err = os.WriteFile(lnkFile, []byte(".vimrc\n"), 0644)
	suite.Require().NoError(err)

	// Call HasUserContent()
	hasContent := hostLnk.HasUserContent()
	suite.True(hasContent, "Should detect host-specific tracker file")
}

func (suite *CoreTestSuite) TestHasUserContent_WithBothTrackers_ReturnsTrue() {
	// Initialize lnk repository
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create both common and host-specific tracker files
	commonLnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk")
	err = os.WriteFile(commonLnkFile, []byte(".bashrc\n"), 0644)
	suite.Require().NoError(err)

	hostLnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk.testhost")
	err = os.WriteFile(hostLnkFile, []byte(".vimrc\n"), 0644)
	suite.Require().NoError(err)

	// Test with common instance
	hasContent := suite.lnk.HasUserContent()
	suite.True(hasContent, "Should detect common tracker file")

	// Test with host-specific instance
	hostLnk := NewLnk(WithHost("testhost"))
	hasContent = hostLnk.HasUserContent()
	suite.True(hasContent, "Should detect host-specific tracker file")
}

func (suite *CoreTestSuite) TestHasUserContent_EmptyDirectory_ReturnsFalse() {
	// Initialize lnk repository
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Call HasUserContent() on empty repository
	hasContent := suite.lnk.HasUserContent()
	suite.False(hasContent, "Should return false for empty repository")
}

func (suite *CoreTestSuite) TestHasUserContent_NonTrackerFiles_ReturnsFalse() {
	// Initialize lnk repository
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create non-tracker files
	randomFile := filepath.Join(suite.tempDir, "lnk", "random.txt")
	err = os.WriteFile(randomFile, []byte("some content"), 0644)
	suite.Require().NoError(err)

	configFile := filepath.Join(suite.tempDir, "lnk", ".gitignore")
	err = os.WriteFile(configFile, []byte("*.log"), 0644)
	suite.Require().NoError(err)

	// Call HasUserContent()
	hasContent := suite.lnk.HasUserContent()
	suite.False(hasContent, "Should return false when only non-tracker files exist")
}

// .lnk file tracking functionality
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
