// Package initializer handles repository initialization and remote setup.
package initializer

import (
	"fmt"
	"os"

	"github.com/yarlson/lnk/internal/git"
	"github.com/yarlson/lnk/internal/lnkerror"
	"github.com/yarlson/lnk/internal/tracker"
)

// Service handles repository initialization and remote setup.
type Service struct {
	repoPath string
	git      *git.Git
	tracker  *tracker.Tracker
}

// New creates a new initializer Service.
func New(repoPath string, g *git.Git, t *tracker.Tracker) *Service {
	return &Service{
		repoPath: repoPath,
		git:      g,
		tracker:  t,
	}
}

// Init initializes the lnk repository.
func (i *Service) Init() error {
	return i.InitWithRemote("")
}

// InitWithRemote initializes the lnk repository, optionally cloning from a remote.
func (i *Service) InitWithRemote(remoteURL string) error {
	return i.InitWithRemoteForce(remoteURL, false)
}

// InitWithRemoteForce initializes the lnk repository with optional force override.
func (i *Service) InitWithRemoteForce(remoteURL string, force bool) error {
	if remoteURL != "" {
		if i.HasUserContent() {
			if !force {
				return lnkerror.WithPathAndSuggestion(lnkerror.ErrManagedFilesExist, i.repoPath, "use 'lnk pull' to update from remote instead of 'lnk init -r'")
			}
		}
		return i.Clone(remoteURL)
	}

	if err := os.MkdirAll(i.repoPath, 0755); err != nil {
		return fmt.Errorf("failed to create lnk directory: %w", err)
	}

	if i.git.IsGitRepository() {
		if i.git.IsLnkRepository() {
			return nil
		}
		return lnkerror.WithPathAndSuggestion(lnkerror.ErrGitRepoExists, i.repoPath, "backup or move the existing repository before initializing lnk")
	}

	return i.git.Init()
}

// Clone clones a repository from the given URL.
func (i *Service) Clone(url string) error {
	return i.git.Clone(url)
}

// AddRemote adds a remote to the repository.
func (i *Service) AddRemote(name, url string) error {
	return i.git.AddRemote(name, url)
}

// HasUserContent checks if the repository contains any user-managed content.
func (i *Service) HasUserContent() bool {
	entries, err := os.ReadDir(i.repoPath)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == ".lnk" || (len(name) > 5 && name[:5] == ".lnk.") {
			return true
		}
	}

	return false
}
