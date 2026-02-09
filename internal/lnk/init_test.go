package lnk

import (
	"os"
	"os/exec"
	"path/filepath"
)

// Task 2.1: Tests for enhanced InitWithRemote() safety check
func (suite *CoreTestSuite) TestInitWithRemote_HasUserContent_ReturnsError() {
	// Initialize and add content first
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create .lnk file to simulate existing content
	lnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk")
	err = os.WriteFile(lnkFile, []byte(".bashrc\n"), 0644)
	suite.Require().NoError(err)

	// Try InitWithRemote - should fail
	err = suite.lnk.InitWithRemote("https://github.com/test/dotfiles.git")
	suite.Error(err, "Should fail when user content exists")
	suite.Contains(err.Error(), "already contains managed files")
	suite.Contains(err.Error(), "lnk pull")

	// Verify .lnk file still exists (no deletion occurred)
	suite.FileExists(lnkFile)
}

func (suite *CoreTestSuite) TestInitWithRemote_EmptyDirectory_Success() {
	// Create a dummy remote directory for testing
	remoteDir := filepath.Join(suite.tempDir, "remote")
	err := os.MkdirAll(remoteDir, 0755)
	suite.Require().NoError(err)

	// Initialize a bare git repository as remote
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// InitWithRemote should succeed on empty directory
	err = suite.lnk.InitWithRemote(remoteDir)
	suite.NoError(err, "Should succeed when no user content exists")

	// Verify repository was cloned
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	suite.DirExists(lnkDir)
	gitDir := filepath.Join(lnkDir, ".git")
	suite.DirExists(gitDir)
}

func (suite *CoreTestSuite) TestInitWithRemote_NoRemoteURL_BypassesSafetyCheck() {
	// Initialize and add content first
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create .lnk file to simulate existing content
	lnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk")
	err = os.WriteFile(lnkFile, []byte(".bashrc\n"), 0644)
	suite.Require().NoError(err)

	// InitWithRemote with empty URL should bypass safety check (this is local init)
	err = suite.lnk.InitWithRemote("")
	suite.NoError(err, "Should bypass safety check when no remote URL provided")
}

func (suite *CoreTestSuite) TestInitWithRemote_ErrorMessage_ContainsSuggestedCommand() {
	// Initialize and add content first
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create host-specific content
	hostLnk := NewLnk(WithHost("testhost"))
	hostLnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk.testhost")
	err = os.WriteFile(hostLnkFile, []byte(".vimrc\n"), 0644)
	suite.Require().NoError(err)

	// Try InitWithRemote - should fail with helpful message
	err = hostLnk.InitWithRemote("https://github.com/test/dotfiles.git")
	suite.Error(err, "Should fail when user content exists")
	suite.Contains(err.Error(), "lnk pull", "Should suggest pull command")
	suite.Contains(err.Error(), "instead of", "Should explain alternative")
}

// TestInitWithRemoteForce tests force initialization
// Note: Tests involving actual remote cloning are skipped as they require network access
func (suite *CoreTestSuite) TestInitWithRemoteForce() {
	tests := []struct {
		name      string
		setupFunc func() error
		remoteURL string
		force     bool
		wantErr   bool
	}{
		{
			name: "init without remote URL",
			setupFunc: func() error {
				return nil
			},
			remoteURL: "",
			force:     false,
			wantErr:   false,
		},
		{
			name: "init with force flag but no remote",
			setupFunc: func() error {
				return suite.lnk.Init()
			},
			remoteURL: "",
			force:     true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			if tt.setupFunc != nil {
				err := tt.setupFunc()
				suite.Require().NoError(err, "Setup failed")
			}

			err := suite.lnk.InitWithRemoteForce(tt.remoteURL, tt.force)

			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
		})
	}
}
