package lnk

import (
	"os"
	"path/filepath"
	"strings"
)

// TestSymlinkRestoration tests symlink restoration after pull
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

// TestGitOperations tests git commit and remote operations
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

// TestStatusDetectsDirtyRepo tests dirty repository detection
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

// TestListManagedItems tests list functionality
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

// TestMultihostFileOperations tests common and host-specific file operations
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

// TestMultihostSymlinkRestoration tests host-specific symlink restoration
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

// TestMultihostIsolation tests that common and host-specific configs don't interfere
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

// TestIsValidSymlink tests symlink validation logic
func (suite *CoreTestSuite) TestIsValidSymlink() {
	// Setup test directory structure
	repoDir := filepath.Join(suite.tempDir, "lnk")
	err := os.MkdirAll(repoDir, 0755)
	suite.Require().NoError(err)

	// Create a valid target file in repo
	validTarget := filepath.Join(repoDir, "target.txt")
	err = os.WriteFile(validTarget, []byte("content"), 0644)
	suite.Require().NoError(err)

	// Create a valid symlink pointing to target
	validSymlink := filepath.Join(suite.tempDir, "valid-link")
	err = os.Symlink(validTarget, validSymlink)
	suite.Require().NoError(err)

	// Create a broken symlink
	brokenSymlink := filepath.Join(suite.tempDir, "broken-link")
	err = os.Symlink(filepath.Join(repoDir, "nonexistent"), brokenSymlink)
	suite.Require().NoError(err)

	// Create symlink pointing to different target
	wrongTarget := filepath.Join(suite.tempDir, "wrong.txt")
	err = os.WriteFile(wrongTarget, []byte("content"), 0644)
	suite.Require().NoError(err)
	wrongSymlink := filepath.Join(suite.tempDir, "wrong-link")
	err = os.Symlink(wrongTarget, wrongSymlink)
	suite.Require().NoError(err)

	// Create regular file (not a symlink)
	regularFile := filepath.Join(suite.tempDir, "regular.txt")
	err = os.WriteFile(regularFile, []byte("content"), 0644)
	suite.Require().NoError(err)

	tests := []struct {
		name           string
		symlink        string
		expectedTarget string
		want           bool
		setup          func()
	}{
		{
			name:           "valid symlink pointing to correct target",
			symlink:        validSymlink,
			expectedTarget: validTarget,
			want:           true,
			setup:          nil,
		},
		{
			name:           "broken symlink pointing to expected target",
			symlink:        brokenSymlink,
			expectedTarget: filepath.Join(repoDir, "nonexistent"),
			want:           true, // Function validates path matching, not target existence
		},
		{
			name:           "symlink pointing to wrong target",
			symlink:        wrongSymlink,
			expectedTarget: validTarget,
			want:           false,
		},
		{
			name:           "not a symlink (regular file)",
			symlink:        regularFile,
			expectedTarget: validTarget,
			want:           false,
		},
		{
			name:           "nonexistent symlink path",
			symlink:        filepath.Join(suite.tempDir, "nonexistent"),
			expectedTarget: validTarget,
			want:           false,
		},
		{
			name:           "relative symlink to target",
			symlink:        filepath.Join(suite.tempDir, "rel-link"),
			expectedTarget: validTarget,
			want:           true,
			setup: func() {
				relLink := filepath.Join(suite.tempDir, "rel-link")
				relTarget, _ := filepath.Rel(suite.tempDir, validTarget)
				_ = os.Symlink(relTarget, relLink)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			if tt.setup != nil {
				tt.setup()
			}

			// Call isValidSymlink with expected target
			got := suite.lnk.isValidSymlink(tt.symlink, tt.expectedTarget)

			suite.Equal(tt.want, got, "Validation mismatch for test case: %s", tt.name)
		})
	}
}

// TestRestoreSymlinks tests symlink restoration with table-driven tests
func (suite *CoreTestSuite) TestRestoreSymlinks() {
	tests := []struct {
		name       string
		setupFunc  func() error
		verifyFunc func()
	}{
		{
			name: "restore symlinks for tracked files",
			setupFunc: func() error {
				err := suite.lnk.Init()
				if err != nil {
					return err
				}

				// Create file in repo directly (simulating a pull)
				repoFile := filepath.Join(suite.tempDir, "lnk", ".bashrc")
				err = os.WriteFile(repoFile, []byte("export PATH"), 0644)
				if err != nil {
					return err
				}

				// Create .lnk tracking file
				lnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk")
				return os.WriteFile(lnkFile, []byte(".bashrc\n"), 0644)
			},
			verifyFunc: func() {
				homeDir, _ := os.UserHomeDir()
				targetFile := filepath.Join(homeDir, ".bashrc")
				defer func() { _ = os.Remove(targetFile) }()

				// File should be a symlink
				info, err := os.Lstat(targetFile)
				suite.NoError(err)
				suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)
			},
		},
		{
			name: "skip missing files in tracking",
			setupFunc: func() error {
				err := suite.lnk.Init()
				if err != nil {
					return err
				}

				// Create .lnk tracking file with missing file
				lnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk")
				return os.WriteFile(lnkFile, []byte("nonexistent.txt\n"), 0644)
			},
			verifyFunc: func() {
				homeDir, _ := os.UserHomeDir()
				targetFile := filepath.Join(homeDir, "nonexistent.txt")
				// File should not exist
				_, err := os.Lstat(targetFile)
				suite.Error(err)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := tt.setupFunc()
			suite.Require().NoError(err, "Setup failed for test: %s", tt.name)

			// Execute RestoreSymlinks
			_, err = suite.lnk.RestoreSymlinks()
			suite.NoError(err, "RestoreSymlinks should not error")

			if tt.verifyFunc != nil {
				tt.verifyFunc()
			}
		})
	}
}

// TestPush tests push operation error paths
func (suite *CoreTestSuite) TestPush() {
	tests := []struct {
		name        string
		setupFunc   func() error
		message     string
		wantErr     bool
		errContains string
	}{
		{
			name: "push without remote configured",
			setupFunc: func() error {
				// Initialize without remote
				return suite.lnk.Init()
			},
			message:     "test commit",
			wantErr:     true,
			errContains: "No remote repository",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Setup
			if tt.setupFunc != nil {
				err := tt.setupFunc()
				suite.Require().NoError(err, "Setup failed for test: %s", tt.name)
			}

			// Execute Push
			err := suite.lnk.Push(tt.message)

			// Verify
			if tt.wantErr {
				suite.Error(err, "Expected error for test: %s", tt.name)
				if tt.errContains != "" {
					suite.Contains(err.Error(), tt.errContains, "Error message mismatch for: %s", tt.name)
				}
			} else {
				suite.NoError(err, "Unexpected error for test: %s", tt.name)
			}
		})
	}
}

// TestPull tests pull operation error paths
func (suite *CoreTestSuite) TestPull() {
	tests := []struct {
		name        string
		setupFunc   func() error
		wantErr     bool
		errContains string
	}{
		{
			name: "pull without remote configured",
			setupFunc: func() error {
				return suite.lnk.Init()
			},
			wantErr:     true,
			errContains: "No remote repository",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Setup
			if tt.setupFunc != nil {
				err := tt.setupFunc()
				suite.Require().NoError(err, "Setup failed for test: %s", tt.name)
			}

			// Execute Pull
			_, err := suite.lnk.Pull()

			// Verify
			if tt.wantErr {
				suite.Error(err, "Expected error for test: %s", tt.name)
				if tt.errContains != "" {
					suite.Contains(err.Error(), tt.errContains, "Error message mismatch for: %s", tt.name)
				}
			} else {
				suite.NoError(err, "Unexpected error for test: %s", tt.name)
			}
		})
	}
}
