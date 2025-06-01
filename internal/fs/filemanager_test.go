package fs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/yarlson/lnk/internal/errors"
)

type FileManagerTestSuite struct {
	suite.Suite
	tempDir     string
	fileManager *FileManager
	ctx         context.Context
}

func (suite *FileManagerTestSuite) SetupTest() {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "lnk_test_*")
	suite.Require().NoError(err)
	suite.tempDir = tempDir

	// Create file manager
	suite.fileManager = New()

	// Create context
	suite.ctx = context.Background()
}

func (suite *FileManagerTestSuite) TearDownTest() {
	err := os.RemoveAll(suite.tempDir)
	suite.Require().NoError(err)
}

func (suite *FileManagerTestSuite) TestExists() {
	// Test existing file
	testFile := filepath.Join(suite.tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	suite.Require().NoError(err)

	exists, err := suite.fileManager.Exists(suite.ctx, testFile)
	suite.NoError(err)
	suite.True(exists)

	// Test non-existing file
	nonExistentFile := filepath.Join(suite.tempDir, "nonexistent.txt")
	exists, err = suite.fileManager.Exists(suite.ctx, nonExistentFile)
	suite.NoError(err)
	suite.False(exists)
}

func (suite *FileManagerTestSuite) TestExistsWithCancellation() {
	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := suite.fileManager.Exists(ctx, "/any/path")
	suite.Equal(context.Canceled, err)
}

func (suite *FileManagerTestSuite) TestIsDirectory() {
	// Test directory
	isDir, err := suite.fileManager.IsDirectory(suite.ctx, suite.tempDir)
	suite.NoError(err)
	suite.True(isDir)

	// Test file
	testFile := filepath.Join(suite.tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	suite.Require().NoError(err)

	isDir, err = suite.fileManager.IsDirectory(suite.ctx, testFile)
	suite.NoError(err)
	suite.False(isDir)

	// Test non-existing file
	nonExistentFile := filepath.Join(suite.tempDir, "nonexistent.txt")
	_, err = suite.fileManager.IsDirectory(suite.ctx, nonExistentFile)
	suite.Error(err)

	// Check that it's a models error
	suite.True(errors.NewFileNotFoundError("").Is(err))
}

func (suite *FileManagerTestSuite) TestMove() {
	// Create test file
	srcFile := filepath.Join(suite.tempDir, "source.txt")
	testContent := []byte("test content")
	err := os.WriteFile(srcFile, testContent, 0644)
	suite.Require().NoError(err)

	// Test moving file
	dstFile := filepath.Join(suite.tempDir, "subdir", "destination.txt")
	err = suite.fileManager.Move(suite.ctx, srcFile, dstFile)
	suite.NoError(err)

	// Verify source doesn't exist
	_, err = os.Stat(srcFile)
	suite.True(os.IsNotExist(err))

	// Verify destination exists with correct content
	content, err := os.ReadFile(dstFile)
	suite.NoError(err)
	suite.Equal(string(testContent), string(content))
}

func (suite *FileManagerTestSuite) TestCreateSymlink() {
	// Create target file
	targetFile := filepath.Join(suite.tempDir, "target.txt")
	err := os.WriteFile(targetFile, []byte("test"), 0644)
	suite.Require().NoError(err)

	// Create symlink
	linkFile := filepath.Join(suite.tempDir, "link.txt")
	err = suite.fileManager.CreateSymlink(suite.ctx, targetFile, linkFile)
	suite.NoError(err)

	// Verify symlink exists and points to target
	info, err := os.Lstat(linkFile)
	suite.NoError(err)
	suite.NotZero(info.Mode() & os.ModeSymlink)

	// Verify symlink target
	target, err := os.Readlink(linkFile)
	suite.NoError(err)

	expectedTarget := "target.txt" // Should be relative
	suite.Equal(expectedTarget, target)
}

func (suite *FileManagerTestSuite) TestReadWriteFile() {
	// Test writing file
	testFile := filepath.Join(suite.tempDir, "subdir", "test.txt")
	testContent := []byte("test content")
	err := suite.fileManager.WriteFile(suite.ctx, testFile, testContent, 0644)
	suite.NoError(err)

	// Test reading file
	content, err := suite.fileManager.ReadFile(suite.ctx, testFile)
	suite.NoError(err)
	suite.Equal(string(testContent), string(content))

	// Test reading non-existent file
	nonExistentFile := filepath.Join(suite.tempDir, "nonexistent.txt")
	_, err = suite.fileManager.ReadFile(suite.ctx, nonExistentFile)
	suite.Error(err)
	suite.True(errors.NewFileNotFoundError("").Is(err))
}

func (suite *FileManagerTestSuite) TestRemove() {
	// Create test file
	testFile := filepath.Join(suite.tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	suite.Require().NoError(err)

	// Remove file
	err = suite.fileManager.Remove(suite.ctx, testFile)
	suite.NoError(err)

	// Verify file doesn't exist
	_, err = os.Stat(testFile)
	suite.True(os.IsNotExist(err))

	// Test removing non-existent file (should not error)
	err = suite.fileManager.Remove(suite.ctx, testFile)
	suite.NoError(err)
}

func (suite *FileManagerTestSuite) TestMkdirAll() {
	// Create nested directory
	nestedDir := filepath.Join(suite.tempDir, "a", "b", "c")
	err := suite.fileManager.MkdirAll(suite.ctx, nestedDir, 0755)
	suite.NoError(err)

	// Verify directory exists
	info, err := os.Stat(nestedDir)
	suite.NoError(err)
	suite.True(info.IsDir())
}

func (suite *FileManagerTestSuite) TestReadlink() {
	// Create target file
	targetFile := filepath.Join(suite.tempDir, "target.txt")
	err := os.WriteFile(targetFile, []byte("test"), 0644)
	suite.Require().NoError(err)

	// Create symlink
	linkFile := filepath.Join(suite.tempDir, "link.txt")
	err = os.Symlink("target.txt", linkFile)
	suite.Require().NoError(err)

	// Test reading symlink
	target, err := suite.fileManager.Readlink(suite.ctx, linkFile)
	suite.NoError(err)
	suite.Equal("target.txt", target)

	// Test reading non-symlink
	_, err = suite.fileManager.Readlink(suite.ctx, targetFile)
	suite.Error(err)
}

func (suite *FileManagerTestSuite) TestStatAndLstat() {
	// Create target file
	targetFile := filepath.Join(suite.tempDir, "target.txt")
	err := os.WriteFile(targetFile, []byte("test"), 0644)
	suite.Require().NoError(err)

	// Create symlink
	linkFile := filepath.Join(suite.tempDir, "link.txt")
	err = os.Symlink("target.txt", linkFile)
	suite.Require().NoError(err)

	// Test Stat on regular file
	info, err := suite.fileManager.Stat(suite.ctx, targetFile)
	suite.NoError(err)
	suite.False(info.IsDir())

	// Test Stat on symlink (should follow link)
	info, err = suite.fileManager.Stat(suite.ctx, linkFile)
	suite.NoError(err)
	suite.False(info.IsDir())

	// Test Lstat on symlink (should not follow link)
	info, err = suite.fileManager.Lstat(suite.ctx, linkFile)
	suite.NoError(err)
	suite.NotZero(info.Mode() & os.ModeSymlink)

	// Test on non-existent file
	nonExistentFile := filepath.Join(suite.tempDir, "nonexistent.txt")
	_, err = suite.fileManager.Stat(suite.ctx, nonExistentFile)
	suite.Error(err)
	suite.True(errors.NewFileNotFoundError("").Is(err))
}

func (suite *FileManagerTestSuite) TestContextCancellation() {
	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Allow time for context to expire
	time.Sleep(1 * time.Millisecond)

	// Test various operations with cancelled context
	_, err := suite.fileManager.Exists(ctx, "/any/path")
	suite.Equal(context.DeadlineExceeded, err)

	_, err = suite.fileManager.IsDirectory(ctx, "/any/path")
	suite.Equal(context.DeadlineExceeded, err)

	err = suite.fileManager.Move(ctx, "/src", "/dst")
	suite.Equal(context.DeadlineExceeded, err)
}

func TestFileManagerSuite(t *testing.T) {
	suite.Run(t, new(FileManagerTestSuite))
}
