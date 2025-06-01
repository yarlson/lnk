package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yarlson/lnk/internal/errors"
	"github.com/yarlson/lnk/internal/models"
)

// GitManager implements the models.GitManager interface
type GitManager struct{}

// New creates a new GitManager instance
func New() *GitManager {
	return &GitManager{}
}

// Init initializes a new Git repository at repoPath
func (g *GitManager) Init(ctx context.Context, repoPath string) error {
	// Try using git init -b main first (Git 2.28+)
	cmd := exec.CommandContext(ctx, "git", "init", "-b", "main")
	cmd.Dir = repoPath

	_, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to regular init + branch rename for older Git versions
		cmd = exec.CommandContext(ctx, "git", "init")
		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			return errors.NewGitOperationError("init", fmt.Errorf("git init failed: %w\nOutput: %s", err, string(output)))
		}

		// Set the default branch to main
		cmd = exec.CommandContext(ctx, "git", "symbolic-ref", "HEAD", "refs/heads/main")
		cmd.Dir = repoPath

		if err := cmd.Run(); err != nil {
			return errors.NewGitOperationError("init", fmt.Errorf("failed to set default branch to main: %w", err))
		}
	}

	return nil
}

// Clone clones a repository from url to repoPath
func (g *GitManager) Clone(ctx context.Context, repoPath, url string) error {
	// Remove the directory if it exists to ensure clean clone
	if err := os.RemoveAll(repoPath); err != nil {
		return errors.NewFileSystemOperationError("remove_existing_dir", repoPath,
			fmt.Errorf("failed to remove existing directory: %w", err))
	}

	// Create parent directory
	parentDir := filepath.Dir(repoPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return errors.NewFileSystemOperationError("create_parent_dir", parentDir,
			fmt.Errorf("failed to create parent directory: %w", err))
	}

	// Clone the repository
	cmd := exec.CommandContext(ctx, "git", "clone", url, repoPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.NewGitOperationError("clone", fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output)))
	}

	// Set up upstream tracking for main branch
	cmd = exec.CommandContext(ctx, "git", "branch", "--set-upstream-to=origin/main", "main")
	cmd.Dir = repoPath
	_, err = cmd.CombinedOutput()
	if err != nil {
		// If main doesn't exist, try master
		cmd = exec.CommandContext(ctx, "git", "branch", "--set-upstream-to=origin/master", "master")
		cmd.Dir = repoPath
		_, err = cmd.CombinedOutput()
		if err != nil {
			// If that also fails, try to set upstream for current branch
			cmd = exec.CommandContext(ctx, "git", "branch", "--set-upstream-to=origin/HEAD")
			cmd.Dir = repoPath
			_, _ = cmd.CombinedOutput() // Ignore error as this is best effort
		}
	}

	return nil
}

// Add stages files for commit
func (g *GitManager) Add(ctx context.Context, repoPath string, files ...string) error {
	args := append([]string{"add"}, files...)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.NewGitOperationError("add", fmt.Errorf("git add failed: %w\nOutput: %s", err, string(output)))
	}

	return nil
}

// Remove removes files from Git tracking
func (g *GitManager) Remove(ctx context.Context, repoPath string, files ...string) error {
	for _, filename := range files {
		// Check if it's a directory in the repository by checking the actual repo path
		fullPath := filepath.Join(repoPath, filename)
		info, err := os.Stat(fullPath)

		var cmd *exec.Cmd
		useRecursive := false
		if err == nil && info.IsDir() {
			useRecursive = true
		}

		if useRecursive {
			// Use -r and --cached flags for directories (only remove from git, not fs)
			cmd = exec.CommandContext(ctx, "git", "rm", "-r", "--cached", filename)
		} else {
			// Regular file (only remove from git, not fs)
			cmd = exec.CommandContext(ctx, "git", "rm", "--cached", filename)
		}

		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			// If we tried without -r and got a "recursively without -r" error, try with -r
			if !useRecursive && strings.Contains(string(output), "recursively without -r") {
				cmd = exec.CommandContext(ctx, "git", "rm", "-r", "--cached", filename)
				cmd.Dir = repoPath
				output, err = cmd.CombinedOutput()
			}

			if err != nil {
				return errors.NewGitOperationError("remove", fmt.Errorf("git rm failed: %w\nOutput: %s", err, string(output)))
			}
		}
	}

	return nil
}

// Commit creates a commit with the given message
func (g *GitManager) Commit(ctx context.Context, repoPath, message string) error {
	// Configure git user if not already configured
	if err := g.ensureGitConfig(ctx, repoPath); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.NewGitOperationError("commit", fmt.Errorf("git commit failed: %w\nOutput: %s", err, string(output)))
	}

	return nil
}

// Push pushes changes to the remote repository
func (g *GitManager) Push(ctx context.Context, repoPath string) error {
	// First ensure we have a remote configured
	_, err := g.GetRemoteURL(ctx, repoPath, "origin")
	if err != nil {
		return errors.NewGitOperationError("push", fmt.Errorf("cannot push: %w", err))
	}

	cmd := exec.CommandContext(ctx, "git", "push", "-u", "origin", "main")
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.NewGitOperationError("push", fmt.Errorf("git push failed: %w\nOutput: %s", err, string(output)))
	}

	return nil
}

// Pull pulls changes from the remote repository
func (g *GitManager) Pull(ctx context.Context, repoPath string) error {
	// First ensure we have a remote configured
	_, err := g.GetRemoteURL(ctx, repoPath, "origin")
	if err != nil {
		return errors.NewGitOperationError("pull", fmt.Errorf("cannot pull: %w", err))
	}

	cmd := exec.CommandContext(ctx, "git", "pull", "origin", "main")
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.NewGitOperationError("pull", fmt.Errorf("git pull failed: %w\nOutput: %s", err, string(output)))
	}

	return nil
}

// Status returns the current Git status
func (g *GitManager) Status(ctx context.Context, repoPath string) (*models.SyncStatus, error) {
	// First check if we have a remote configured - this should match old behavior
	_, err := g.GetRemoteURL(ctx, repoPath, "origin")
	if err != nil {
		// If origin doesn't exist, check if we have any remotes at all
		cmd := exec.CommandContext(ctx, "git", "remote")
		cmd.Dir = repoPath

		output, err := cmd.Output()
		if err != nil {
			return nil, errors.NewGitOperationError("list_remotes", err)
		}

		remotes := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(remotes) == 0 || remotes[0] == "" {
			return nil, errors.NewGitOperationError("status", fmt.Errorf("no remote configured"))
		}
	}

	// Get current branch
	currentBranch, err := g.getCurrentBranch(ctx, repoPath)
	if err != nil {
		return nil, errors.NewGitOperationError("get_current_branch", err)
	}

	// Check for uncommitted changes
	dirty, err := g.HasChanges(ctx, repoPath)
	if err != nil {
		return nil, errors.NewGitOperationError("check_changes", err)
	}

	// Get the remote URL
	remoteURL, err := g.GetRemoteURL(ctx, repoPath, "origin")
	hasRemote := err == nil

	// Initialize status with basic information
	status := &models.SyncStatus{
		CurrentBranch: currentBranch,
		Dirty:         dirty,
		HasRemote:     hasRemote,
		RemoteURL:     remoteURL,
	}

	// If no remote, we can't determine ahead/behind counts
	if !hasRemote {
		return status, nil
	}

	// Get the remote tracking branch
	remoteBranch, err := g.getRemoteTrackingBranch(ctx, repoPath)
	if err != nil {
		// No upstream branch set, assume origin/main
		remoteBranch = "origin/main"
	}
	status.RemoteBranch = remoteBranch

	// Get ahead/behind counts
	status.Ahead = g.getAheadCount(ctx, repoPath, remoteBranch)
	status.Behind = g.getBehindCount(ctx, repoPath, remoteBranch)

	// Get last commit hash
	lastCommitHash, err := g.getLastCommitHash(ctx, repoPath)
	if err == nil {
		status.LastCommitHash = lastCommitHash
	}

	return status, nil
}

// IsRepository checks if the path is a Git repository
func (g *GitManager) IsRepository(ctx context.Context, repoPath string) (bool, error) {
	gitDir := filepath.Join(repoPath, ".git")
	_, err := os.Stat(gitDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.NewFileSystemOperationError("check_git_dir", gitDir, err)
	}
	return true, nil
}

// HasChanges checks if there are uncommitted changes
func (g *GitManager) HasChanges(ctx context.Context, repoPath string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return false, errors.NewGitOperationError("status", fmt.Errorf("git status failed: %w", err))
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}

// AddRemote adds a remote to the repository
func (g *GitManager) AddRemote(ctx context.Context, repoPath, name, url string) error {
	// Check if remote already exists
	existingURL, err := g.GetRemoteURL(ctx, repoPath, name)
	if err == nil {
		// Remote exists, check if URL matches
		if existingURL == url {
			// Same URL, idempotent - do nothing
			return nil
		}
		// Different URL, error
		return errors.NewGitOperationError("add_remote",
			fmt.Errorf("remote %s already exists with different URL: %s (trying to add: %s)", name, existingURL, url))
	}

	// Remote doesn't exist, add it
	cmd := exec.CommandContext(ctx, "git", "remote", "add", name, url)
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.NewGitOperationError("add_remote", fmt.Errorf("git remote add failed: %w\nOutput: %s", err, string(output)))
	}

	return nil
}

// GetRemoteURL returns the URL of a remote
func (g *GitManager) GetRemoteURL(ctx context.Context, repoPath, name string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", name)
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", errors.NewGitOperationError("get_remote_url", fmt.Errorf("failed to get remote URL for %s: %w", name, err))
	}

	return strings.TrimSpace(string(output)), nil
}

// IsLnkRepository checks if the repository appears to be managed by lnk
func (g *GitManager) IsLnkRepository(ctx context.Context, repoPath string) (bool, error) {
	isRepo, err := g.IsRepository(ctx, repoPath)
	if err != nil {
		return false, err
	}
	if !isRepo {
		return false, nil
	}

	// Check if this looks like a lnk repository
	// We consider it a lnk repo if:
	// 1. It has no commits (fresh repo), OR
	// 2. All commits start with "lnk:" pattern

	commits, err := g.getCommits(ctx, repoPath)
	if err != nil {
		return false, errors.NewGitOperationError("get_commits", err)
	}

	// If no commits, it's a fresh repo - could be lnk
	if len(commits) == 0 {
		return true, nil
	}

	// If all commits start with "lnk:", it's definitely ours
	// If ANY commit doesn't start with "lnk:", it's probably not ours
	for _, commit := range commits {
		if !strings.HasPrefix(commit, "lnk:") {
			return false, nil
		}
	}

	return true, nil
}

// Helper methods

// ensureGitConfig configures git user if not already configured
func (g *GitManager) ensureGitConfig(ctx context.Context, repoPath string) error {
	// Check if user.name is configured
	cmd := exec.CommandContext(ctx, "git", "config", "user.name")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		// Set default user.name
		cmd = exec.CommandContext(ctx, "git", "config", "user.name", "lnk")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			return errors.NewGitOperationError("config_user_name", fmt.Errorf("failed to set git user.name: %w", err))
		}
	}

	// Check if user.email is configured
	cmd = exec.CommandContext(ctx, "git", "config", "user.email")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		// Set default user.email
		cmd = exec.CommandContext(ctx, "git", "config", "user.email", "lnk@local")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			return errors.NewGitOperationError("config_user_email", fmt.Errorf("failed to set git user.email: %w", err))
		}
	}

	return nil
}

// getCurrentBranch returns the current branch name
func (g *GitManager) getCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// For empty repositories, HEAD might not exist yet, default to main
		errStr := string(output)

		if strings.Contains(errStr, "fatal: ambiguous argument 'HEAD'") ||
			strings.Contains(errStr, "unknown revision") ||
			strings.Contains(errStr, "not a valid ref") ||
			strings.Contains(errStr, "bad revision") {
			return "main", nil
		}
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	// If the branch is HEAD (detached state), try to get the default branch
	if branch == "HEAD" {
		return "main", nil
	}

	return branch, nil
}

// getRemoteTrackingBranch returns the remote tracking branch
func (g *GitManager) getRemoteTrackingBranch(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no upstream branch set: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// getAheadCount returns how many commits ahead of remote
func (g *GitManager) getAheadCount(ctx context.Context, repoPath, remoteBranch string) int {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", fmt.Sprintf("%s..HEAD", remoteBranch))
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		// If remote branch doesn't exist, count all local commits
		cmd = exec.CommandContext(ctx, "git", "rev-list", "--count", "HEAD")
		cmd.Dir = repoPath

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
func (g *GitManager) getBehindCount(ctx context.Context, repoPath, remoteBranch string) int {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", fmt.Sprintf("HEAD..%s", remoteBranch))
	cmd.Dir = repoPath

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

// getLastCommitHash returns the hash of the last commit
func (g *GitManager) getLastCommitHash(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get last commit hash: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// getCommits returns commit messages
func (g *GitManager) getCommits(ctx context.Context, repoPath string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "--pretty=format:%s")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		// If there are no commits, git log will fail
		// Use CombinedOutput to get both stdout and stderr to check the error message
		cmd = exec.CommandContext(ctx, "git", "log", "--pretty=format:%s")
		cmd.Dir = repoPath
		combinedOutput, _ := cmd.CombinedOutput()
		errStr := string(combinedOutput)

		if strings.Contains(errStr, "does not have any commits yet") ||
			strings.Contains(errStr, "bad default revision") ||
			strings.Contains(errStr, "unknown revision") ||
			strings.Contains(errStr, "ambiguous argument") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return []string{}, nil
	}

	commitMessages := strings.Split(outputStr, "\n")
	return commitMessages, nil
}
