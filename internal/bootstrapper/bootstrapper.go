// Package bootstrapper handles bootstrap script discovery and execution.
package bootstrapper

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/yarlson/lnk/internal/git"
	"github.com/yarlson/lnk/internal/lnkerror"
)

// Runner handles bootstrap script discovery and execution.
type Runner struct {
	repoPath string
	git      *git.Git
}

// New creates a new bootstrap Runner.
func New(repoPath string, g *git.Git) *Runner {
	return &Runner{
		repoPath: repoPath,
		git:      g,
	}
}

// FindScript searches for a bootstrap script in the repository.
func (r *Runner) FindScript() (string, error) {
	if !r.git.IsGitRepository() {
		return "", lnkerror.WithSuggestion(lnkerror.ErrNotInitialized, "run 'lnk init' first")
	}

	scriptPath := filepath.Join(r.repoPath, "bootstrap.sh")
	if _, err := os.Stat(scriptPath); err == nil {
		return "bootstrap.sh", nil
	}

	return "", nil
}

// RunScript executes the bootstrap script with configurable I/O.
func (r *Runner) RunScript(scriptName string, stdout, stderr io.Writer, stdin io.Reader) error {
	scriptPath := filepath.Join(r.repoPath, scriptName)

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return lnkerror.WithPath(lnkerror.ErrBootstrapNotFound, scriptName)
	}

	if err := os.Chmod(scriptPath, 0755); err != nil {
		return lnkerror.Wrap(lnkerror.ErrBootstrapPerms)
	}

	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = r.repoPath
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin

	if err := cmd.Run(); err != nil {
		return lnkerror.WithSuggestion(lnkerror.ErrBootstrapFailed, err.Error())
	}

	return nil
}
