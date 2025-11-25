package git

import (
	"context"
	stderrors "errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/yarlson/lnk/internal/errors"
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

		output, err := cmd.CombinedOutput()
		if err != nil {
			if stderrors.Is(err, context.DeadlineExceeded) {
				return &errors.GitTimeoutError{
					Command: "git init",
					Timeout: shortTimeout,
					Err:     err,
				}
			}
			return &errors.GitInitError{Output: string(output), Err: err}
		}

		// Set the default branch to main
		cmd = g.execGitCommand(shortTimeout, "symbolic-ref", "HEAD", "refs/heads/main")

		if err := cmd.Run(); err != nil {
			if stderrors.Is(err, context.DeadlineExceeded) {
				return &errors.GitTimeoutError{
					Command: "git symbolic-ref HEAD refs/heads/main",
					Timeout: shortTimeout,
					Err:     err,
				}
			}
			return &errors.BranchSetupError{Err: err}
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
		return &errors.RemoteExistsError{Remote: name, ExistingURL: existingURL, NewURL: url}
	}

	// Remote doesn't exist, add it
	cmd := g.execGitCommand(shortTimeout, "remote", "add", name, url)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if stderrors.Is(err, context.DeadlineExceeded) {
			return &errors.GitTimeoutError{
				Command: fmt.Sprintf("git remote add %s %s", name, url),
				Timeout: shortTimeout,
				Err:     err,
			}
		}
		return &errors.GitCommandError{Command: "remote add", Output: string(output), Err: err}
	}

	return nil
}

// getRemoteURL returns the URL for a remote, or error if not found
func (g *Git) getRemoteURL(name string) (string, error) {
	cmd := g.execGitCommand(shortTimeout, "remote", "get-url", name)

	output, err := cmd.Output()
	if err != nil {
		if stderrors.Is(err, context.DeadlineExceeded) {
			return "", &errors.GitTimeoutError{
				Command: fmt.Sprintf("git remote get-url %s", name),
				Timeout: shortTimeout,
				Err:     err,
			}
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

	output, err := cmd.CombinedOutput()
	if err != nil {
		if stderrors.Is(err, context.DeadlineExceeded) {
			return &errors.GitTimeoutError{
				Command: fmt.Sprintf("git add %s", filename),
				Timeout: shortTimeout,
				Err:     err,
			}
		}
		return &errors.GitCommandError{Command: "add", Output: string(output), Err: err}
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

	output, err := cmd.CombinedOutput()
	if err != nil {
		if stderrors.Is(err, context.DeadlineExceeded) {
			return &errors.GitTimeoutError{
				Command: fmt.Sprintf("git rm --cached %s", filename),
				Timeout: shortTimeout,
				Err:     err,
			}
		}
		return &errors.GitCommandError{Command: "rm", Output: string(output), Err: err}
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

	output, err := cmd.CombinedOutput()
	if err != nil {
		if stderrors.Is(err, context.DeadlineExceeded) {
			return &errors.GitTimeoutError{
				Command: "git commit",
				Timeout: shortTimeout,
				Err:     err,
			}
		}
		return &errors.GitCommandError{Command: "commit", Output: string(output), Err: err}
	}

	return nil
}

// ensureGitConfig ensures that git user.name and user.email are configured
func (g *Git) ensureGitConfig() error {
	// Check if user.name is configured
	cmd := g.execGitCommand(shortTimeout, "config", "user.name")
	if output, err := cmd.Output(); err != nil || len(strings.TrimSpace(string(output))) == 0 {
		if err != nil && stderrors.Is(err, context.DeadlineExceeded) {
			return &errors.GitTimeoutError{
				Command: "git config user.name",
				Timeout: shortTimeout,
				Err:     err,
			}
		}
		// Set a default user.name
		cmd = g.execGitCommand(shortTimeout, "config", "user.name", "Lnk User")
		if err := cmd.Run(); err != nil {
			if stderrors.Is(err, context.DeadlineExceeded) {
				return &errors.GitTimeoutError{
					Command: "git config user.name 'Lnk User'",
					Timeout: shortTimeout,
					Err:     err,
				}
			}
			return &errors.GitConfigError{Setting: "user.name", Err: err}
		}
	}

	// Check if user.email is configured
	cmd = g.execGitCommand(shortTimeout, "config", "user.email")
	if output, err := cmd.Output(); err != nil || len(strings.TrimSpace(string(output))) == 0 {
		if err != nil && stderrors.Is(err, context.DeadlineExceeded) {
			return &errors.GitTimeoutError{
				Command: "git config user.email",
				Timeout: shortTimeout,
				Err:     err,
			}
		}
		// Set a default user.email
		cmd = g.execGitCommand(shortTimeout, "config", "user.email", "lnk@localhost")
		if err := cmd.Run(); err != nil {
			if stderrors.Is(err, context.DeadlineExceeded) {
				return &errors.GitTimeoutError{
					Command: "git config user.email 'lnk@localhost'",
					Timeout: shortTimeout,
					Err:     err,
				}
			}
			return &errors.GitConfigError{Setting: "user.email", Err: err}
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
		if stderrors.Is(err, context.DeadlineExceeded) {
			return nil, &errors.GitTimeoutError{
				Command: "git log --oneline --format=%s",
				Timeout: shortTimeout,
				Err:     err,
			}
		}
		// If there are no commits yet, return empty slice
		outputStr := string(output)
		if strings.Contains(outputStr, "does not have any commits yet") {
			return []string{}, nil
		}
		return nil, &errors.GitCommandError{Command: "log", Output: outputStr, Err: err}
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
			if stderrors.Is(err, context.DeadlineExceeded) {
				return "", &errors.GitTimeoutError{
					Command: "git remote",
					Timeout: shortTimeout,
					Err:     err,
				}
			}
			return "", &errors.GitCommandError{Command: "remote", Output: string(output), Err: err}
		}

		remotes := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(remotes) == 0 || remotes[0] == "" {
			return "", &errors.NoRemoteError{}
		}

		// Use the first remote
		url, err = g.getRemoteURL(remotes[0])
		if err != nil {
			return "", &errors.RemoteNotFoundError{Remote: remotes[0], Err: err}
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
		return nil, &errors.UncommittedChangesError{Err: err}
	}

	// Get the remote tracking branch
	cmd := g.execGitCommand(shortTimeout, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")

	output, err := cmd.Output()
	if err != nil {
		if stderrors.Is(err, context.DeadlineExceeded) {
			return nil, &errors.GitTimeoutError{
				Command: "git rev-parse --abbrev-ref --symbolic-full-name @{u}",
				Timeout: shortTimeout,
				Err:     err,
			}
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
		if stderrors.Is(err, context.DeadlineExceeded) {
			return false, &errors.GitTimeoutError{
				Command: "git status --porcelain",
				Timeout: shortTimeout,
				Err:     err,
			}
		}
		return false, &errors.GitCommandError{Command: "status", Output: string(output), Err: err}
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}

// AddAll stages all changes in the repository
func (g *Git) AddAll() error {
	cmd := g.execGitCommand(shortTimeout, "add", "-A")

	output, err := cmd.CombinedOutput()
	if err != nil {
		if stderrors.Is(err, context.DeadlineExceeded) {
			return &errors.GitTimeoutError{
				Command: "git add -A",
				Timeout: shortTimeout,
				Err:     err,
			}
		}
		return &errors.GitCommandError{Command: "add", Output: string(output), Err: err}
	}

	return nil
}

// Push pushes changes to remote
func (g *Git) Push() error {
	// First ensure we have a remote configured
	_, err := g.GetRemoteInfo()
	if err != nil {
		return &errors.PushError{Reason: err.Error(), Err: err}
	}

	cmd := g.execGitCommand(longTimeout, "push", "-u", "origin")

	output, err := cmd.CombinedOutput()
	if err != nil {
		if stderrors.Is(err, context.DeadlineExceeded) {
			return &errors.GitTimeoutError{
				Command: "git push -u origin",
				Timeout: longTimeout,
				Err:     err,
			}
		}
		return &errors.PushError{Output: string(output), Err: err}
	}

	return nil
}

// Pull pulls changes from remote
func (g *Git) Pull() error {
	// First ensure we have a remote configured
	_, err := g.GetRemoteInfo()
	if err != nil {
		return &errors.PullError{Reason: err.Error(), Err: err}
	}

	cmd := g.execGitCommand(longTimeout, "pull", "origin")

	output, err := cmd.CombinedOutput()
	if err != nil {
		if stderrors.Is(err, context.DeadlineExceeded) {
			return &errors.GitTimeoutError{
				Command: "git pull origin",
				Timeout: longTimeout,
				Err:     err,
			}
		}
		return &errors.PullError{Output: string(output), Err: err}
	}

	return nil
}

// Clone clones a repository from the given URL
func (g *Git) Clone(url string) error {
	// Remove the directory if it exists to ensure clean clone
	if err := os.RemoveAll(g.repoPath); err != nil {
		return &errors.DirectoryRemovalError{Path: g.repoPath, Err: err}
	}

	// Create parent directory
	parentDir := filepath.Dir(g.repoPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return &errors.DirectoryCreationError{Path: parentDir, Err: err}
	}

	// Clone the repository
	// Note: Can't use execGitCommand here because it sets cmd.Dir to g.repoPath,
	// which doesn't exist yet. Clone needs to run from parent directory.
	ctx, cancel := context.WithTimeout(context.Background(), longTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "clone", url, g.repoPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if stderrors.Is(err, context.DeadlineExceeded) {
			return &errors.GitTimeoutError{
				Command: fmt.Sprintf("git clone %s", url),
				Timeout: longTimeout,
				Err:     err,
			}
		}
		return &errors.GitCommandError{Command: "clone", Output: string(output), Err: err}
	}

	// Set up upstream tracking for main branch
	cmd = g.execGitCommand(shortTimeout, "branch", "--set-upstream-to=origin/main", "main")
	_, err = cmd.CombinedOutput()
	if err != nil {
		if stderrors.Is(err, context.DeadlineExceeded) {
			return &errors.GitTimeoutError{
				Command: "git branch --set-upstream-to=origin/main main",
				Timeout: shortTimeout,
				Err:     err,
			}
		}
		// If main doesn't exist, try master
		cmd = g.execGitCommand(shortTimeout, "branch", "--set-upstream-to=origin/master", "master")
		_, err = cmd.CombinedOutput()
		if err != nil {
			if stderrors.Is(err, context.DeadlineExceeded) {
				return &errors.GitTimeoutError{
					Command: "git branch --set-upstream-to=origin/master master",
					Timeout: shortTimeout,
					Err:     err,
				}
			}
			// If that also fails, try to set upstream for current branch
			cmd = g.execGitCommand(shortTimeout, "branch", "--set-upstream-to=origin/HEAD")
			_, _ = cmd.CombinedOutput() // Ignore error as this is best effort
		}
	}

	return nil
}
