package lnk

import (
	"fmt"
	"os"
	"path/filepath"
)

// Test core add functionality with files
func (suite *CoreTestSuite) TestCoreFileOperations() {
	// Initialize first
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create a test file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	content := "export PATH=$PATH:/usr/local/bin"
	err = os.WriteFile(testFile, []byte(content), 0644)
	suite.Require().NoError(err)

	// Add the file
	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Verify symlink and repo file
	info, err := os.Lstat(testFile)
	suite.Require().NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)

	// The repository file will preserve the directory structure
	lnkDir := filepath.Join(suite.tempDir, "lnk")

	// Find the .bashrc file in the repository (it should be at the relative path from HOME)
	repoFile := filepath.Join(lnkDir, ".bashrc")
	suite.FileExists(repoFile)

	// Verify content is preserved
	repoContent, err := os.ReadFile(repoFile)
	suite.Require().NoError(err)
	suite.Equal(content, string(repoContent))

	// Test remove
	err = suite.lnk.Remove(testFile)
	suite.Require().NoError(err)

	// Verify symlink is gone and regular file is restored
	info, err = os.Lstat(testFile)
	suite.Require().NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink) // Not a symlink

	// Verify content is preserved
	restoredContent, err := os.ReadFile(testFile)
	suite.Require().NoError(err)
	suite.Equal(content, string(restoredContent))
}

// Test core add/remove functionality with directories
func (suite *CoreTestSuite) TestCoreDirectoryOperations() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create a directory with files
	testDir := filepath.Join(suite.tempDir, "testdir")
	err = os.MkdirAll(testDir, 0755)
	suite.Require().NoError(err)

	testFile := filepath.Join(testDir, "config.txt")
	content := "test config"
	err = os.WriteFile(testFile, []byte(content), 0644)
	suite.Require().NoError(err)

	// Add the directory
	err = suite.lnk.Add(testDir)
	suite.Require().NoError(err)

	// Verify directory is now a symlink
	info, err := os.Lstat(testDir)
	suite.Require().NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)

	// Check that the repository directory preserves the structure
	lnkDir := filepath.Join(suite.tempDir, "lnk")

	// The directory should be at the relative path from HOME
	repoDir := filepath.Join(lnkDir, "testdir")
	suite.DirExists(repoDir)

	// Remove the directory
	err = suite.lnk.Remove(testDir)
	suite.Require().NoError(err)

	// Verify symlink is gone and regular directory is restored
	info, err = os.Lstat(testDir)
	suite.Require().NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink) // Not a symlink
	suite.True(info.IsDir())                                // Is a directory

	// Verify content is preserved
	restoredContent, err := os.ReadFile(testFile)
	suite.Require().NoError(err)
	suite.Equal(content, string(restoredContent))
}

// Test edge case: files with same basename from different directories should be handled properly
func (suite *CoreTestSuite) TestSameBasenameFilesOverwrite() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create two directories with files having the same basename
	dirA := filepath.Join(suite.tempDir, "a")
	dirB := filepath.Join(suite.tempDir, "b")
	err = os.MkdirAll(dirA, 0755)
	suite.Require().NoError(err)
	err = os.MkdirAll(dirB, 0755)
	suite.Require().NoError(err)

	// Create files with same basename but different content
	fileA := filepath.Join(dirA, "config.json")
	fileB := filepath.Join(dirB, "config.json")
	contentA := `{"name": "config_a"}`
	contentB := `{"name": "config_b"}`

	err = os.WriteFile(fileA, []byte(contentA), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(fileB, []byte(contentB), 0644)
	suite.Require().NoError(err)

	// Add first file
	err = suite.lnk.Add(fileA)
	suite.Require().NoError(err)

	// Verify first file is managed correctly and preserves content
	info, err := os.Lstat(fileA)
	suite.Require().NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)

	symlinkContentA, err := os.ReadFile(fileA)
	suite.Require().NoError(err)
	suite.Equal(contentA, string(symlinkContentA), "First file should preserve its original content")

	// Add second file - this should work without overwriting the first
	err = suite.lnk.Add(fileB)
	suite.Require().NoError(err)

	// Verify second file is managed
	info, err = os.Lstat(fileB)
	suite.Require().NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)

	// CORRECT BEHAVIOR: Both files should preserve their original content
	symlinkContentA, err = os.ReadFile(fileA)
	suite.Require().NoError(err)
	symlinkContentB, err := os.ReadFile(fileB)
	suite.Require().NoError(err)

	suite.Equal(contentA, string(symlinkContentA), "First file should keep its original content")
	suite.Equal(contentB, string(symlinkContentB), "Second file should keep its original content")

	// Both files should be removable independently
	err = suite.lnk.Remove(fileA)
	suite.Require().NoError(err, "First file should be removable")

	// First file should be restored with correct content
	info, err = os.Lstat(fileA)
	suite.Require().NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink) // Not a symlink anymore

	restoredContentA, err := os.ReadFile(fileA)
	suite.Require().NoError(err)
	suite.Equal(contentA, string(restoredContentA), "Restored file should have original content")

	// Second file should still be manageable and removable
	err = suite.lnk.Remove(fileB)
	suite.Require().NoError(err, "Second file should also be removable without errors")

	// Second file should be restored with correct content
	info, err = os.Lstat(fileB)
	suite.Require().NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink) // Not a symlink anymore

	restoredContentB, err := os.ReadFile(fileB)
	suite.Require().NoError(err)
	suite.Equal(contentB, string(restoredContentB), "Second restored file should have original content")
}

// Test another variant: adding files with same basename should work correctly
func (suite *CoreTestSuite) TestSameBasenameSequentialAdd() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create subdirectories in different locations
	configDir := filepath.Join(suite.tempDir, "config")
	backupDir := filepath.Join(suite.tempDir, "backup")
	err = os.MkdirAll(configDir, 0755)
	suite.Require().NoError(err)
	err = os.MkdirAll(backupDir, 0755)
	suite.Require().NoError(err)

	// Create files with same basename (.bashrc)
	configBashrc := filepath.Join(configDir, ".bashrc")
	backupBashrc := filepath.Join(backupDir, ".bashrc")

	originalContent := "export PATH=/usr/local/bin:$PATH"
	backupContent := "export PATH=/opt/bin:$PATH"

	err = os.WriteFile(configBashrc, []byte(originalContent), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(backupBashrc, []byte(backupContent), 0644)
	suite.Require().NoError(err)

	// Add first .bashrc
	err = suite.lnk.Add(configBashrc)
	suite.Require().NoError(err)

	// Add second .bashrc - should work without overwriting the first
	err = suite.lnk.Add(backupBashrc)
	suite.Require().NoError(err)

	// Check .lnk tracking file should track both properly
	lnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk")
	lnkContent, err := os.ReadFile(lnkFile)
	suite.Require().NoError(err)

	// Both entries should be tracked and distinguishable
	content := string(lnkContent)
	suite.Contains(content, ".bashrc", "Both .bashrc files should be tracked")

	// Both files should maintain their distinct content
	content1, err := os.ReadFile(configBashrc)
	suite.Require().NoError(err)
	content2, err := os.ReadFile(backupBashrc)
	suite.Require().NoError(err)

	suite.Equal(originalContent, string(content1), "First file should keep original content")
	suite.Equal(backupContent, string(content2), "Second file should keep its distinct content")

	// Both should be removable independently
	err = suite.lnk.Remove(configBashrc)
	suite.Require().NoError(err, "First .bashrc should be removable")

	err = suite.lnk.Remove(backupBashrc)
	suite.Require().NoError(err, "Second .bashrc should be removable")
}

// TestAdd tests Add() function with various error paths
func (suite *CoreTestSuite) TestAdd() {
	tests := []struct {
		name        string
		setupFunc   func() (string, error)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful add of regular file",
			setupFunc: func() (string, error) {
				err := suite.lnk.Init()
				if err != nil {
					return "", err
				}
				testFile := filepath.Join(suite.tempDir, ".testrc")
				err = os.WriteFile(testFile, []byte("test content"), 0644)
				return testFile, err
			},
			wantErr: false,
		},
		{
			name: "add nonexistent file",
			setupFunc: func() (string, error) {
				err := suite.lnk.Init()
				if err != nil {
					return "", err
				}
				return filepath.Join(suite.tempDir, "nonexistent"), nil
			},
			wantErr:     true,
			errContains: "File or directory not found",
		},
		{
			name: "add file already managed",
			setupFunc: func() (string, error) {
				err := suite.lnk.Init()
				if err != nil {
					return "", err
				}
				testFile := filepath.Join(suite.tempDir, ".bashrc")
				err = os.WriteFile(testFile, []byte("content"), 0644)
				if err != nil {
					return "", err
				}
				// Add it once
				err = suite.lnk.Add(testFile)
				return testFile, err
			},
			wantErr:     true,
			errContains: "already managed",
		},
		{
			name: "add directory successfully",
			setupFunc: func() (string, error) {
				err := suite.lnk.Init()
				if err != nil {
					return "", err
				}
				testDir := filepath.Join(suite.tempDir, ".config", "myapp")
				err = os.MkdirAll(testDir, 0755)
				if err != nil {
					return "", err
				}
				// Create a file inside to make it non-empty
				testFile := filepath.Join(testDir, "config.txt")
				err = os.WriteFile(testFile, []byte("config"), 0644)
				return testDir, err
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			filePath, err := tt.setupFunc()
			suite.Require().NoError(err, "Setup failed for test: %s", tt.name)

			// Execute Add (only if setup succeeded and file wasn't already added)
			if !tt.wantErr || tt.errContains != "already managed" {
				err = suite.lnk.Add(filePath)
			} else {
				// For "already managed" test, try to add again
				err = suite.lnk.Add(filePath)
			}

			// Verify
			if tt.wantErr {
				suite.Error(err, "Expected error for test: %s", tt.name)
				if tt.errContains != "" {
					suite.Contains(err.Error(), tt.errContains, "Error message mismatch for: %s", tt.name)
				}
			} else {
				suite.NoError(err, "Unexpected error for test: %s", tt.name)

				// Verify file is now a symlink
				info, err := os.Lstat(filePath)
				suite.NoError(err)
				suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink, "File should be a symlink")
			}
		})
	}
}

func (suite *CoreTestSuite) TestAddMultiple() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create multiple test files
	file1 := filepath.Join(suite.tempDir, "file1.txt")
	file2 := filepath.Join(suite.tempDir, "file2.txt")
	file3 := filepath.Join(suite.tempDir, "file3.txt")

	content1 := "content1"
	content2 := "content2"
	content3 := "content3"

	err = os.WriteFile(file1, []byte(content1), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(file2, []byte(content2), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(file3, []byte(content3), 0644)
	suite.Require().NoError(err)

	// Test AddMultiple method - should succeed
	paths := []string{file1, file2, file3}
	err = suite.lnk.AddMultiple(paths)
	suite.NoError(err, "AddMultiple should succeed")

	// Verify all files are now symlinks
	for _, file := range paths {
		info, err := os.Lstat(file)
		suite.NoError(err)
		suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink, "File should be a symlink: %s", file)
	}

	// Verify all files exist in storage
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	suite.FileExists(filepath.Join(lnkDir, "file1.txt"))
	suite.FileExists(filepath.Join(lnkDir, "file2.txt"))
	suite.FileExists(filepath.Join(lnkDir, "file3.txt"))

	// Verify .lnk file contains all entries
	lnkFile := filepath.Join(lnkDir, ".lnk")
	lnkContent, err := os.ReadFile(lnkFile)
	suite.NoError(err)
	suite.Equal("file1.txt\nfile2.txt\nfile3.txt\n", string(lnkContent))

	// Verify Git commit was created
	commits, err := suite.lnk.GetCommits()
	suite.NoError(err)
	suite.T().Logf("Commits: %v", commits)
	// Should have at least 1 commit for the batch add
	suite.GreaterOrEqual(len(commits), 1)
	// The most recent commit should mention multiple files
	suite.Contains(commits[0], "added 3 files")
}

func (suite *CoreTestSuite) TestAddMultipleWithConflicts() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create test files
	file1 := filepath.Join(suite.tempDir, "file1.txt")
	file2 := filepath.Join(suite.tempDir, "file2.txt")
	file3 := filepath.Join(suite.tempDir, "file3.txt")

	err = os.WriteFile(file1, []byte("content1"), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(file2, []byte("content2"), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(file3, []byte("content3"), 0644)
	suite.Require().NoError(err)

	// Add file2 individually first
	err = suite.lnk.Add(file2)
	suite.Require().NoError(err)

	// Now try to add all three - should fail due to conflict with file2
	paths := []string{file1, file2, file3}
	err = suite.lnk.AddMultiple(paths)
	suite.Error(err, "AddMultiple should fail due to conflict")
	suite.Contains(err.Error(), "already managed")

	// Verify no partial changes were made
	// file1 and file3 should still be regular files, not symlinks
	info1, err := os.Lstat(file1)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info1.Mode()&os.ModeSymlink, "file1 should not be a symlink")

	info3, err := os.Lstat(file3)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info3.Mode()&os.ModeSymlink, "file3 should not be a symlink")

	// file2 should still be managed (was added before)
	info2, err := os.Lstat(file2)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info2.Mode()&os.ModeSymlink, "file2 should remain a symlink")
}

func (suite *CoreTestSuite) TestAddMultipleRollback() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create test files - one will be invalid to force rollback
	file1 := filepath.Join(suite.tempDir, "file1.txt")
	file2 := filepath.Join(suite.tempDir, "nonexistent.txt") // This doesn't exist
	file3 := filepath.Join(suite.tempDir, "file3.txt")

	err = os.WriteFile(file1, []byte("content1"), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(file3, []byte("content3"), 0644)
	suite.Require().NoError(err)
	// Note: file2 is intentionally not created

	// Try to add all files - should fail and rollback
	paths := []string{file1, file2, file3}
	err = suite.lnk.AddMultiple(paths)
	suite.Error(err, "AddMultiple should fail due to nonexistent file")

	// Verify rollback - no files should be symlinks
	info1, err := os.Lstat(file1)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info1.Mode()&os.ModeSymlink, "file1 should not be a symlink after rollback")

	info3, err := os.Lstat(file3)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info3.Mode()&os.ModeSymlink, "file3 should not be a symlink after rollback")

	// Verify no files in storage
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	suite.NoFileExists(filepath.Join(lnkDir, "file1.txt"))
	suite.NoFileExists(filepath.Join(lnkDir, "file3.txt"))

	// Verify .lnk file is empty or doesn't contain these files
	lnkFile := filepath.Join(lnkDir, ".lnk")
	if _, err := os.Stat(lnkFile); err == nil {
		lnkContent, err := os.ReadFile(lnkFile)
		suite.NoError(err)
		content := string(lnkContent)
		suite.NotContains(content, "file1.txt")
		suite.NotContains(content, "file3.txt")
	}
}

func (suite *CoreTestSuite) TestValidateMultiplePaths() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create a mix of valid and invalid paths
	validFile := filepath.Join(suite.tempDir, "valid.txt")
	err = os.WriteFile(validFile, []byte("content"), 0644)
	suite.Require().NoError(err)

	nonexistentFile := filepath.Join(suite.tempDir, "nonexistent.txt")
	// Don't create this file

	// Create a valid directory
	validDir := filepath.Join(suite.tempDir, "validdir")
	err = os.MkdirAll(validDir, 0755)
	suite.Require().NoError(err)

	// Test validation fails early with detailed error
	paths := []string{validFile, nonexistentFile, validDir}
	err = suite.lnk.AddMultiple(paths)
	suite.Error(err, "Should fail due to nonexistent file")
	suite.Contains(err.Error(), "validation failed")
	suite.Contains(err.Error(), "nonexistent.txt")

	// Verify no partial changes were made
	info, err := os.Lstat(validFile)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink, "valid file should not be a symlink")

	info, err = os.Lstat(validDir)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink, "valid directory should not be a symlink")
}

func (suite *CoreTestSuite) TestAtomicRollbackOnFailure() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create test files
	file1 := filepath.Join(suite.tempDir, "file1.txt")
	file2 := filepath.Join(suite.tempDir, "file2.txt")
	file3 := filepath.Join(suite.tempDir, "file3.txt")

	content1 := "original content 1"
	content2 := "original content 2"
	content3 := "original content 3"

	err = os.WriteFile(file1, []byte(content1), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(file2, []byte(content2), 0644)
	suite.Require().NoError(err)
	err = os.WriteFile(file3, []byte(content3), 0644)
	suite.Require().NoError(err)

	// Add file2 individually first to create a conflict
	err = suite.lnk.Add(file2)
	suite.Require().NoError(err)

	// Store original states
	info1Before, err := os.Lstat(file1)
	suite.Require().NoError(err)
	info3Before, err := os.Lstat(file3)
	suite.Require().NoError(err)

	// Try to add all files - should fail and rollback completely
	paths := []string{file1, file2, file3}
	err = suite.lnk.AddMultiple(paths)
	suite.Error(err, "Should fail due to conflict with file2")

	// Verify complete rollback
	info1After, err := os.Lstat(file1)
	suite.NoError(err)
	suite.Equal(info1Before.Mode(), info1After.Mode(), "file1 mode should be unchanged")

	info3After, err := os.Lstat(file3)
	suite.NoError(err)
	suite.Equal(info3Before.Mode(), info3After.Mode(), "file3 mode should be unchanged")

	// Verify original contents are preserved
	content1After, err := os.ReadFile(file1)
	suite.NoError(err)
	suite.Equal(content1, string(content1After), "file1 content should be preserved")

	content3After, err := os.ReadFile(file3)
	suite.NoError(err)
	suite.Equal(content3, string(content3After), "file3 content should be preserved")

	// file2 should still be managed (was added before)
	info2, err := os.Lstat(file2)
	suite.NoError(err)
	suite.Equal(os.ModeSymlink, info2.Mode()&os.ModeSymlink, "file2 should remain a symlink")
}

func (suite *CoreTestSuite) TestDetailedErrorMessages() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Test with multiple types of errors
	validFile := filepath.Join(suite.tempDir, "valid.txt")
	err = os.WriteFile(validFile, []byte("content"), 0644)
	suite.Require().NoError(err)

	nonexistentFile := filepath.Join(suite.tempDir, "does-not-exist.txt")
	alreadyManagedFile := filepath.Join(suite.tempDir, "already-managed.txt")
	err = os.WriteFile(alreadyManagedFile, []byte("managed"), 0644)
	suite.Require().NoError(err)

	// Add one file first to create conflict
	err = suite.lnk.Add(alreadyManagedFile)
	suite.Require().NoError(err)

	// Test with nonexistent file
	paths := []string{validFile, nonexistentFile}
	err = suite.lnk.AddMultiple(paths)
	suite.Error(err, "Should fail due to nonexistent file")
	suite.Contains(err.Error(), "validation failed", "Error should mention validation failure")
	suite.Contains(err.Error(), "does-not-exist.txt", "Error should include specific filename")

	// Test with already managed file
	paths = []string{validFile, alreadyManagedFile}
	err = suite.lnk.AddMultiple(paths)
	suite.Error(err, "Should fail due to already managed file")
	suite.Contains(err.Error(), "already managed", "Error should mention already managed")
	suite.Contains(err.Error(), "already-managed.txt", "Error should include specific filename")
}

// Task 2.2: Directory Walking Logic Tests

func (suite *CoreTestSuite) TestWalkDirectory() {
	// Initialize lnk repository
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create nested directory structure
	configDir := filepath.Join(suite.tempDir, ".config", "myapp")
	err = os.MkdirAll(configDir, 0755)
	suite.Require().NoError(err)

	themeDir := filepath.Join(configDir, "themes")
	err = os.MkdirAll(themeDir, 0755)
	suite.Require().NoError(err)

	// Create files in different levels
	file1 := filepath.Join(configDir, "config.json")
	file2 := filepath.Join(configDir, "settings.json")
	file3 := filepath.Join(themeDir, "dark.json")
	file4 := filepath.Join(themeDir, "light.json")

	suite.Require().NoError(os.WriteFile(file1, []byte("config"), 0644))
	suite.Require().NoError(os.WriteFile(file2, []byte("settings"), 0644))
	suite.Require().NoError(os.WriteFile(file3, []byte("dark theme"), 0644))
	suite.Require().NoError(os.WriteFile(file4, []byte("light theme"), 0644))

	// Call walkDirectory method
	files, err := suite.lnk.files.WalkDirectory(configDir)
	suite.Require().NoError(err, "walkDirectory should succeed")

	// Should find all 4 files
	suite.Len(files, 4, "Should find all files in nested structure")

	// Check that all expected files are found (order may vary)
	expectedFiles := []string{file1, file2, file3, file4}
	for _, expectedFile := range expectedFiles {
		suite.Contains(files, expectedFile, "Should include file %s", expectedFile)
	}
}

func (suite *CoreTestSuite) TestWalkDirectoryIncludesHiddenFiles() {
	// Initialize lnk repository
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create directory with hidden files and directories
	testDir := filepath.Join(suite.tempDir, "test-hidden")
	err = os.MkdirAll(testDir, 0755)
	suite.Require().NoError(err)

	hiddenDir := filepath.Join(testDir, ".hidden")
	err = os.MkdirAll(hiddenDir, 0755)
	suite.Require().NoError(err)

	// Create regular and hidden files
	regularFile := filepath.Join(testDir, "regular.txt")
	hiddenFile := filepath.Join(testDir, ".hidden-file")
	hiddenDirFile := filepath.Join(hiddenDir, "file-in-hidden.txt")

	suite.Require().NoError(os.WriteFile(regularFile, []byte("regular"), 0644))
	suite.Require().NoError(os.WriteFile(hiddenFile, []byte("hidden"), 0644))
	suite.Require().NoError(os.WriteFile(hiddenDirFile, []byte("in hidden dir"), 0644))

	// Call walkDirectory method
	files, err := suite.lnk.files.WalkDirectory(testDir)
	suite.Require().NoError(err, "walkDirectory should succeed with hidden files")

	// Should find all files including hidden ones
	suite.Len(files, 3, "Should find all files including hidden ones")
	suite.Contains(files, regularFile, "Should include regular file")
	suite.Contains(files, hiddenFile, "Should include hidden file")
	suite.Contains(files, hiddenDirFile, "Should include file in hidden directory")
}

func (suite *CoreTestSuite) TestWalkDirectorySymlinkHandling() {
	// Initialize lnk repository
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create directory structure
	testDir := filepath.Join(suite.tempDir, "test-symlinks")
	err = os.MkdirAll(testDir, 0755)
	suite.Require().NoError(err)

	// Create a regular file
	regularFile := filepath.Join(testDir, "regular.txt")
	suite.Require().NoError(os.WriteFile(regularFile, []byte("regular"), 0644))

	// Create a symlink to the regular file
	symlinkFile := filepath.Join(testDir, "link-to-regular.txt")
	err = os.Symlink(regularFile, symlinkFile)
	suite.Require().NoError(err)

	// Call walkDirectory method
	files, err := suite.lnk.files.WalkDirectory(testDir)
	suite.Require().NoError(err, "walkDirectory should handle symlinks")

	// Should include both regular file and properly handle symlink
	// (exact behavior depends on implementation - could include symlink as file)
	suite.GreaterOrEqual(len(files), 1, "Should find at least the regular file")
	suite.Contains(files, regularFile, "Should include regular file")

	// The symlink handling behavior will be defined in implementation
	// For now, we just ensure no errors occur
}

func (suite *CoreTestSuite) TestWalkDirectoryEmptyDirs() {
	// Initialize lnk repository
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create directory structure with empty directories
	testDir := filepath.Join(suite.tempDir, "test-empty")
	err = os.MkdirAll(testDir, 0755)
	suite.Require().NoError(err)

	// Create empty subdirectories
	emptyDir1 := filepath.Join(testDir, "empty1")
	emptyDir2 := filepath.Join(testDir, "empty2")
	err = os.MkdirAll(emptyDir1, 0755)
	suite.Require().NoError(err)
	err = os.MkdirAll(emptyDir2, 0755)
	suite.Require().NoError(err)

	// Create one file in a non-empty directory
	nonEmptyDir := filepath.Join(testDir, "non-empty")
	err = os.MkdirAll(nonEmptyDir, 0755)
	suite.Require().NoError(err)

	testFile := filepath.Join(nonEmptyDir, "test.txt")
	suite.Require().NoError(os.WriteFile(testFile, []byte("content"), 0644))

	// Call walkDirectory method
	files, err := suite.lnk.files.WalkDirectory(testDir)
	suite.Require().NoError(err, "walkDirectory should skip empty directories")

	// Should only find the one file, not empty directories
	suite.Len(files, 1, "Should only find files, not empty directories")
	suite.Contains(files, testFile, "Should include the actual file")
}

// Task 2.3: Progress Indication System Tests

func (suite *CoreTestSuite) TestProgressReporting() {
	// Initialize lnk repository
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create directory with multiple files to test progress reporting
	testDir := filepath.Join(suite.tempDir, "progress-test")
	err = os.MkdirAll(testDir, 0755)
	suite.Require().NoError(err)

	// Create 15 files to exceed threshold
	expectedFiles := 15
	for i := 0; i < expectedFiles; i++ {
		file := filepath.Join(testDir, fmt.Sprintf("file%d.txt", i))
		suite.Require().NoError(os.WriteFile(file, []byte(fmt.Sprintf("content %d", i)), 0644))
	}

	// Track progress calls
	var progressCalls []struct {
		Current     int
		Total       int
		CurrentFile string
	}

	progressCallback := func(current, total int, currentFile string) {
		progressCalls = append(progressCalls, struct {
			Current     int
			Total       int
			CurrentFile string
		}{
			Current:     current,
			Total:       total,
			CurrentFile: currentFile,
		})
	}

	// Call AddRecursiveWithProgress method
	err = suite.lnk.AddRecursiveWithProgress([]string{testDir}, progressCallback)
	suite.Require().NoError(err, "AddRecursiveWithProgress should succeed")

	// Verify progress was reported
	suite.Greater(len(progressCalls), 0, "Progress callback should be called")
	suite.Equal(expectedFiles, len(progressCalls), "Should have progress calls for each file")

	// Verify progress order and totals
	for i, call := range progressCalls {
		suite.Equal(i+1, call.Current, "Current count should increment")
		suite.Equal(expectedFiles, call.Total, "Total should be consistent")
		suite.NotEmpty(call.CurrentFile, "CurrentFile should be provided")
	}
}

func (suite *CoreTestSuite) TestProgressThreshold() {
	// Initialize lnk repository
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Test with few files (under threshold)
	smallDir := filepath.Join(suite.tempDir, "small-test")
	err = os.MkdirAll(smallDir, 0755)
	suite.Require().NoError(err)

	// Create only 5 files (under 10 threshold)
	for i := 0; i < 5; i++ {
		file := filepath.Join(smallDir, fmt.Sprintf("small%d.txt", i))
		suite.Require().NoError(os.WriteFile(file, []byte(fmt.Sprintf("content %d", i)), 0644))
	}

	// Track progress calls for small operation
	smallProgressCalls := 0
	smallCallback := func(current, total int, currentFile string) {
		smallProgressCalls++
	}

	err = suite.lnk.AddRecursiveWithProgress([]string{smallDir}, smallCallback)
	suite.Require().NoError(err, "AddRecursiveWithProgress should succeed for small operation")

	// Should NOT call progress for small operations
	suite.Equal(0, smallProgressCalls, "Progress should not be called for operations under threshold")

	// Test with many files (over threshold)
	largeDir := filepath.Join(suite.tempDir, "large-test")
	err = os.MkdirAll(largeDir, 0755)
	suite.Require().NoError(err)

	// Create 15 files (over 10 threshold)
	for i := 0; i < 15; i++ {
		file := filepath.Join(largeDir, fmt.Sprintf("large%d.txt", i))
		suite.Require().NoError(os.WriteFile(file, []byte(fmt.Sprintf("content %d", i)), 0644))
	}

	// Track progress calls for large operation
	largeProgressCalls := 0
	largeCallback := func(current, total int, currentFile string) {
		largeProgressCalls++
	}

	err = suite.lnk.AddRecursiveWithProgress([]string{largeDir}, largeCallback)
	suite.Require().NoError(err, "AddRecursiveWithProgress should succeed for large operation")

	// Should call progress for large operations
	suite.Equal(15, largeProgressCalls, "Progress should be called for operations over threshold")
}

// Task 3.1: Dry-Run Mode Core Tests

func (suite *CoreTestSuite) TestPreviewAdd() {
	// Initialize lnk repository
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create test files
	testFile1 := filepath.Join(suite.tempDir, "test1.txt")
	testFile2 := filepath.Join(suite.tempDir, "test2.txt")
	suite.Require().NoError(os.WriteFile(testFile1, []byte("content1"), 0644))
	suite.Require().NoError(os.WriteFile(testFile2, []byte("content2"), 0644))

	// Test PreviewAdd for multiple files
	files, err := suite.lnk.PreviewAdd([]string{testFile1, testFile2}, false)
	suite.Require().NoError(err, "PreviewAdd should succeed")

	// Should return both files
	suite.Len(files, 2, "Should preview both files")
	suite.Contains(files, testFile1, "Should include first file")
	suite.Contains(files, testFile2, "Should include second file")

	// Verify no actual changes were made (files should still be regular files)
	info, err := os.Lstat(testFile1)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink, "File should not be symlink after preview")

	info, err = os.Lstat(testFile2)
	suite.NoError(err)
	suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink, "File should not be symlink after preview")
}

func (suite *CoreTestSuite) TestPreviewAddRecursive() {
	// Initialize lnk repository
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create directory structure
	configDir := filepath.Join(suite.tempDir, ".config", "test-app")
	err = os.MkdirAll(configDir, 0755)
	suite.Require().NoError(err)

	// Create files in directory
	expectedFiles := 5
	var createdFiles []string
	for i := 1; i <= expectedFiles; i++ {
		file := filepath.Join(configDir, fmt.Sprintf("config%d.json", i))
		suite.Require().NoError(os.WriteFile(file, []byte(fmt.Sprintf("config %d", i)), 0644))
		createdFiles = append(createdFiles, file)
	}

	// Test PreviewAdd with recursive
	files, err := suite.lnk.PreviewAdd([]string{configDir}, true)
	suite.Require().NoError(err, "PreviewAdd recursive should succeed")

	// Should return all files in directory
	suite.Len(files, expectedFiles, "Should preview all files in directory")

	// Check that all created files are included
	for _, createdFile := range createdFiles {
		suite.Contains(files, createdFile, "Should include file %s", createdFile)
	}

	// Verify no actual changes were made
	for _, createdFile := range createdFiles {
		info, err := os.Lstat(createdFile)
		suite.NoError(err)
		suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink, "File should not be symlink after preview")
	}
}

func (suite *CoreTestSuite) TestPreviewAddValidation() {
	// Initialize lnk repository
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Test with nonexistent file
	nonexistentFile := filepath.Join(suite.tempDir, "nonexistent.txt")
	_, err = suite.lnk.PreviewAdd([]string{nonexistentFile}, false)
	suite.Error(err, "PreviewAdd should fail for nonexistent file")
	suite.Contains(err.Error(), "failed to stat", "Error should mention stat failure")

	// Create and add a file first
	testFile := filepath.Join(suite.tempDir, "test.txt")
	suite.Require().NoError(os.WriteFile(testFile, []byte("content"), 0644))
	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Test preview with already managed file
	_, err = suite.lnk.PreviewAdd([]string{testFile}, false)
	suite.Error(err, "PreviewAdd should fail for already managed file")
	suite.Contains(err.Error(), "already managed", "Error should mention already managed")
}

// TestAddRecursive tests recursive add operation
func (suite *CoreTestSuite) TestAddRecursive() {
	tests := []struct {
		name        string
		setupFunc   func() []string
		wantErr     bool
		errContains string
		verifyFunc  func()
	}{
		{
			name: "recursive add with single directory",
			setupFunc: func() []string {
				// Create directory with files
				testDir := filepath.Join(suite.tempDir, ".config", "app")
				err := os.MkdirAll(testDir, 0755)
				suite.Require().NoError(err)

				for i := 1; i <= 3; i++ {
					file := filepath.Join(testDir, fmt.Sprintf("file%d.txt", i))
					err := os.WriteFile(file, []byte(fmt.Sprintf("content%d", i)), 0644)
					suite.Require().NoError(err)
				}

				return []string{testDir}
			},
			wantErr: false,
			verifyFunc: func() {
				// Verify all files are now symlinks
				testDir := filepath.Join(suite.tempDir, ".config", "app")
				for i := 1; i <= 3; i++ {
					file := filepath.Join(testDir, fmt.Sprintf("file%d.txt", i))
					info, err := os.Lstat(file)
					suite.NoError(err)
					suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink, "File %d should be symlink", i)
				}
			},
		},
		{
			name: "recursive add with nested directories",
			setupFunc: func() []string {
				// Create nested structure
				testDir := filepath.Join(suite.tempDir, ".config", "nested")
				subDir := filepath.Join(testDir, "sub")
				err := os.MkdirAll(subDir, 0755)
				suite.Require().NoError(err)

				// File in parent
				parentFile := filepath.Join(testDir, "parent.txt")
				err = os.WriteFile(parentFile, []byte("parent"), 0644)
				suite.Require().NoError(err)

				// File in subdirectory
				subFile := filepath.Join(subDir, "child.txt")
				err = os.WriteFile(subFile, []byte("child"), 0644)
				suite.Require().NoError(err)

				return []string{testDir}
			},
			wantErr: false,
			verifyFunc: func() {
				// Verify both files are symlinks
				parentFile := filepath.Join(suite.tempDir, ".config", "nested", "parent.txt")
				subFile := filepath.Join(suite.tempDir, ".config", "nested", "sub", "child.txt")

				for _, file := range []string{parentFile, subFile} {
					info, err := os.Lstat(file)
					suite.NoError(err)
					suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)
				}
			},
		},
		{
			name: "recursive add with nonexistent directory",
			setupFunc: func() []string {
				return []string{filepath.Join(suite.tempDir, "nonexistent")}
			},
			wantErr:     true,
			errContains: "failed to stat",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Initialize
			err := suite.lnk.Init()
			suite.Require().NoError(err)

			// Setup
			paths := tt.setupFunc()

			// Execute AddRecursive
			err = suite.lnk.AddRecursive(paths)

			// Verify
			if tt.wantErr {
				suite.Error(err, "Expected error for test: %s", tt.name)
				if tt.errContains != "" {
					suite.Contains(err.Error(), tt.errContains, "Error message mismatch for: %s", tt.name)
				}
			} else {
				suite.NoError(err, "Unexpected error for test: %s", tt.name)
				if tt.verifyFunc != nil {
					tt.verifyFunc()
				}
			}
		})
	}
}

// TestRollbackOperations tests the rollback mechanism
func (suite *CoreTestSuite) TestRollbackOperations() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	tests := []struct {
		name       string
		setupFunc  func() ([]func() error, error)
		wantErr    bool
		verifyFunc func()
	}{
		{
			name: "rollback file move operation",
			setupFunc: func() ([]func() error, error) {
				// Create a test file
				testFile := filepath.Join(suite.tempDir, "test.txt")
				err := os.WriteFile(testFile, []byte("content"), 0644)
				if err != nil {
					return nil, err
				}

				// Move file to repo
				repoFile := filepath.Join(suite.lnk.tracker.RepoPath(), "test.txt")
				err = os.Rename(testFile, repoFile)
				if err != nil {
					return nil, err
				}

				// Get file info
				info, err := os.Stat(repoFile)
				if err != nil {
					return nil, err
				}

				// Create rollback action
				action := suite.lnk.files.CreateRollbackAction(testFile, repoFile, "test.txt", info)
				return []func() error{action}, nil
			},
			wantErr: false,
			verifyFunc: func() {
				// Verify file is back in original location
				testFile := filepath.Join(suite.tempDir, "test.txt")
				suite.FileExists(testFile, "File should be restored to original location")

				// Verify repo file is removed
				repoFile := filepath.Join(suite.lnk.tracker.RepoPath(), "test.txt")
				suite.NoFileExists(repoFile, "Repo file should be removed")

				// Verify content is preserved
				content, err := os.ReadFile(testFile)
				suite.NoError(err)
				suite.Equal("content", string(content))
			},
		},
		{
			name: "rollback multiple operations",
			setupFunc: func() ([]func() error, error) {
				actions := []func() error{}

				// Create two test files and move them
				for i := 1; i <= 2; i++ {
					testFile := filepath.Join(suite.tempDir, fmt.Sprintf("test%d.txt", i))
					err := os.WriteFile(testFile, []byte(fmt.Sprintf("content%d", i)), 0644)
					if err != nil {
						return nil, err
					}

					repoFile := filepath.Join(suite.lnk.tracker.RepoPath(), fmt.Sprintf("test%d.txt", i))
					err = os.Rename(testFile, repoFile)
					if err != nil {
						return nil, err
					}

					info, err := os.Stat(repoFile)
					if err != nil {
						return nil, err
					}

					action := suite.lnk.files.CreateRollbackAction(testFile, repoFile, fmt.Sprintf("test%d.txt", i), info)
					actions = append(actions, action)
				}

				return actions, nil
			},
			wantErr: false,
			verifyFunc: func() {
				// Verify both files are restored
				for i := 1; i <= 2; i++ {
					testFile := filepath.Join(suite.tempDir, fmt.Sprintf("test%d.txt", i))
					suite.FileExists(testFile, "File %d should be restored", i)

					content, err := os.ReadFile(testFile)
					suite.NoError(err)
					suite.Equal(fmt.Sprintf("content%d", i), string(content))
				}
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Setup
			actions, err := tt.setupFunc()
			suite.Require().NoError(err, "Setup failed for test: %s", tt.name)

			// Execute rollback
			suite.lnk.files.RollbackAll(actions)

			// Verify
			if tt.verifyFunc != nil {
				tt.verifyFunc()
			}
		})
	}
}
