package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ErrorsTestSuite struct {
	suite.Suite
}

func (suite *ErrorsTestSuite) TestErrorCodeString() {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{ErrorCodeFileNotFound, "FILE_NOT_FOUND"},
		{ErrorCodeFileAlreadyManaged, "FILE_ALREADY_MANAGED"},
		{ErrorCodeNotSymlink, "NOT_SYMLINK"},
		{ErrorCodeRepoNotInitialized, "REPO_NOT_INITIALIZED"},
		{ErrorCodeNoRemoteConfigured, "NO_REMOTE_CONFIGURED"},
		{ErrorCodeOperationAborted, "OPERATION_ABORTED"},
		{ErrorCodeConfigNotFound, "CONFIG_NOT_FOUND"},
		{ErrorCodeInvalidPath, "INVALID_PATH"},
		{ErrorCodePermissionDenied, "PERMISSION_DENIED"},
		{ErrorCodeGitOperation, "GIT_OPERATION"},
		{ErrorCodeFileSystemOperation, "FILE_SYSTEM_OPERATION"},
		{ErrorCodeUnknown, "UNKNOWN"},
	}

	for _, tt := range tests {
		suite.Run(tt.expected, func() {
			result := tt.code.String()
			suite.Equal(tt.expected, result)
		})
	}
}

func (suite *ErrorsTestSuite) TestLnkErrorError() {
	suite.Run("without_cause", func() {
		err := NewLnkError(ErrorCodeFileNotFound, "test file not found")
		expected := "test file not found"
		suite.Equal(expected, err.Error())
	})

	suite.Run("with_cause", func() {
		cause := errors.New("underlying error")
		err := WrapError(ErrorCodeFileSystemOperation, "file operation failed", cause)
		expected := "file operation failed: underlying error"
		suite.Equal(expected, err.Error())
	})
}

func (suite *ErrorsTestSuite) TestLnkErrorUnwrap() {
	cause := errors.New("underlying error")
	err := WrapError(ErrorCodeFileSystemOperation, "file operation failed", cause)

	unwrapped := err.Unwrap()
	suite.Equal(cause, unwrapped)
}

func (suite *ErrorsTestSuite) TestLnkErrorIs() {
	err1 := NewLnkError(ErrorCodeFileNotFound, "file not found")
	err2 := NewLnkError(ErrorCodeFileNotFound, "another file not found")
	err3 := NewLnkError(ErrorCodeFileAlreadyManaged, "file already managed")

	// Same error code should match
	suite.True(errors.Is(err1, err2), "expected errors with same code to match")

	// Different error codes should not match
	suite.False(errors.Is(err1, err3), "expected errors with different codes to not match")

	// Test with wrapped errors
	cause := errors.New("io error")
	wrappedErr := WrapError(ErrorCodeFileSystemOperation, "wrapped", cause)
	suite.True(errors.Is(wrappedErr, cause), "expected wrapped error to match its cause")
}

func (suite *ErrorsTestSuite) TestLnkErrorWithContext() {
	err := NewLnkError(ErrorCodeFileNotFound, "file not found")
	err = err.WithContext("path", "/test/file.txt")
	err = err.WithContext("operation", "read")

	suite.Equal("/test/file.txt", err.Context["path"])
	suite.Equal("read", err.Context["operation"])
}

func (suite *ErrorsTestSuite) TestNewFileNotFoundError() {
	path := "/test/file.txt"
	err := NewFileNotFoundError(path)

	suite.Equal(ErrorCodeFileNotFound, err.Code)
	suite.Equal(path, err.Context["path"])
}

func (suite *ErrorsTestSuite) TestNewFileAlreadyManagedError() {
	path := "/test/file.txt"
	err := NewFileAlreadyManagedError(path)

	suite.Equal(ErrorCodeFileAlreadyManaged, err.Code)
	suite.Equal(path, err.Context["path"])
}

func (suite *ErrorsTestSuite) TestNewRepoNotInitializedError() {
	repoPath := "/test/repo"
	err := NewRepoNotInitializedError(repoPath)

	suite.Equal(ErrorCodeRepoNotInitialized, err.Code)
	suite.Equal(repoPath, err.Context["repo_path"])
}

func (suite *ErrorsTestSuite) TestNewGitOperationError() {
	operation := "push"
	cause := errors.New("network error")
	err := NewGitOperationError(operation, cause)

	suite.Equal(ErrorCodeGitOperation, err.Code)
	suite.Equal(cause, err.Cause)
	suite.Equal(operation, err.Context["operation"])
}

func TestErrorsSuite(t *testing.T) {
	suite.Run(t, new(ErrorsTestSuite))
}
