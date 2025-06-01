package errors

import (
	"errors"
	"fmt"
)

// Standard error variables
var (
	// ErrFileNotFound indicates a file or directory was not found
	ErrFileNotFound = errors.New("file not found")

	// ErrFileAlreadyManaged indicates a file is already being managed by lnk
	ErrFileAlreadyManaged = errors.New("file already managed")

	// ErrNotSymlink indicates the file is not a symbolic link
	ErrNotSymlink = errors.New("not a symlink")

	// ErrRepoNotInitialized indicates the lnk repository has not been initialized
	ErrRepoNotInitialized = errors.New("repository not initialized")

	// ErrNoRemoteConfigured indicates no Git remote has been configured
	ErrNoRemoteConfigured = errors.New("no remote configured")

	// ErrOperationAborted indicates an operation was aborted by the user
	ErrOperationAborted = errors.New("operation aborted")

	// ErrConfigNotFound indicates a configuration file was not found
	ErrConfigNotFound = errors.New("configuration not found")

	// ErrInvalidPath indicates an invalid file path was provided
	ErrInvalidPath = errors.New("invalid path")

	// ErrPermissionDenied indicates insufficient permissions for the operation
	ErrPermissionDenied = errors.New("permission denied")

	// ErrGitOperation indicates a Git operation failed
	ErrGitOperation = errors.New("git operation failed")

	// ErrFileSystemOperation indicates a file system operation failed
	ErrFileSystemOperation = errors.New("file system operation failed")
)

// ErrorCode represents different types of errors that can occur
type ErrorCode int

const (
	// ErrorCodeUnknown represents an unknown error
	ErrorCodeUnknown ErrorCode = iota

	// ErrorCodeFileNotFound represents file not found errors
	ErrorCodeFileNotFound

	// ErrorCodeFileAlreadyManaged represents file already managed errors
	ErrorCodeFileAlreadyManaged

	// ErrorCodeNotSymlink represents not a symlink errors
	ErrorCodeNotSymlink

	// ErrorCodeRepoNotInitialized represents repository not initialized errors
	ErrorCodeRepoNotInitialized

	// ErrorCodeNoRemoteConfigured represents no remote configured errors
	ErrorCodeNoRemoteConfigured

	// ErrorCodeOperationAborted represents operation aborted errors
	ErrorCodeOperationAborted

	// ErrorCodeConfigNotFound represents configuration not found errors
	ErrorCodeConfigNotFound

	// ErrorCodeInvalidPath represents invalid path errors
	ErrorCodeInvalidPath

	// ErrorCodePermissionDenied represents permission denied errors
	ErrorCodePermissionDenied

	// ErrorCodeGitOperation represents Git operation errors
	ErrorCodeGitOperation

	// ErrorCodeFileSystemOperation represents file system operation errors
	ErrorCodeFileSystemOperation
)

// String returns a string representation of the error code
func (e ErrorCode) String() string {
	switch e {
	case ErrorCodeFileNotFound:
		return "FILE_NOT_FOUND"
	case ErrorCodeFileAlreadyManaged:
		return "FILE_ALREADY_MANAGED"
	case ErrorCodeNotSymlink:
		return "NOT_SYMLINK"
	case ErrorCodeRepoNotInitialized:
		return "REPO_NOT_INITIALIZED"
	case ErrorCodeNoRemoteConfigured:
		return "NO_REMOTE_CONFIGURED"
	case ErrorCodeOperationAborted:
		return "OPERATION_ABORTED"
	case ErrorCodeConfigNotFound:
		return "CONFIG_NOT_FOUND"
	case ErrorCodeInvalidPath:
		return "INVALID_PATH"
	case ErrorCodePermissionDenied:
		return "PERMISSION_DENIED"
	case ErrorCodeGitOperation:
		return "GIT_OPERATION"
	case ErrorCodeFileSystemOperation:
		return "FILE_SYSTEM_OPERATION"
	default:
		return "UNKNOWN"
	}
}

// LnkError represents a structured error with additional context
type LnkError struct {
	// Code represents the type of error
	Code ErrorCode

	// Message is the human-readable error message
	Message string

	// Cause is the underlying error that caused this error
	Cause error

	// Context provides additional context about when/where the error occurred
	Context map[string]interface{}
}

// Error implements the error interface
func (e *LnkError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying cause error for Go 1.13+ error handling
func (e *LnkError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for Go 1.13+ error handling
func (e *LnkError) Is(target error) bool {
	if lnkErr, ok := target.(*LnkError); ok {
		return e.Code == lnkErr.Code
	}
	return errors.Is(e.Cause, target)
}

// WithContext adds context information to the error
func (e *LnkError) WithContext(key string, value interface{}) *LnkError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// NewLnkError creates a new LnkError with the given code and message
func NewLnkError(code ErrorCode, message string) *LnkError {
	return &LnkError{
		Code:    code,
		Message: message,
		Context: make(map[string]interface{}),
	}
}

// WrapError wraps an existing error with LnkError context
func WrapError(code ErrorCode, message string, cause error) *LnkError {
	return &LnkError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// Helper functions for creating common errors

// NewFileNotFoundError creates a file not found error
func NewFileNotFoundError(path string) *LnkError {
	return NewLnkError(ErrorCodeFileNotFound, fmt.Sprintf("❌ File does not exist: \033[31m%s\033[0m", path)).
		WithContext("path", path)
}

// NewFileAlreadyManagedError creates a file already managed error
func NewFileAlreadyManagedError(path string) *LnkError {
	return NewLnkError(ErrorCodeFileAlreadyManaged, fmt.Sprintf("file already managed: %s", path)).
		WithContext("path", path)
}

// NewNotSymlinkError creates a not symlink error
func NewNotSymlinkError(path string) *LnkError {
	return NewLnkError(ErrorCodeNotSymlink, fmt.Sprintf("not a symlink: %s", path)).
		WithContext("path", path)
}

// NewRepoNotInitializedError creates a repository not initialized error
func NewRepoNotInitializedError(repoPath string) *LnkError {
	return NewLnkError(ErrorCodeRepoNotInitialized, "Lnk repository not initialized").
		WithContext("repo_path", repoPath)
}

// NewNoRemoteConfiguredError creates a no remote configured error
func NewNoRemoteConfiguredError() *LnkError {
	return NewLnkError(ErrorCodeNoRemoteConfigured, "no git remote configured")
}

// NewConfigNotFoundError creates a configuration not found error
func NewConfigNotFoundError(host string) *LnkError {
	return NewLnkError(ErrorCodeConfigNotFound, fmt.Sprintf("configuration not found for host: %s", host)).
		WithContext("host", host)
}

// NewInvalidPathError creates an invalid path error
func NewInvalidPathError(path string, reason string) *LnkError {
	return NewLnkError(ErrorCodeInvalidPath, fmt.Sprintf("invalid path %s: %s", path, reason)).
		WithContext("path", path).
		WithContext("reason", reason)
}

// NewPermissionDeniedError creates a permission denied error
func NewPermissionDeniedError(operation, path string) *LnkError {
	return NewLnkError(ErrorCodePermissionDenied, fmt.Sprintf("permission denied for %s: %s", operation, path)).
		WithContext("operation", operation).
		WithContext("path", path)
}

// NewGitOperationError creates a Git operation error
func NewGitOperationError(operation string, cause error) *LnkError {
	return WrapError(ErrorCodeGitOperation, fmt.Sprintf("git %s failed", operation), cause).
		WithContext("operation", operation)
}

// NewFileSystemOperationError creates a file system operation error
func NewFileSystemOperationError(operation, path string, cause error) *LnkError {
	return WrapError(ErrorCodeFileSystemOperation, fmt.Sprintf("file system %s failed for %s", operation, path), cause).
		WithContext("operation", operation).
		WithContext("path", path)
}
