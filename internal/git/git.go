// Package git provides Git operations for lnk.
package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/yarlson/lnk/internal/lnkerr"
)

// Sentinel errors for git operations.
var (
	ErrGitInit        = errors.New("Failed to initialize git repository. Please ensure git is installed and try again.")
	ErrBranchSetup    = errors.New("Failed to set up the default branch. Please check your git installation.")
	ErrRemoteExists   = errors.New("Remote is already configured with a different repository")
	ErrGitCommand     = errors.New("Git operation failed. Please check your repository state and try again.")
	ErrNoRemote       = errors.New("No remote repository is configured. Please add a remote repository first.")
	ErrRemoteNotFound = errors.New("Remote repository is not configured")
	ErrGitConfig      = errors.New("Failed to configure git settings. Please check your git installation.")
	ErrPush           = errors.New("Failed to push changes to remote repository. Please check your network connection and repository permissions.")
	ErrPull           = errors.New("Failed to pull changes from remote repository. Please check your network connection and resolve any conflicts.")
	ErrGitTimeout     = errors.New("git operation timed out")
	ErrDirRemove      = errors.New("Failed to prepare directory for operation. Please check directory permissions.")
	ErrDirCreate      = errors.New("Failed to create directory. Please check permissions and available disk space.")
	ErrUncommitted    = errors.New("Failed to check repository status. Please verify your git repository is valid.")
)

const (
	// shortTimeout for fast local operations (status, add, commit, etc.)
	shortTimeout = 30 * time.Second

	// longTimeout for network operations and large transfers (clone, push, pull)
	longTimeout = 5 * time.Minute
)

// Git handles Git operations
type Git struct {
	repoPath string
}

// New creates a new Git instance
func New(repoPath string) *Git {
	return &Git{
		repoPath: repoPath,
	}
}

// execGitCommand creates a git command with timeout context
func (g *Git) execGitCommand(timeout time.Duration, args ...string) *exec.Cmd {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// Note: cancel is not deferred here because the command takes ownership
	// of the context. The context will be automatically cleaned up when the
	// command completes or the timeout expires.
	_ = cancel // Prevent unused variable error

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.repoPath
	return cmd
}

// Init initializes a new Git repository
func (g *Git) Init() error {
	// Try using git init -b main first (Git 2.28+)
	cmd := g.execGitCommand(shortTimeout, "init", "-b", "main")

	_, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to regular init + branch rename for older Git versions
		cmd = g.execGitCommand(shortTimeout, "init")

		_, err := cmd.CombinedOutput()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return lnkerr.WithSuggestion(ErrGitTimeout, "check system resources and try again")
			}
			return lnkerr.WithSuggestion(ErrGitInit, "ensure git is installed and try again")
		}

		// Set the default branch to main
		cmd = g.execGitCommand(shortTimeout, "symbolic-ref", "HEAD", "refs/heads/main")

		if err := cmd.Run(); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return lnkerr.WithSuggestion(ErrGitTimeout, "check system resources and try again")
			}
			return lnkerr.WithSuggestion(ErrBranchSetup, "check your git installation")
		}
	}

	return nil
}

// AddRemote adds a remote to the repository
func (g *Git) AddRemote(name, url string) error {
	// Check if remote already exists
	existingURL, err := g.getRemoteURL(name)
	if err == nil {
		// Remote exists, check if URL matches
		if existingURL == url {
			// Same URL, idempotent - do nothing
			return nil
		}
		// Different URL, error
		return lnkerr.WithPathAndSuggestion(ErrRemoteExists, name, "existing: "+existingURL+", new: "+url)
	}

	// Remote doesn't exist, add it
	cmd := g.execGitCommand(shortTimeout, "remote", "add", name, url)

	_, err = cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return lnkerr.WithSuggestion(ErrGitTimeout, "check system resources and try again")
		}
		return lnkerr.WithSuggestion(ErrGitCommand, "check the repository URL and try again")
	}

	return nil
}

// getRemoteURL returns the URL for a remote, or error if not found
func (g *Git) getRemoteURL(name string) (string, error) {
	cmd := g.execGitCommand(shortTimeout, "remote", "get-url", name)

	output, err := cmd.Output()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return "", lnkerr.Wrap(ErrGitTimeout)
		}
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// IsGitRepository checks if the directory contains a Git repository
func (g *Git) IsGitRepository() bool {
	gitDir := filepath.Join(g.repoPath, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}

// IsLnkRepository checks if the repository appears to be managed by lnk
func (g *Git) IsLnkRepository() bool {
	if !g.IsGitRepository() {
		return false
	}

	// Check if this looks like a lnk repository
	// We consider it a lnk repo if:
	// 1. It has no commits (fresh repo), OR
	// 2. All commits start with "lnk:" pattern

	commits, err := g.GetCommits()
	if err != nil {
		return false
	}

	// If no commits, it's a fresh repo - could be lnk
	if len(commits) == 0 {
		return true
	}

	// If all commits start with "lnk:", it's definitely ours
	// If ANY commit doesn't start with "lnk:", it's probably not ours
	for _, commit := range commits {
		if !strings.HasPrefix(commit, "lnk:") {
			return false
		}
	}

	return true
}

// AddAndCommit stages a file and commits it
func (g *Git) AddAndCommit(filename, message string) error {
	// Stage the file
	if err := g.Add(filename); err != nil {
		return err
	}

	// Commit the changes
	if err := g.Commit(message); err != nil {
		return err
	}

	return nil
}

// RemoveAndCommit removes a file from Git and commits the change
func (g *Git) RemoveAndCommit(filename, message string) error {
	// Remove the file from Git
	if err := g.Remove(filename); err != nil {
		return err
	}

	// Commit the changes
	if err := g.Commit(message); err != nil {
		return err
	}

	return nil
}

// Add stages a file
func (g *Git) Add(filename string) error {
	cmd := g.execGitCommand(shortTimeout, "add", filename)

	_, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return lnkerr.Wrap(ErrGitTimeout)
		}
		return lnkerr.WithSuggestion(ErrGitCommand, "check file permissions and try again")
	}

	return nil
}

// Remove removes a file from Git tracking
func (g *Git) Remove(filename string) error {
	// Check if it's a directory that needs -r flag
	fullPath := filepath.Join(g.repoPath, filename)
	info, err := os.Stat(fullPath)

	var cmd *exec.Cmd
	if err == nil && info.IsDir() {
		// Use -r and --cached flags for directories (only remove from git, not filesystem)
		cmd = g.execGitCommand(shortTimeout, "rm", "-r", "--cached", filename)
	} else {
		// Regular file (only remove from git, not filesystem)
		cmd = g.execGitCommand(shortTimeout, "rm", "--cached", filename)
	}

	_, err = cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return lnkerr.Wrap(ErrGitTimeout)
		}
		return lnkerr.WithSuggestion(ErrGitCommand, "check if the file exists and try again")
	}

	return nil
}

// Commit creates a commit with the given message
func (g *Git) Commit(message string) error {
	// Configure git user if not already configured
	if err := g.ensureGitConfig(); err != nil {
		return err
	}

	cmd := g.execGitCommand(shortTimeout, "commit", "-m", message)

	_, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return lnkerr.Wrap(ErrGitTimeout)
		}
		return lnkerr.WithSuggestion(ErrGitCommand, "ensure you have staged changes and try again")
	}

	return nil
}

// ensureGitConfig ensures that git user.name and user.email are configured
func (g *Git) ensureGitConfig() error {
	// Check if user.name is configured
	cmd := g.execGitCommand(shortTimeout, "config", "user.name")
	if output, err := cmd.Output(); err != nil || len(strings.TrimSpace(string(output))) == 0 {
		if err != nil && errors.Is(err, context.DeadlineExceeded) {
			return lnkerr.Wrap(ErrGitTimeout)
		}
		// Set a default user.name
		cmd = g.execGitCommand(shortTimeout, "config", "user.name", "Lnk User")
		if err := cmd.Run(); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return lnkerr.Wrap(ErrGitTimeout)
			}
			return lnkerr.WithSuggestion(ErrGitConfig, "check your git installation")
		}
	}

	// Check if user.email is configured
	cmd = g.execGitCommand(shortTimeout, "config", "user.email")
	if output, err := cmd.Output(); err != nil || len(strings.TrimSpace(string(output))) == 0 {
		if err != nil && errors.Is(err, context.DeadlineExceeded) {
			return lnkerr.Wrap(ErrGitTimeout)
		}
		// Set a default user.email
		cmd = g.execGitCommand(shortTimeout, "config", "user.email", "lnk@localhost")
		if err := cmd.Run(); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return lnkerr.Wrap(ErrGitTimeout)
			}
			return lnkerr.WithSuggestion(ErrGitConfig, "check your git installation")
		}
	}

	return nil
}

// GetCommits returns the list of commit messages for testing purposes
func (g *Git) GetCommits() ([]string, error) {
	// Check if .git directory exists
	gitDir := filepath.Join(g.repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	cmd := g.execGitCommand(shortTimeout, "log", "--oneline", "--format=%s")

	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, lnkerr.Wrap(ErrGitTimeout)
		}
		// If there are no commits yet, return empty slice
		outputStr := string(output)
		if strings.Contains(outputStr, "does not have any commits yet") {
			return []string{}, nil
		}
		return nil, lnkerr.Wrap(ErrGitCommand)
	}

	commits := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(commits) == 1 && commits[0] == "" {
		return []string{}, nil
	}

	return commits, nil
}

// GetRemoteInfo returns information about the default remote
func (g *Git) GetRemoteInfo() (string, error) {
	// First try to get origin remote
	url, err := g.getRemoteURL("origin")
	if err != nil {
		// If origin doesn't exist, try to get any remote
		cmd := g.execGitCommand(shortTimeout, "remote")

		output, err := cmd.Output()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return "", lnkerr.Wrap(ErrGitTimeout)
			}
			return "", lnkerr.Wrap(ErrGitCommand)
		}

		remotes := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(remotes) == 0 || remotes[0] == "" {
			return "", lnkerr.WithSuggestion(ErrNoRemote, "add a remote repository first")
		}

		// Use the first remote
		url, err = g.getRemoteURL(remotes[0])
		if err != nil {
			return "", lnkerr.WithPath(ErrRemoteNotFound, remotes[0])
		}
	}

	return url, nil
}

// StatusInfo contains repository status information
type StatusInfo struct {
	Ahead  int
	Behind int
	Remote string
	Dirty  bool
}

// GetStatus returns the repository status relative to remote
func (g *Git) GetStatus() (*StatusInfo, error) {
	// Check if we have a remote
	_, err := g.GetRemoteInfo()
	if err != nil {
		return nil, err
	}

	// Check for uncommitted changes
	dirty, err := g.HasChanges()
	if err != nil {
		return nil, lnkerr.WithSuggestion(ErrUncommitted, "verify your git repository is valid")
	}

	// Get the remote tracking branch
	cmd := g.execGitCommand(shortTimeout, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")

	output, err := cmd.Output()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, lnkerr.Wrap(ErrGitTimeout)
		}
		// No upstream branch set, assume origin/main
		remoteBranch := "origin/main"
		return &StatusInfo{
			Ahead:  g.getAheadCount(remoteBranch),
			Behind: 0, // Can't be behind if no upstream
			Remote: remoteBranch,
			Dirty:  dirty,
		}, nil
	}

	remoteBranch := strings.TrimSpace(string(output))

	return &StatusInfo{
		Ahead:  g.getAheadCount(remoteBranch),
		Behind: g.getBehindCount(remoteBranch),
		Remote: remoteBranch,
		Dirty:  dirty,
	}, nil
}

// getAheadCount returns how many commits ahead of remote
func (g *Git) getAheadCount(remoteBranch string) int {
	cmd := g.execGitCommand(shortTimeout, "rev-list", "--count", fmt.Sprintf("%s..HEAD", remoteBranch))

	output, err := cmd.Output()
	if err != nil {
		// If remote branch doesn't exist, count all local commits
		cmd = g.execGitCommand(shortTimeout, "rev-list", "--count", "HEAD")

		output, err = cmd.Output()
		if err != nil {
			return 0
		}
	}

	count := strings.TrimSpace(string(output))
	if count == "" {
		return 0
	}

	// Convert to int
	var ahead int
	if _, err := fmt.Sscanf(count, "%d", &ahead); err != nil {
		return 0
	}

	return ahead
}

// getBehindCount returns how many commits behind remote
func (g *Git) getBehindCount(remoteBranch string) int {
	cmd := g.execGitCommand(shortTimeout, "rev-list", "--count", fmt.Sprintf("HEAD..%s", remoteBranch))

	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	count := strings.TrimSpace(string(output))
	if count == "" {
		return 0
	}

	// Convert to int
	var behind int
	if _, err := fmt.Sscanf(count, "%d", &behind); err != nil {
		return 0
	}

	return behind
}

// HasChanges checks if there are uncommitted changes
func (g *Git) HasChanges() (bool, error) {
	cmd := g.execGitCommand(shortTimeout, "status", "--porcelain")

	output, err := cmd.Output()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return false, lnkerr.Wrap(ErrGitTimeout)
		}
		return false, lnkerr.Wrap(ErrGitCommand)
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}

// AddAll stages all changes in the repository
func (g *Git) AddAll() error {
	cmd := g.execGitCommand(shortTimeout, "add", "-A")

	_, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return lnkerr.Wrap(ErrGitTimeout)
		}
		return lnkerr.WithSuggestion(ErrGitCommand, "check file permissions and try again")
	}

	return nil
}

// Push pushes changes to remote
func (g *Git) Push() error {
	// First ensure we have a remote configured
	_, err := g.GetRemoteInfo()
	if err != nil {
		return lnkerr.WithSuggestion(ErrPush, err.Error())
	}

	cmd := g.execGitCommand(longTimeout, "push", "-u", "origin")

	_, err = cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return lnkerr.Wrap(ErrGitTimeout)
		}
		return lnkerr.WithSuggestion(ErrPush, "check your network connection and repository permissions")
	}

	return nil
}

// Pull pulls changes from remote
func (g *Git) Pull() error {
	// First ensure we have a remote configured
	_, err := g.GetRemoteInfo()
	if err != nil {
		return lnkerr.WithSuggestion(ErrPull, err.Error())
	}

	cmd := g.execGitCommand(longTimeout, "pull", "origin")

	_, err = cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return lnkerr.Wrap(ErrGitTimeout)
		}
		return lnkerr.WithSuggestion(ErrPull, "check your network connection and resolve any conflicts")
	}

	return nil
}

// Clone clones a repository from the given URL
func (g *Git) Clone(url string) error {
	// Remove the directory if it exists to ensure clean clone
	if err := os.RemoveAll(g.repoPath); err != nil {
		return lnkerr.WithPath(ErrDirRemove, g.repoPath)
	}

	// Create parent directory
	parentDir := filepath.Dir(g.repoPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return lnkerr.WithPath(ErrDirCreate, parentDir)
	}

	// Clone the repository
	// Note: Can't use execGitCommand here because it sets cmd.Dir to g.repoPath,
	// which doesn't exist yet. Clone needs to run from parent directory.
	ctx, cancel := context.WithTimeout(context.Background(), longTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "clone", url, g.repoPath)
	_, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return lnkerr.Wrap(ErrGitTimeout)
		}
		return lnkerr.WithSuggestion(ErrGitCommand, "check the repository URL and your network connection")
	}

	// Set up upstream tracking for main branch
	cmd = g.execGitCommand(shortTimeout, "branch", "--set-upstream-to=origin/main", "main")
	_, err = cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return lnkerr.Wrap(ErrGitTimeout)
		}
		// If main doesn't exist, try master
		cmd = g.execGitCommand(shortTimeout, "branch", "--set-upstream-to=origin/master", "master")
		_, err = cmd.CombinedOutput()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return lnkerr.Wrap(ErrGitTimeout)
			}
			// If that also fails, try to set upstream for current branch
			cmd = g.execGitCommand(shortTimeout, "branch", "--set-upstream-to=origin/HEAD")
			_, _ = cmd.CombinedOutput() // Ignore error as this is best effort
		}
	}

	return nil
}
