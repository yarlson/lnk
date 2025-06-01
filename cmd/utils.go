package cmd

import (
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	
	"github.com/yarlson/lnk/internal/errors"
)

// printf is a helper function to simplify output formatting in commands
func printf(cmd *cobra.Command, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), format, args...)
}

// formatError provides user-friendly error formatting while preserving specific error messages for tests
func formatError(err error) error {
	if err == nil {
		return nil
	}

	// Handle typed LnkError first
	var lnkErr *errors.LnkError
	if stderrors.As(err, &lnkErr) {
		return formatLnkError(lnkErr)
	}

	// Handle other error patterns with improved messages
	errMsg := err.Error()

	// Git-related errors
	if strings.Contains(errMsg, "git") {
		if strings.Contains(errMsg, "no remote configured") {
			return fmt.Errorf("🚫 no remote configured\n   💡 Add a remote first: \033[1mgit remote add origin <url>\033[0m in \033[36m~/.config/lnk\033[0m")
		}
		if strings.Contains(errMsg, "authentication") || strings.Contains(errMsg, "permission denied") {
			return fmt.Errorf("🔐 \033[31mGit authentication failed\033[0m\n   💡 Check your SSH keys or credentials: \033[36mhttps://docs.github.com/en/authentication\033[0m")
		}
		if strings.Contains(errMsg, "not found") && strings.Contains(errMsg, "remote") {
			return fmt.Errorf("🌐 \033[31mRemote repository not found\033[0m\n   💡 Verify the repository URL is correct and you have access")
		}
	}

	// Service initialization errors
	if strings.Contains(errMsg, "failed to initialize lnk service") {
		return fmt.Errorf("⚠️  \033[31mFailed to initialize lnk\033[0m\n   💡 This is likely a system configuration issue. Please check permissions and try again.")
	}

	// Return original error for unhandled cases to maintain test compatibility
	return err
}

// formatLnkError formats typed LnkError instances with user-friendly messages
func formatLnkError(lnkErr *errors.LnkError) error {
	switch lnkErr.Code {
	case errors.ErrorCodeFileNotFound:
		// Preserve "File does not exist" for test compatibility but add consistent colors
		if path, ok := lnkErr.Context["path"].(string); ok {
			return fmt.Errorf("❌ \033[31mFile does not exist:\033[0m \033[36m%s\033[0m\n   💡 Check the file path and try again", path)
		}
		return fmt.Errorf("❌ \033[31mFile does not exist\033[0m\n   💡 Check the file path and try again")

	case errors.ErrorCodeRepoNotInitialized:
		// Preserve "Lnk repository not initialized" for test compatibility but add consistent colors
		return fmt.Errorf("📦 \033[31mLnk repository not initialized\033[0m\n   💡 Run \033[1mlnk init\033[0m to get started")

	case errors.ErrorCodeNotSymlink:
		// Preserve "not a symlink" for test compatibility but add consistent colors
		return fmt.Errorf("🔗 \033[31mnot a symlink\033[0m\n   💡 Only files managed by lnk can be removed. Use \033[1mlnk list\033[0m to see managed files")

	case errors.ErrorCodeFileAlreadyManaged:
		if path, ok := lnkErr.Context["path"].(string); ok {
			return fmt.Errorf("✨ \033[33mFile is already managed by lnk:\033[0m \033[36m%s\033[0m\n   💡 Use \033[1mlnk list\033[0m to see all managed files", path)
		}
		return fmt.Errorf("✨ \033[33mFile is already managed by lnk\033[0m\n   💡 Use \033[1mlnk list\033[0m to see all managed files")

	case errors.ErrorCodeNoRemoteConfigured:
		// Preserve "no remote configured" for test compatibility but add consistent colors
		return fmt.Errorf("🚫 \033[31mno remote configured\033[0m\n   💡 Add a remote first: \033[1mgit remote add origin <url>\033[0m in \033[36m~/.config/lnk\033[0m")

	case errors.ErrorCodePermissionDenied:
		if path, ok := lnkErr.Context["path"].(string); ok {
			return fmt.Errorf("🔒 \033[31mPermission denied:\033[0m \033[36m%s\033[0m\n   💡 Check file permissions or run with appropriate privileges", path)
		}
		return fmt.Errorf("🔒 \033[31mPermission denied\033[0m\n   💡 Check file permissions or run with appropriate privileges")

	case errors.ErrorCodeGitOperation:
		// Check if this is a "no remote configured" case by examining the underlying error first
		if lnkErr.Cause != nil && strings.Contains(lnkErr.Cause.Error(), "no remote configured") {
			return fmt.Errorf("🚫 \033[31mno remote configured\033[0m\n   💡 Add a remote first: \033[1mgit remote add origin <url>\033[0m in \033[36m~/.config/lnk\033[0m")
		}

		operation := lnkErr.Context["operation"]
		if op, ok := operation.(string); ok {
			switch op {
			case "get_status", "status":
				return fmt.Errorf("🔧 \033[31mGit operation failed\033[0m\n   💡 Run \033[1mgit status\033[0m in \033[36m~/.config/lnk\033[0m for details")
			case "push_to_remote", "push":
				return fmt.Errorf("🚀 \033[31mGit operation failed\033[0m\n   💡 Check your internet connection and Git credentials\n   💡 Run \033[1mgit status\033[0m in \033[36m~/.config/lnk\033[0m for details")
			case "pull_from_remote", "pull":
				return fmt.Errorf("⬇️  \033[31mGit operation failed\033[0m\n   💡 Check your internet connection and resolve any conflicts\n   💡 Run \033[1mgit status\033[0m in \033[36m~/.config/lnk\033[0m for details")
			case "clone_repository", "clone":
				return fmt.Errorf("📥 \033[31mGit operation failed\033[0m\n   💡 Check the repository URL and your access permissions\n   💡 Ensure you have the correct SSH keys or credentials")
			case "commit_changes", "commit":
				return fmt.Errorf("💾 \033[31mGit operation failed\033[0m\n   💡 Check if you have Git user.name and user.email configured\n   💡 Run \033[1mgit config --global user.name \"Your Name\"\033[0m")
			default:
				return fmt.Errorf("🔧 \033[31mGit operation failed\033[0m\n   💡 Run \033[1mgit status\033[0m in \033[36m~/.config/lnk\033[0m for details")
			}
		}
		return fmt.Errorf("🔧 \033[31mGit operation failed\033[0m\n   💡 Run \033[1mgit status\033[0m in \033[36m~/.config/lnk\033[0m for details")

	case errors.ErrorCodeFileSystemOperation:
		operation := lnkErr.Context["operation"]
		path := lnkErr.Context["path"]

		// Determine user-friendly message based on operation and underlying cause
		if op, ok := operation.(string); ok {
			switch op {
			case "stat_symlink", "check_file_exists":
				// Use consistent "File does not exist" messaging
				if pathStr, pathOk := path.(string); pathOk {
					return fmt.Errorf("❌ \033[31mFile does not exist:\033[0m \033[36m%s\033[0m\n   💡 Check the file path and try again", pathStr)
				}
				return fmt.Errorf("❌ \033[31mFile does not exist\033[0m\n   💡 Check the file path and try again")
			case "move_file":
				return fmt.Errorf("📁 \033[31mFile operation failed\033[0m\n   💡 Check file permissions and available disk space")
			case "create_symlink":
				return fmt.Errorf("🔗 \033[31mFile operation failed\033[0m\n   💡 Check directory permissions and ensure target file exists")
			case "remove_symlink", "remove_file":
				return fmt.Errorf("🗑️  \033[31mFile operation failed\033[0m\n   💡 Check file permissions and ensure file exists")
			case "read_symlink":
				return fmt.Errorf("🔗 \033[31mFile operation failed\033[0m\n   💡 The symlink may be broken or you don't have permission to read it")
			case "resolve_path", "get_relative_path":
				if pathStr, pathOk := path.(string); pathOk {
					return fmt.Errorf("📂 \033[31mInvalid file path:\033[0m \033[36m%s\033[0m\n   💡 Check the file path and try again", pathStr)
				}
				return fmt.Errorf("📂 \033[31mInvalid file path\033[0m\n   💡 Check the file path and try again")
			case "create_dest_dir", "create_repo_dir":
				return fmt.Errorf("📁 \033[31mFile operation failed\033[0m\n   💡 Check permissions and available disk space")
			default:
				// Don't expose cryptic operation names - give generic but helpful message
				return fmt.Errorf("💽 \033[31mFile operation failed\033[0m\n   💡 Check file permissions, paths, and available disk space")
			}
		}
		return fmt.Errorf("💽 \033[31mFile operation failed\033[0m\n   💡 Check file permissions and available disk space")

	case errors.ErrorCodeInvalidPath:
		if path, ok := lnkErr.Context["path"].(string); ok {
			return fmt.Errorf("📂 \033[31mInvalid file path:\033[0m \033[36m%s\033[0m\n   💡 Check the file path and try again", path)
		}
		return fmt.Errorf("📂 \033[31mInvalid file path\033[0m\n   💡 Check the file path and try again")

	default:
		// For unknown LnkError types, preserve original message but add context
		return fmt.Errorf("⚠️  \033[31m%s\033[0m", lnkErr.Error())
	}
}

// wrapServiceError wraps service errors with consistent messaging while preserving specific errors for tests
func wrapServiceError(operation string, err error) error {
	if err == nil {
		return nil
	}

	// For typed errors, format them nicely
	var lnkErr *errors.LnkError
	if stderrors.As(err, &lnkErr) {
		return formatLnkError(lnkErr)
	}

	// For other errors, provide operation context but preserve original message for tests
	return fmt.Errorf("failed to %s: %w", operation, err)
}
