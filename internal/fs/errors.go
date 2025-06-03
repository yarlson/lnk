package fs

import "fmt"

// ANSI color codes for consistent formatting
const (
	colorReset = "\033[0m"
	colorRed   = "\033[31m"
	colorBold  = "\033[1m"
)

// formatError creates a consistently formatted error message with ❌ prefix
func formatError(message string, args ...interface{}) string {
	return fmt.Sprintf("❌ "+message, args...)
}

// formatPath formats a file path with red color
func formatPath(path string) string {
	return fmt.Sprintf("%s%s%s", colorRed, path, colorReset)
}

// formatCommand formats a command with bold styling
func formatCommand(command string) string {
	return fmt.Sprintf("%s%s%s", colorBold, command, colorReset)
}

// FileNotExistsError represents an error when a file does not exist
type FileNotExistsError struct {
	Path string
	Err  error
}

func (e *FileNotExistsError) Error() string {
	return formatError("File or directory not found: %s", formatPath(e.Path))
}

func (e *FileNotExistsError) Unwrap() error {
	return e.Err
}

// FileCheckError represents an error when failing to check a file
type FileCheckError struct {
	Err error
}

func (e *FileCheckError) Error() string {
	return formatError("Unable to access file. Please check file permissions and try again.")
}

func (e *FileCheckError) Unwrap() error {
	return e.Err
}

// UnsupportedFileTypeError represents an error when a file type is not supported
type UnsupportedFileTypeError struct {
	Path string
}

func (e *UnsupportedFileTypeError) Error() string {
	return formatError("Cannot manage this type of file: %s\n   💡 lnk can only manage regular files and directories", formatPath(e.Path))
}

func (e *UnsupportedFileTypeError) Unwrap() error {
	return nil
}

// NotManagedByLnkError represents an error when a file is not managed by lnk
type NotManagedByLnkError struct {
	Path string
}

func (e *NotManagedByLnkError) Error() string {
	return formatError("File is not managed by lnk: %s\n   💡 Use %s to manage this file first",
		formatPath(e.Path), formatCommand("lnk add"))
}

func (e *NotManagedByLnkError) Unwrap() error {
	return nil
}

// SymlinkReadError represents an error when failing to read a symlink
type SymlinkReadError struct {
	Err error
}

func (e *SymlinkReadError) Error() string {
	return formatError("Unable to read symlink. The file may be corrupted or have invalid permissions.")
}

func (e *SymlinkReadError) Unwrap() error {
	return e.Err
}

// DirectoryCreationError represents an error when failing to create a directory
type DirectoryCreationError struct {
	Operation string
	Err       error
}

func (e *DirectoryCreationError) Error() string {
	return formatError("Failed to create directory. Please check permissions and available disk space.")
}

func (e *DirectoryCreationError) Unwrap() error {
	return e.Err
}

// RelativePathCalculationError represents an error when failing to calculate relative path
type RelativePathCalculationError struct {
	Err error
}

func (e *RelativePathCalculationError) Error() string {
	return formatError("Unable to create symlink due to path configuration issues. Please check file locations.")
}

func (e *RelativePathCalculationError) Unwrap() error {
	return e.Err
}
