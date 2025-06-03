package git

import "fmt"

// ANSI color codes for consistent formatting
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

// formatError creates a consistently formatted error message with ❌ prefix
func formatError(message string, args ...interface{}) string {
	return fmt.Sprintf("❌ "+message, args...)
}

// formatURL formats a URL with styling
func formatURL(url string) string {
	return fmt.Sprintf("%s%s%s", colorBold, url, colorReset)
}

// formatRemote formats a remote name with styling
func formatRemote(remote string) string {
	return fmt.Sprintf("%s%s%s", colorGreen, remote, colorReset)
}

// GitInitError represents an error during git initialization
type GitInitError struct {
	Output string
	Err    error
}

func (e *GitInitError) Error() string {
	return formatError("Failed to initialize git repository. Please ensure git is installed and try again.")
}

func (e *GitInitError) Unwrap() error {
	return e.Err
}

// BranchSetupError represents an error setting up the default branch
type BranchSetupError struct {
	Err error
}

func (e *BranchSetupError) Error() string {
	return formatError("Failed to set up the default branch. Please check your git installation.")
}

func (e *BranchSetupError) Unwrap() error {
	return e.Err
}

// RemoteExistsError represents an error when a remote already exists with different URL
type RemoteExistsError struct {
	Remote      string
	ExistingURL string
	NewURL      string
}

func (e *RemoteExistsError) Error() string {
	return formatError("Remote %s is already configured with a different repository (%s). Cannot add %s.",
		formatRemote(e.Remote), formatURL(e.ExistingURL), formatURL(e.NewURL))
}

func (e *RemoteExistsError) Unwrap() error {
	return nil
}

// GitCommandError represents a generic git command execution error
type GitCommandError struct {
	Command string
	Output  string
	Err     error
}

func (e *GitCommandError) Error() string {
	// Provide user-friendly messages based on common command types
	switch e.Command {
	case "add":
		return formatError("Failed to stage files for commit. Please check file permissions and try again.")
	case "commit":
		return formatError("Failed to create commit. Please ensure you have staged changes and try again.")
	case "remote add":
		return formatError("Failed to add remote repository. Please check the repository URL and try again.")
	case "rm":
		return formatError("Failed to remove file from git tracking. Please check if the file exists and try again.")
	case "log":
		return formatError("Failed to retrieve commit history.")
	case "remote":
		return formatError("Failed to retrieve remote repository information.")
	case "clone":
		return formatError("Failed to clone repository. Please check the repository URL and your network connection.")
	default:
		return formatError("Git operation failed. Please check your repository state and try again.")
	}
}

func (e *GitCommandError) Unwrap() error {
	return e.Err
}

// NoRemoteError represents an error when no remote is configured
type NoRemoteError struct{}

func (e *NoRemoteError) Error() string {
	return formatError("No remote repository is configured. Please add a remote repository first.")
}

func (e *NoRemoteError) Unwrap() error {
	return nil
}

// RemoteNotFoundError represents an error when a specific remote is not found
type RemoteNotFoundError struct {
	Remote string
	Err    error
}

func (e *RemoteNotFoundError) Error() string {
	return formatError("Remote repository %s is not configured.", formatRemote(e.Remote))
}

func (e *RemoteNotFoundError) Unwrap() error {
	return e.Err
}

// GitConfigError represents an error with git configuration
type GitConfigError struct {
	Setting string
	Err     error
}

func (e *GitConfigError) Error() string {
	return formatError("Failed to configure git settings. Please check your git installation.")
}

func (e *GitConfigError) Unwrap() error {
	return e.Err
}

// UncommittedChangesError represents an error checking for uncommitted changes
type UncommittedChangesError struct {
	Err error
}

func (e *UncommittedChangesError) Error() string {
	return formatError("Failed to check repository status. Please verify your git repository is valid.")
}

func (e *UncommittedChangesError) Unwrap() error {
	return e.Err
}

// DirectoryRemovalError represents an error removing a directory
type DirectoryRemovalError struct {
	Path string
	Err  error
}

func (e *DirectoryRemovalError) Error() string {
	return formatError("Failed to prepare directory for operation. Please check directory permissions.")
}

func (e *DirectoryRemovalError) Unwrap() error {
	return e.Err
}

// DirectoryCreationError represents an error creating a directory
type DirectoryCreationError struct {
	Path string
	Err  error
}

func (e *DirectoryCreationError) Error() string {
	return formatError("Failed to create directory. Please check permissions and available disk space.")
}

func (e *DirectoryCreationError) Unwrap() error {
	return e.Err
}

// PushError represents an error during git push operation
type PushError struct {
	Reason string
	Output string
	Err    error
}

func (e *PushError) Error() string {
	if e.Reason != "" {
		return formatError("Cannot push changes: %s", e.Reason)
	}
	return formatError("Failed to push changes to remote repository. Please check your network connection and repository permissions.")
}

func (e *PushError) Unwrap() error {
	return e.Err
}

// PullError represents an error during git pull operation
type PullError struct {
	Reason string
	Output string
	Err    error
}

func (e *PullError) Error() string {
	if e.Reason != "" {
		return formatError("Cannot pull changes: %s", e.Reason)
	}
	return formatError("Failed to pull changes from remote repository. Please check your network connection and resolve any conflicts.")
}

func (e *PullError) Unwrap() error {
	return e.Err
}
