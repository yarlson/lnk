package errors

import (
	"fmt"
	"time"
)

// Shared error types for fs and git packages
// These errors represent infrastructure-level failures

// File system errors

// FileNotExistsError represents an error when a file does not exist
type FileNotExistsError struct {
	Path string
	Err  error
}

func (e *FileNotExistsError) Error() string {
	return "File or directory not found: " + e.Path
}

func (e *FileNotExistsError) Unwrap() error {
	return e.Err
}

// FileCheckError represents an error when failing to check a file
type FileCheckError struct {
	Err error
}

func (e *FileCheckError) Error() string {
	return "Unable to access file. Please check file permissions and try again."
}

func (e *FileCheckError) Unwrap() error {
	return e.Err
}

// UnsupportedFileTypeError represents an error when a file type is not supported
type UnsupportedFileTypeError struct {
	Path       string
	Suggestion string
}

func (e *UnsupportedFileTypeError) Error() string {
	return "Cannot manage this type of file: " + e.Path
}

func (e *UnsupportedFileTypeError) Unwrap() error {
	return nil
}

// NotManagedByLnkError represents an error when a file is not managed by lnk
type NotManagedByLnkError struct {
	Path       string
	Suggestion string
}

func (e *NotManagedByLnkError) Error() string {
	return "File is not managed by lnk: " + e.Path
}

func (e *NotManagedByLnkError) Unwrap() error {
	return nil
}

// SymlinkReadError represents an error when failing to read a symlink
type SymlinkReadError struct {
	Err error
}

func (e *SymlinkReadError) Error() string {
	return "Unable to read symlink. The file may be corrupted or have invalid permissions."
}

func (e *SymlinkReadError) Unwrap() error {
	return e.Err
}

// DirectoryCreationError represents an error when failing to create a directory
type DirectoryCreationError struct {
	Path string
	Err  error
}

func (e *DirectoryCreationError) Error() string {
	return "Failed to create directory. Please check permissions and available disk space."
}

func (e *DirectoryCreationError) Unwrap() error {
	return e.Err
}

// RelativePathCalculationError represents an error when failing to calculate relative path
type RelativePathCalculationError struct {
	Err error
}

func (e *RelativePathCalculationError) Error() string {
	return "Unable to create symlink due to path configuration issues. Please check file locations."
}

func (e *RelativePathCalculationError) Unwrap() error {
	return e.Err
}

// Git-specific errors

// DirectoryRemovalError represents an error removing a directory
type DirectoryRemovalError struct {
	Path string
	Err  error
}

func (e *DirectoryRemovalError) Error() string {
	return "Failed to prepare directory for operation. Please check directory permissions."
}

func (e *DirectoryRemovalError) Unwrap() error {
	return e.Err
}

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

func (e *PullError) Unwrap() error {
	return e.Err
}

// GitTimeoutError represents a git operation that exceeded its timeout
type GitTimeoutError struct {
	Command string
	Timeout time.Duration
	Err     error
}

func (e *GitTimeoutError) Error() string {
	return fmt.Sprintf("git operation timed out after %v: %s", e.Timeout, e.Command)
}

func (e *GitTimeoutError) Unwrap() error {
	return e.Err
}
