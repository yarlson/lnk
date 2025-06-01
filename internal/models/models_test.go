package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ModelsTestSuite struct {
	suite.Suite
}

func (suite *ModelsTestSuite) TestManagedFile() {
	now := time.Now()

	file := ManagedFile{
		ID:           "test-id",
		OriginalPath: "/home/user/.vimrc",
		RepoPath:     "/home/user/.config/lnk/.vimrc",
		RelativePath: ".vimrc",
		Host:         "workstation",
		IsDirectory:  false,
		AddedAt:      now,
		UpdatedAt:    now,
	}

	suite.Equal("test-id", file.ID)
	suite.Equal("/home/user/.vimrc", file.OriginalPath)
	suite.Equal("workstation", file.Host)
}

func (suite *ModelsTestSuite) TestRepositoryConfig() {
	now := time.Now()

	config := RepositoryConfig{
		Path:          "/home/user/.config/lnk",
		DefaultRemote: "origin",
		Created:       now,
		LastSync:      now,
	}

	suite.Equal("/home/user/.config/lnk", config.Path)
	suite.Equal("origin", config.DefaultRemote)
}

func (suite *ModelsTestSuite) TestHostConfig() {
	now := time.Now()

	managedFile := ManagedFile{
		RelativePath: ".vimrc",
		Host:         "workstation",
	}

	config := HostConfig{
		Name:         "workstation",
		ManagedFiles: []ManagedFile{managedFile},
		LastUpdate:   now,
	}

	suite.Equal("workstation", config.Name)
	suite.Len(config.ManagedFiles, 1)
	suite.Equal(".vimrc", config.ManagedFiles[0].RelativePath)
}

func (suite *ModelsTestSuite) TestSyncStatusIsClean() {
	tests := []struct {
		name     string
		dirty    bool
		expected bool
	}{
		{
			name:     "clean_repository",
			dirty:    false,
			expected: true,
		},
		{
			name:     "dirty_repository",
			dirty:    true,
			expected: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			status := SyncStatus{Dirty: tt.dirty}
			result := status.IsClean()
			suite.Equal(tt.expected, result)
		})
	}
}

func (suite *ModelsTestSuite) TestSyncStatusIsSynced() {
	tests := []struct {
		name     string
		ahead    int
		behind   int
		expected bool
	}{
		{
			name:     "fully_synced",
			ahead:    0,
			behind:   0,
			expected: true,
		},
		{
			name:     "ahead_of_remote",
			ahead:    2,
			behind:   0,
			expected: false,
		},
		{
			name:     "behind_remote",
			ahead:    0,
			behind:   3,
			expected: false,
		},
		{
			name:     "diverged",
			ahead:    1,
			behind:   2,
			expected: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			status := SyncStatus{
				Ahead:  tt.ahead,
				Behind: tt.behind,
			}
			result := status.IsSynced()
			suite.Equal(tt.expected, result)
		})
	}
}

func (suite *ModelsTestSuite) TestSyncStatusNeedsSync() {
	tests := []struct {
		name     string
		ahead    int
		behind   int
		expected bool
	}{
		{
			name:     "fully_synced",
			ahead:    0,
			behind:   0,
			expected: false,
		},
		{
			name:     "ahead_of_remote",
			ahead:    2,
			behind:   0,
			expected: true,
		},
		{
			name:     "behind_remote",
			ahead:    0,
			behind:   3,
			expected: true,
		},
		{
			name:     "diverged",
			ahead:    1,
			behind:   2,
			expected: true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			status := SyncStatus{
				Ahead:  tt.ahead,
				Behind: tt.behind,
			}
			result := status.NeedsSync()
			suite.Equal(tt.expected, result)
		})
	}
}

func TestModelsSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}
