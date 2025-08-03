package git

// Structured errors that separate content from presentation
// These will be formatted by the cmd package based on user preferences

// GitInitError represents an error during git initialization
type GitInitError struct {
	Output string
	Err    error
}

func (e *GitInitError) Error() string {
	return "Failed to initialize git repository. Please ensure git is installed and try again."
}

func (e *GitInitError) Unwrap() error {
	return e.Err
}

// BranchSetupError represents an error setting up the default branch
type BranchSetupError struct {
	Err error
}

func (e *BranchSetupError) Error() string {
	return "Failed to set up the default branch. Please check your git installation."
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
	return "Remote " + e.Remote + " is already configured with a different repository (" + e.ExistingURL + "). Cannot add " + e.NewURL + "."
}

func (e *RemoteExistsError) GetRemote() string {
	return e.Remote
}

func (e *RemoteExistsError) GetExistingURL() string {
	return e.ExistingURL
}

func (e *RemoteExistsError) GetNewURL() string {
	return e.NewURL
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
		return "Failed to stage files for commit. Please check file permissions and try again."
	case "commit":
		return "Failed to create commit. Please ensure you have staged changes and try again."
	case "remote add":
		return "Failed to add remote repository. Please check the repository URL and try again."
	case "rm":
		return "Failed to remove file from git tracking. Please check if the file exists and try again."
	case "log":
		return "Failed to retrieve commit history."
	case "remote":
		return "Failed to retrieve remote repository information."
	case "clone":
		return "Failed to clone repository. Please check the repository URL and your network connection."
	default:
		return "Git operation failed. Please check your repository state and try again."
	}
}

func (e *GitCommandError) GetCommand() string {
	return e.Command
}

func (e *GitCommandError) Unwrap() error {
	return e.Err
}

// NoRemoteError represents an error when no remote is configured
type NoRemoteError struct{}

func (e *NoRemoteError) Error() string {
	return "No remote repository is configured. Please add a remote repository first."
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
	return "Remote repository " + e.Remote + " is not configured."
}

func (e *RemoteNotFoundError) GetRemote() string {
	return e.Remote
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
	return "Failed to configure git settings. Please check your git installation."
}

func (e *GitConfigError) Unwrap() error {
	return e.Err
}

// UncommittedChangesError represents an error checking for uncommitted changes
type UncommittedChangesError struct {
	Err error
}

func (e *UncommittedChangesError) Error() string {
	return "Failed to check repository status. Please verify your git repository is valid."
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
	return "Failed to prepare directory for operation. Please check directory permissions."
}

func (e *DirectoryRemovalError) GetPath() string {
	return e.Path
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
	return "Failed to create directory. Please check permissions and available disk space."
}

func (e *DirectoryCreationError) GetPath() string {
	return e.Path
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
		return "Cannot push changes: " + e.Reason
	}
	return "Failed to push changes to remote repository. Please check your network connection and repository permissions."
}

func (e *PushError) GetReason() string {
	return e.Reason
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
		return "Cannot pull changes: " + e.Reason
	}
	return "Failed to pull changes from remote repository. Please check your network connection and resolve any conflicts."
}

func (e *PullError) GetReason() string {
	return e.Reason
}

func (e *PullError) Unwrap() error {
	return e.Err
}

// ErrorWithPath is an interface for git errors that have an associated file path
type ErrorWithPath interface {
	error
	GetPath() string
}

// ErrorWithRemote is an interface for git errors that involve a remote
type ErrorWithRemote interface {
	error
	GetRemote() string
}

// ErrorWithReason is an interface for git errors that have a specific reason
type ErrorWithReason interface {
	error
	GetReason() string
}
