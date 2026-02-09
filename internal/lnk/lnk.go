// Package lnk Package core implements the business logic for lnk.
package lnk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yarlson/lnk/internal/fs"
	"github.com/yarlson/lnk/internal/git"
)

// Sentinel errors for lnk operations.
var (
	ErrManagedFilesExist = errors.New("Directory already contains managed files")
	ErrGitRepoExists     = errors.New("Directory contains an existing Git repository")
	ErrAlreadyManaged    = errors.New("File is already managed by lnk")
	ErrNotManaged        = errors.New("File is not managed by lnk")
	ErrNotInitialized    = errors.New("Lnk repository not initialized")
	ErrBootstrapNotFound = errors.New("Bootstrap script not found")
	ErrBootstrapFailed   = errors.New("Bootstrap script failed with error")
	ErrBootstrapPerms    = errors.New("Failed to make bootstrap script executable")
)

// Lnk represents the main application logic
type Lnk struct {
	repoPath string
	host     string // Host-specific configuration
	git      *git.Git
	fs       *fs.FileSystem
}

// Option configures a Lnk instance.
type Option func(*Lnk)

// WithHost sets the host for host-specific configuration
func WithHost(host string) Option {
	return func(l *Lnk) {
		l.host = host
	}
}

// NewLnk creates a new Lnk instance with optional configuration
func NewLnk(opts ...Option) *Lnk {
	repoPath := GetRepoPath()
	lnk := &Lnk{
		repoPath: repoPath,
		host:     "",
		git:      git.New(repoPath),
		fs:       fs.New(),
	}

	for _, opt := range opts {
		opt(lnk)
	}

	return lnk
}

// HasUserContent checks if the repository contains managed files
// by looking for .lnk tracker files (common or host-specific)
func (l *Lnk) HasUserContent() bool {
	// Check for common tracker file
	commonTracker := filepath.Join(l.repoPath, ".lnk")
	if _, err := os.Stat(commonTracker); err == nil {
		return true
	}

	// Check for host-specific tracker files if host is set
	if l.host != "" {
		hostTracker := filepath.Join(l.repoPath, fmt.Sprintf(".lnk.%s", l.host))
		if _, err := os.Stat(hostTracker); err == nil {
			return true
		}
	} else {
		// If no specific host is set, check for any host-specific tracker files
		// This handles cases where we want to detect any managed content
		pattern := filepath.Join(l.repoPath, ".lnk.*")
		matches, err := filepath.Glob(pattern)
		if err == nil && len(matches) > 0 {
			return true
		}
	}

	return false
}

// GetCurrentHostname returns the current system hostname
func GetCurrentHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}
	return hostname, nil
}

// GetRepoPath returns the path to the lnk repository directory
// It respects XDG_CONFIG_HOME if set, otherwise defaults to ~/.config/lnk
func GetRepoPath() string {
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// Fallback to current directory if we can't get home
			xdgConfig = "."
		} else {
			xdgConfig = filepath.Join(homeDir, ".config")
		}
	}
	return filepath.Join(xdgConfig, "lnk")
}

// getHostStoragePath returns the storage path for host-specific or common files
func (l *Lnk) getHostStoragePath() string {
	if l.host == "" {
		// Common configuration - store in root of repo
		return l.repoPath
	}
	// Host-specific configuration - store in host subdirectory
	return filepath.Join(l.repoPath, l.host+".lnk")
}

// getLnkFileName returns the appropriate .lnk tracking file name
func (l *Lnk) getLnkFileName() string {
	if l.host == "" {
		return ".lnk"
	}
	return ".lnk." + l.host
}

// getRelativePath converts an absolute path to a relative path from home directory
func getRelativePath(absPath string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check if the file is under home directory
	relPath, err := filepath.Rel(homeDir, absPath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	// If the relative path starts with "..", the file is outside home directory
	// In this case, use the absolute path as relative (without the leading slash)
	if strings.HasPrefix(relPath, "..") {
		// Use absolute path but remove leading slash and drive letter (for cross-platform)
		cleanPath := strings.TrimPrefix(absPath, "/")
		return cleanPath, nil
	}

	return relPath, nil
}
