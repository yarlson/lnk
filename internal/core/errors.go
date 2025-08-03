package core

import "fmt"

// LnkError represents a structured error with separate content and formatting hints
type LnkError struct {
	Message    string
	Suggestion string
	Path       string
	ErrorType  string
}

func (e *LnkError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("%s\n   %s", e.Message, e.Suggestion)
	}
	return e.Message
}

// Error constructors that separate content from presentation

func ErrDirectoryContainsManagedFiles(path string) error {
	return &LnkError{
		Message:    fmt.Sprintf("Directory %s already contains managed files", path),
		Suggestion: "Use 'lnk pull' to update from remote instead of 'lnk init -r'",
		Path:       path,
		ErrorType:  "managed_files_exist",
	}
}

func ErrDirectoryContainsGitRepo(path string) error {
	return &LnkError{
		Message:    fmt.Sprintf("Directory %s contains an existing Git repository", path),
		Suggestion: "Please backup or move the existing repository before initializing lnk",
		Path:       path,
		ErrorType:  "git_repo_exists",
	}
}

func ErrFileAlreadyManaged(path string) error {
	return &LnkError{
		Message:   fmt.Sprintf("File is already managed by lnk: %s", path),
		Path:      path,
		ErrorType: "already_managed",
	}
}

func ErrFileNotManaged(path string) error {
	return &LnkError{
		Message:   fmt.Sprintf("File is not managed by lnk: %s", path),
		Path:      path,
		ErrorType: "not_managed",
	}
}

func ErrRepositoryNotInitialized() error {
	return &LnkError{
		Message:    "Lnk repository not initialized",
		Suggestion: "Run 'lnk init' first",
		ErrorType:  "not_initialized",
	}
}

func ErrBootstrapScriptNotFound(script string) error {
	return &LnkError{
		Message:   fmt.Sprintf("Bootstrap script not found: %s", script),
		Path:      script,
		ErrorType: "script_not_found",
	}
}

func ErrBootstrapScriptFailed(err error) error {
	return &LnkError{
		Message:   fmt.Sprintf("Bootstrap script failed with error: %v", err),
		ErrorType: "script_failed",
	}
}

func ErrBootstrapScriptNotExecutable(err error) error {
	return &LnkError{
		Message:   fmt.Sprintf("Failed to make bootstrap script executable: %v", err),
		ErrorType: "script_permissions",
	}
}
