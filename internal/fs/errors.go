package fs

import "github.com/yarlson/lnk/internal/errors"

// Re-export shared error types for backward compatibility
// These will be removed once all internal usages are updated

type FileNotExistsError = errors.FileNotExistsError
type FileCheckError = errors.FileCheckError
type UnsupportedFileTypeError = errors.UnsupportedFileTypeError
type NotManagedByLnkError = errors.NotManagedByLnkError
type SymlinkReadError = errors.SymlinkReadError
type DirectoryCreationError = errors.DirectoryCreationError
type RelativePathCalculationError = errors.RelativePathCalculationError
