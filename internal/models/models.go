package models

import (
	"os"
	"time"
)

// ManagedFile represents a file or directory managed by lnk
type ManagedFile struct {
	// ID for potential future database use
	ID string `json:"id,omitempty"`

	// OriginalPath is the original absolute path where the file was located
	OriginalPath string `json:"original_path"`

	// RepoPath is the path within the lnk repository
	RepoPath string `json:"repo_path"`

	// RelativePath is the path relative to the home directory (or absolute for files outside home)
	RelativePath string `json:"relative_path"`

	// Host is the hostname where this file is managed
	Host string `json:"host"`

	// IsDirectory indicates whether this is a directory
	IsDirectory bool `json:"is_directory"`

	// SymlinkTarget is the current symlink target (if the original location is now a symlink)
	SymlinkTarget string `json:"symlink_target,omitempty"`

	// AddedAt is when the file was first added to lnk
	AddedAt time.Time `json:"added_at,omitempty"`

	// UpdatedAt is when the file was last updated
	UpdatedAt time.Time `json:"updated_at,omitempty"`

	// Mode stores the file permissions
	Mode os.FileMode `json:"mode,omitempty"`
}

// RepositoryConfig represents the lnk repository settings
type RepositoryConfig struct {
	// Path is the absolute path to the lnk repository
	Path string `json:"path"`

	// DefaultRemote is the default Git remote for sync operations
	DefaultRemote string `json:"default_remote,omitempty"`

	// Created is when the repository was created
	Created time.Time `json:"created,omitempty"`

	// LastSync is when the repository was last synced
	LastSync time.Time `json:"last_sync,omitempty"`
}

// HostConfig represents configuration specific to a host
type HostConfig struct {
	// Name is the hostname
	Name string `json:"name"`

	// ManagedFiles is the list of files managed on this host
	ManagedFiles []ManagedFile `json:"managed_files"`

	// LastUpdate is when this host configuration was last updated
	LastUpdate time.Time `json:"last_update,omitempty"`
}

// SyncStatus represents Git repository sync status
type SyncStatus struct {
	// Ahead is the number of commits ahead of remote
	Ahead int `json:"ahead"`

	// Behind is the number of commits behind remote
	Behind int `json:"behind"`

	// CurrentBranch is the currently checked out branch
	CurrentBranch string `json:"current_branch"`

	// RemoteBranch is the remote tracking branch
	RemoteBranch string `json:"remote_branch"`

	// RemoteURL is the URL of the remote repository
	RemoteURL string `json:"remote_url"`

	// Dirty indicates if there are uncommitted changes
	Dirty bool `json:"dirty"`

	// LastCommitHash is the hash of the last commit
	LastCommitHash string `json:"last_commit_hash"`

	// HasRemote indicates if a remote is configured
	HasRemote bool `json:"has_remote"`
}

// IsClean returns true if the repository is clean (no uncommitted changes)
func (s *SyncStatus) IsClean() bool {
	return !s.Dirty
}

// IsSynced returns true if the repository is in sync with remote (ahead=0, behind=0)
func (s *SyncStatus) IsSynced() bool {
	return s.Ahead == 0 && s.Behind == 0
}

// NeedsSync returns true if the repository needs to be synced with remote
func (s *SyncStatus) NeedsSync() bool {
	return s.Ahead > 0 || s.Behind > 0
}
