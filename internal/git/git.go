package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// Init initializes a new Git repository
func (g *Git) Init() error {
	// Try using git init -b main first (Git 2.28+)
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = g.repoPath

	_, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to regular init + branch rename for older Git versions
		cmd = exec.Command("git", "init")
		cmd.Dir = g.repoPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git init failed: %w\nOutput: %s", err, string(output))
		}

		// Set the default branch to main
		cmd = exec.Command("git", "symbolic-ref", "HEAD", "refs/heads/main")
		cmd.Dir = g.repoPath

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set default branch to main: %w", err)
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
		return fmt.Errorf("remote %s already exists with different URL: %s (trying to add: %s)", name, existingURL, url)
	}

	// Remote doesn't exist, add it
	cmd := exec.Command("git", "remote", "add", name, url)
	cmd.Dir = g.repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git remote add failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// getRemoteURL returns the URL for a remote, or error if not found
func (g *Git) getRemoteURL(name string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", name)
	cmd.Dir = g.repoPath

	output, err := cmd.Output()
	if err != nil {
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
	if err := g.add(filename); err != nil {
		return err
	}

	// Commit the changes
	if err := g.commit(message); err != nil {
		return err
	}

	return nil
}

// RemoveAndCommit removes a file from Git and commits the change
func (g *Git) RemoveAndCommit(filename, message string) error {
	// Remove the file from Git
	if err := g.remove(filename); err != nil {
		return err
	}

	// Commit the changes
	if err := g.commit(message); err != nil {
		return err
	}

	return nil
}

// add stages a file
func (g *Git) add(filename string) error {
	cmd := exec.Command("git", "add", filename)
	cmd.Dir = g.repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// remove removes a file from Git tracking
func (g *Git) remove(filename string) error {
	cmd := exec.Command("git", "rm", filename)
	cmd.Dir = g.repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git rm failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// commit creates a commit with the given message
func (g *Git) commit(message string) error {
	// Configure git user if not already configured
	if err := g.ensureGitConfig(); err != nil {
		return err
	}

	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = g.repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// ensureGitConfig ensures that git user.name and user.email are configured
func (g *Git) ensureGitConfig() error {
	// Check if user.name is configured
	cmd := exec.Command("git", "config", "user.name")
	cmd.Dir = g.repoPath
	if output, err := cmd.Output(); err != nil || len(strings.TrimSpace(string(output))) == 0 {
		// Set a default user.name
		cmd = exec.Command("git", "config", "user.name", "Lnk User")
		cmd.Dir = g.repoPath
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set git user.name: %w", err)
		}
	}

	// Check if user.email is configured
	cmd = exec.Command("git", "config", "user.email")
	cmd.Dir = g.repoPath
	if output, err := cmd.Output(); err != nil || len(strings.TrimSpace(string(output))) == 0 {
		// Set a default user.email
		cmd = exec.Command("git", "config", "user.email", "lnk@localhost")
		cmd.Dir = g.repoPath
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set git user.email: %w", err)
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

	cmd := exec.Command("git", "log", "--oneline", "--format=%s")
	cmd.Dir = g.repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// If there are no commits yet, return empty slice
		outputStr := string(output)
		if strings.Contains(outputStr, "does not have any commits yet") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	commits := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(commits) == 1 && commits[0] == "" {
		return []string{}, nil
	}

	return commits, nil
}
