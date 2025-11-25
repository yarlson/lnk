package git

import "github.com/yarlson/lnk/internal/errors"

// Re-export shared error types for backward compatibility
// These will be removed once all internal usages are updated

type GitInitError = errors.GitInitError
type BranchSetupError = errors.BranchSetupError
type RemoteExistsError = errors.RemoteExistsError
type GitCommandError = errors.GitCommandError
type NoRemoteError = errors.NoRemoteError
type RemoteNotFoundError = errors.RemoteNotFoundError
type GitConfigError = errors.GitConfigError
type UncommittedChangesError = errors.UncommittedChangesError
type DirectoryRemovalError = errors.DirectoryRemovalError
type DirectoryCreationError = errors.DirectoryCreationError
type PushError = errors.PushError
type PullError = errors.PullError
