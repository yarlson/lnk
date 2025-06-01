package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type GitManagerTestSuite struct {
	suite.Suite
	tempDir    string
	gitManager *GitManager
	ctx        context.Context
}

func (suite *GitManagerTestSuite) SetupTest() {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "lnk_git_test_*")
	suite.Require().NoError(err)
	suite.tempDir = tempDir

	// Create git manager
	suite.gitManager = New()

	// Create context
	suite.ctx = context.Background()
}

func (suite *GitManagerTestSuite) TearDownTest() {
	err := os.RemoveAll(suite.tempDir)
	suite.Require().NoError(err)
}

// Helper function to check if file exists
func (suite *GitManagerTestSuite) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (suite *GitManagerTestSuite) TestInit() {
	repoPath := filepath.Join(suite.tempDir, "test-repo")

	// Create the directory
	err := os.MkdirAll(repoPath, 0755)
	suite.Require().NoError(err)

	// Test init
	err = suite.gitManager.Init(suite.ctx, repoPath)
	suite.NoError(err)

	// Verify repository was created
	isRepo, err := suite.gitManager.IsRepository(suite.ctx, repoPath)
	suite.NoError(err)
	suite.True(isRepo)
}

func (suite *GitManagerTestSuite) TestAddCommit() {
	repoPath := filepath.Join(suite.tempDir, "test-repo")

	// Create and initialize repository
	err := os.MkdirAll(repoPath, 0755)
	suite.Require().NoError(err)

	err = suite.gitManager.Init(suite.ctx, repoPath)
	suite.Require().NoError(err)

	// Create a test file
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	suite.Require().NoError(err)

	// Test adding file
	err = suite.gitManager.Add(suite.ctx, repoPath, "test.txt")
	suite.NoError(err)

	// Test commit
	err = suite.gitManager.Commit(suite.ctx, repoPath, "lnk: test commit")
	suite.NoError(err)

	// Verify no uncommitted changes
	hasChanges, err := suite.gitManager.HasChanges(suite.ctx, repoPath)
	suite.NoError(err)
	suite.False(hasChanges)
}

func (suite *GitManagerTestSuite) TestStatus() {
	repoPath := filepath.Join(suite.tempDir, "test-repo")

	// Create and initialize repository
	err := os.MkdirAll(repoPath, 0755)
	suite.Require().NoError(err)

	err = suite.gitManager.Init(suite.ctx, repoPath)
	suite.Require().NoError(err)

	// Test status on empty repository should fail with no remote configured
	_, err = suite.gitManager.Status(suite.ctx, repoPath)
	suite.Error(err)
	suite.Contains(err.Error(), "no remote configured")

	// Add a remote to make status work
	testURL := "https://github.com/test/repo.git"
	err = suite.gitManager.AddRemote(suite.ctx, repoPath, "origin", testURL)
	suite.Require().NoError(err)

	// Test status with remote configured but no commits
	status, err := suite.gitManager.Status(suite.ctx, repoPath)
	suite.NoError(err)

	suite.Equal("main", status.CurrentBranch)
	suite.False(status.Dirty)
	suite.True(status.HasRemote)

	// Create and commit a file
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	suite.Require().NoError(err)

	// Test dirty status
	status, err = suite.gitManager.Status(suite.ctx, repoPath)
	suite.NoError(err)
	suite.True(status.Dirty)

	// Add and commit
	err = suite.gitManager.Add(suite.ctx, repoPath, "test.txt")
	suite.Require().NoError(err)

	err = suite.gitManager.Commit(suite.ctx, repoPath, "lnk: test commit")
	suite.Require().NoError(err)

	// Test clean status
	status, err = suite.gitManager.Status(suite.ctx, repoPath)
	suite.NoError(err)
	suite.False(status.Dirty)
	suite.NotEmpty(status.LastCommitHash)
}

func (suite *GitManagerTestSuite) TestRemoteOperations() {
	repoPath := filepath.Join(suite.tempDir, "test-repo")

	// Create and initialize repository
	err := os.MkdirAll(repoPath, 0755)
	suite.Require().NoError(err)

	err = suite.gitManager.Init(suite.ctx, repoPath)
	suite.Require().NoError(err)

	// Test adding remote
	testURL := "https://github.com/test/repo.git"
	err = suite.gitManager.AddRemote(suite.ctx, repoPath, "origin", testURL)
	suite.NoError(err)

	// Test getting remote URL
	remoteURL, err := suite.gitManager.GetRemoteURL(suite.ctx, repoPath, "origin")
	suite.NoError(err)
	suite.Equal(testURL, remoteURL)

	// Test idempotent add (same URL)
	err = suite.gitManager.AddRemote(suite.ctx, repoPath, "origin", testURL)
	suite.NoError(err)

	// Test adding remote with different URL should fail
	err = suite.gitManager.AddRemote(suite.ctx, repoPath, "origin", "https://github.com/different/repo.git")
	suite.Error(err)
}

func (suite *GitManagerTestSuite) TestIsLnkRepository() {
	tests := []struct {
		name     string
		setup    func(string) error
		expected bool
	}{
		{
			name: "not_a_repository",
			setup: func(path string) error {
				return os.MkdirAll(path, 0755)
			},
			expected: false,
		},
		{
			name: "empty_git_repository",
			setup: func(path string) error {
				if err := os.MkdirAll(path, 0755); err != nil {
					return err
				}
				return suite.gitManager.Init(suite.ctx, path)
			},
			expected: true,
		},
		{
			name: "repository_with_lnk_commits",
			setup: func(path string) error {
				if err := os.MkdirAll(path, 0755); err != nil {
					return err
				}
				if err := suite.gitManager.Init(suite.ctx, path); err != nil {
					return err
				}

				// Create and commit a file with lnk prefix
				testFile := filepath.Join(path, "test.txt")
				if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
					return err
				}
				if err := suite.gitManager.Add(suite.ctx, path, "test.txt"); err != nil {
					return err
				}
				return suite.gitManager.Commit(suite.ctx, path, "lnk: add test file")
			},
			expected: true,
		},
		{
			name: "repository_with_non-lnk_commits",
			setup: func(path string) error {
				if err := os.MkdirAll(path, 0755); err != nil {
					return err
				}
				if err := suite.gitManager.Init(suite.ctx, path); err != nil {
					return err
				}

				// Create and commit a file without lnk prefix
				testFile := filepath.Join(path, "test.txt")
				if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
					return err
				}
				if err := suite.gitManager.Add(suite.ctx, path, "test.txt"); err != nil {
					return err
				}
				return suite.gitManager.Commit(suite.ctx, path, "regular commit")
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			repoPath := filepath.Join(suite.tempDir, tt.name)
			err := tt.setup(repoPath)
			suite.Require().NoError(err)

			isLnk, err := suite.gitManager.IsLnkRepository(suite.ctx, repoPath)
			suite.NoError(err)
			suite.Equal(tt.expected, isLnk)
		})
	}
}

func (suite *GitManagerTestSuite) TestContextCancellation() {
	repoPath := filepath.Join(suite.tempDir, "test-repo")

	err := os.MkdirAll(repoPath, 0755)
	suite.Require().NoError(err)

	// Test context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// This should fail due to context timeout
	err = suite.gitManager.Init(ctx, repoPath)
	suite.Error(err)

	// Verify the error is context-related
	suite.NotNil(ctx.Err())
}

func (suite *GitManagerTestSuite) TestRemove() {
	repoPath := filepath.Join(suite.tempDir, "test-repo")

	// Create and initialize repository
	err := os.MkdirAll(repoPath, 0755)
	suite.Require().NoError(err)

	err = suite.gitManager.Init(suite.ctx, repoPath)
	suite.Require().NoError(err)

	// Create and add files
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	suite.Require().NoError(err)

	err = suite.gitManager.Add(suite.ctx, repoPath, "test.txt")
	suite.Require().NoError(err)

	err = suite.gitManager.Commit(suite.ctx, repoPath, "lnk: add test file")
	suite.Require().NoError(err)

	// Test removing file
	err = suite.gitManager.Remove(suite.ctx, repoPath, "test.txt")
	suite.NoError(err)

	// Verify file is removed from git but still exists on fs
	suite.True(suite.fileExists(testFile))

	// Verify repository has changes
	hasChanges, err := suite.gitManager.HasChanges(suite.ctx, repoPath)
	suite.NoError(err)
	suite.True(hasChanges)
}

func TestGitManagerSuite(t *testing.T) {
	suite.Run(t, new(GitManagerTestSuite))
}
