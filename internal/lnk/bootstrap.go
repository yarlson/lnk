package lnk

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/yarlson/lnk/internal/lnkerror"
)

// FindBootstrapScript searches for a bootstrap script in the repository
func (l *Lnk) FindBootstrapScript() (string, error) {
	// Check if repository is initialized
	if !l.git.IsGitRepository() {
		return "", lnkerror.WithSuggestion(ErrNotInitialized, "run 'lnk init' first")
	}

	// Look for bootstrap.sh - simple, opinionated choice
	scriptPath := filepath.Join(l.repoPath, "bootstrap.sh")
	if _, err := os.Stat(scriptPath); err == nil {
		return "bootstrap.sh", nil
	}

	return "", nil // No bootstrap script found
}

// RunBootstrapScript executes the bootstrap script with configurable I/O
func (l *Lnk) RunBootstrapScript(scriptName string, stdout, stderr io.Writer, stdin io.Reader) error {
	scriptPath := filepath.Join(l.repoPath, scriptName)

	// Verify the script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return lnkerror.WithPath(ErrBootstrapNotFound, scriptName)
	}

	// Make sure it's executable
	if err := os.Chmod(scriptPath, 0755); err != nil {
		return lnkerror.Wrap(ErrBootstrapPerms)
	}

	// Run with bash (since we only support bootstrap.sh)
	cmd := exec.Command("bash", scriptPath)

	// Set working directory to the repository
	cmd.Dir = l.repoPath

	// Connect to provided I/O streams
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin

	// Run the script
	if err := cmd.Run(); err != nil {
		return lnkerror.WithSuggestion(ErrBootstrapFailed, err.Error())
	}

	return nil
}
