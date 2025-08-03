package fs

// Structured errors that separate content from presentation
// These will be formatted by the cmd package based on user preferences

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

// GetPath returns the path for formatting purposes
func (e *FileNotExistsError) GetPath() string {
	return e.Path
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
	Path string
}

func (e *UnsupportedFileTypeError) Error() string {
	return "Cannot manage this type of file: " + e.Path
}

func (e *UnsupportedFileTypeError) GetPath() string {
	return e.Path
}

func (e *UnsupportedFileTypeError) GetSuggestion() string {
	return "lnk can only manage regular files and directories"
}

func (e *UnsupportedFileTypeError) Unwrap() error {
	return nil
}

// NotManagedByLnkError represents an error when a file is not managed by lnk
type NotManagedByLnkError struct {
	Path string
}

func (e *NotManagedByLnkError) Error() string {
	return "File is not managed by lnk: " + e.Path
}

func (e *NotManagedByLnkError) GetPath() string {
	return e.Path
}

func (e *NotManagedByLnkError) GetSuggestion() string {
	return "Use 'lnk add' to manage this file first"
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
	Operation string
	Err       error
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

// ErrorWithPath is an interface for errors that have an associated file path
type ErrorWithPath interface {
	error
	GetPath() string
}

// ErrorWithSuggestion is an interface for errors that provide helpful suggestions
type ErrorWithSuggestion interface {
	error
	GetSuggestion() string
}
