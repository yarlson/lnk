package pathresolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ResolverTestSuite struct {
	suite.Suite
	resolver *Resolver
}

func (suite *ResolverTestSuite) SetupTest() {
	suite.resolver = New()
}

func (suite *ResolverTestSuite) TestGetRepoStoragePath() {
	// Test with XDG_CONFIG_HOME set
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	suite.Run("with_XDG_CONFIG_HOME_set", func() {
		testXDG := "/test/config"
		os.Setenv("XDG_CONFIG_HOME", testXDG)

		path, err := suite.resolver.GetRepoStoragePath()
		suite.NoError(err)

		expected := filepath.Join(testXDG, "lnk")
		suite.Equal(expected, path)
	})

	suite.Run("without_XDG_CONFIG_HOME", func() {
		os.Unsetenv("XDG_CONFIG_HOME")

		path, err := suite.resolver.GetRepoStoragePath()
		suite.NoError(err)

		homeDir, _ := os.UserHomeDir()
		expected := filepath.Join(homeDir, ".config", "lnk")
		suite.Equal(expected, path)
	})
}

func (suite *ResolverTestSuite) TestGetTrackingFilePath() {
	repoPath := "/test/repo"

	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{
			name:     "common_config",
			host:     "",
			expected: filepath.Join(repoPath, ".lnk"),
		},
		{
			name:     "host-specific_config",
			host:     "myhost",
			expected: filepath.Join(repoPath, ".lnk.myhost"),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			path, err := suite.resolver.GetTrackingFilePath(repoPath, tt.host)
			suite.NoError(err)
			suite.Equal(tt.expected, path)
		})
	}
}

func (suite *ResolverTestSuite) TestGetHostStoragePath() {
	repoPath := "/test/repo"

	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{
			name:     "common_config",
			host:     "",
			expected: repoPath,
		},
		{
			name:     "host-specific_config",
			host:     "myhost",
			expected: filepath.Join(repoPath, "myhost.lnk"),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			path, err := suite.resolver.GetHostStoragePath(repoPath, tt.host)
			suite.NoError(err)
			suite.Equal(tt.expected, path)
		})
	}
}

func (suite *ResolverTestSuite) TestGetRelativePathFromHome() {
	homeDir, err := os.UserHomeDir()
	suite.Require().NoError(err)

	tests := []struct {
		name     string
		absPath  string
		expected string
	}{
		{
			name:     "file_in_home",
			absPath:  filepath.Join(homeDir, "Documents", "test.txt"),
			expected: filepath.Join("Documents", "test.txt"),
		},
		{
			name:     "file_outside_home",
			absPath:  "/etc/hosts",
			expected: "etc/hosts",
		},
		{
			name:     "home_directory_itself",
			absPath:  homeDir,
			expected: ".",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result, err := suite.resolver.GetRelativePathFromHome(tt.absPath)
			suite.NoError(err)
			suite.Equal(tt.expected, result)
		})
	}
}

func (suite *ResolverTestSuite) TestGetAbsolutePathInHome() {
	homeDir, err := os.UserHomeDir()
	suite.Require().NoError(err)

	tests := []struct {
		name     string
		relPath  string
		expected string
	}{
		{
			name:     "relative_path_in_home",
			relPath:  filepath.Join("Documents", "test.txt"),
			expected: filepath.Join(homeDir, "Documents", "test.txt"),
		},
		{
			name:     "already_absolute_path",
			relPath:  "/etc/hosts",
			expected: "/etc/hosts",
		},
		{
			name:     "absolute-like_path_without_leading_slash",
			relPath:  "etc/hosts",
			expected: "/etc/hosts",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result, err := suite.resolver.GetAbsolutePathInHome(tt.relPath)
			suite.NoError(err)
			suite.Equal(tt.expected, result)
		})
	}
}

func (suite *ResolverTestSuite) TestIsUnderHome() {
	homeDir, err := os.UserHomeDir()
	suite.Require().NoError(err)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "file_in_home",
			path:     filepath.Join(homeDir, "Documents", "test.txt"),
			expected: true,
		},
		{
			name:     "file_outside_home",
			path:     "/etc/hosts",
			expected: false,
		},
		{
			name:     "home_directory_itself",
			path:     homeDir,
			expected: true,
		},
		{
			name:     "parent_of_home",
			path:     filepath.Dir(homeDir),
			expected: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result, err := suite.resolver.IsUnderHome(tt.path)
			suite.NoError(err)
			suite.Equal(tt.expected, result)
		})
	}
}

func (suite *ResolverTestSuite) TestGetFileStoragePathInRepo() {
	repoPath := "/test/repo"

	tests := []struct {
		name         string
		host         string
		relativePath string
		expected     string
	}{
		{
			name:         "common_config_file",
			host:         "",
			relativePath: "Documents/test.txt",
			expected:     filepath.Join(repoPath, "Documents", "test.txt"),
		},
		{
			name:         "host-specific_file",
			host:         "myhost",
			relativePath: "Documents/test.txt",
			expected:     filepath.Join(repoPath, "myhost.lnk", "Documents", "test.txt"),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result, err := suite.resolver.GetFileStoragePathInRepo(repoPath, tt.host, tt.relativePath)
			suite.NoError(err)
			suite.Equal(tt.expected, result)
		})
	}
}

func TestResolverSuite(t *testing.T) {
	suite.Run(t, new(ResolverTestSuite))
}
