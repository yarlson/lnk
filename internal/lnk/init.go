package lnk

import (
	"fmt"
	"os"

	"github.com/yarlson/lnk/internal/lnkerror"
)

// Init initializes the lnk repository
func (l *Lnk) Init() error {
	return l.InitWithRemote("")
}

// InitWithRemote initializes the lnk repository, optionally cloning from a remote
func (l *Lnk) InitWithRemote(remoteURL string) error {
	return l.InitWithRemoteForce(remoteURL, false)
}

// InitWithRemoteForce initializes the lnk repository with optional force override
func (l *Lnk) InitWithRemoteForce(remoteURL string, force bool) error {
	if remoteURL != "" {
		// Safety check: prevent data loss by checking for existing managed files
		if l.HasUserContent() {
			if !force {
				return lnkerror.WithPathAndSuggestion(ErrManagedFilesExist, l.repoPath, "use 'lnk pull' to update from remote instead of 'lnk init -r'")
			}
		}
		// Clone from remote
		return l.Clone(remoteURL)
	}

	// Create the repository directory
	if err := os.MkdirAll(l.repoPath, 0755); err != nil {
		return fmt.Errorf("failed to create lnk directory: %w", err)
	}

	// Check if there's already a Git repository
	if l.git.IsGitRepository() {
		// Repository exists, check if it's a lnk repository
		if l.git.IsLnkRepository() {
			// It's a lnk repository, init is idempotent - do nothing
			return nil
		} else {
			// It's not a lnk repository, error to prevent data loss
			return lnkerror.WithPathAndSuggestion(ErrGitRepoExists, l.repoPath, "backup or move the existing repository before initializing lnk")
		}
	}

	// No existing repository, initialize Git repository
	return l.git.Init()
}

// Clone clones a repository from the given URL
func (l *Lnk) Clone(url string) error {
	return l.git.Clone(url)
}

// AddRemote adds a remote to the repository
func (l *Lnk) AddRemote(name, url string) error {
	return l.git.AddRemote(name, url)
}
