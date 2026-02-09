package lnk

import (
	"os"
	"os/exec"
	"path/filepath"
)

// TestDoctorNoInvalidEntries tests Doctor with no issues
func (suite *CoreTestSuite) TestDoctorNoInvalidEntries() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add a real file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Run doctor — nothing should be invalid
	result, err := suite.lnk.Doctor()
	suite.Require().NoError(err)
	suite.False(result.HasIssues())
	suite.Empty(result.InvalidEntries)
	suite.Empty(result.BrokenSymlinks)
}

// TestDoctorRemovesInvalidEntries tests Doctor removes entries for missing repo files
func (suite *CoreTestSuite) TestDoctorRemovesInvalidEntries() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add a real file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Add a second file so we can make it invalid
	testFile2 := filepath.Join(suite.tempDir, ".vimrc")
	err = os.WriteFile(testFile2, []byte("set number"), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testFile2)
	suite.Require().NoError(err)

	// Now manually delete .vimrc from the repo (simulate file disappearing)
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	repoVimrc := filepath.Join(lnkDir, ".vimrc")
	err = os.Remove(repoVimrc)
	suite.Require().NoError(err)

	// Run doctor — should remove .vimrc from tracking
	result, err := suite.lnk.Doctor()
	suite.Require().NoError(err)
	suite.Equal([]string{".vimrc"}, result.InvalidEntries)

	// Verify .lnk file only contains .bashrc
	items, err := suite.lnk.getManagedItems()
	suite.Require().NoError(err)
	suite.Equal([]string{".bashrc"}, items)

	// Verify a git commit was created with doctor message
	commits, err := suite.lnk.GetCommits()
	suite.Require().NoError(err)
	suite.Contains(commits[0], "lnk: cleaned 1 invalid entry")
}

// TestDoctorRemovesPathTraversalEntries tests Doctor with path traversal entries
func (suite *CoreTestSuite) TestDoctorRemovesPathTraversalEntries() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Manually write .lnk file with a path traversal entry
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	lnkFile := filepath.Join(lnkDir, ".lnk")
	err = os.WriteFile(lnkFile, []byte("../../etc/passwd\n"), 0644)
	suite.Require().NoError(err)

	// Stage the .lnk file
	cmd := exec.Command("git", "add", ".lnk")
	cmd.Dir = lnkDir
	err = cmd.Run()
	suite.Require().NoError(err)

	cmd = exec.Command("git", "-c", "user.email=test@test.com", "-c", "user.name=Test", "commit", "-m", "add .lnk")
	cmd.Dir = lnkDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// Run doctor — should remove the traversal entry
	result, err := suite.lnk.Doctor()
	suite.Require().NoError(err)
	suite.Equal([]string{"../../etc/passwd"}, result.InvalidEntries)

	// Verify .lnk file is now empty
	items, err := suite.lnk.getManagedItems()
	suite.Require().NoError(err)
	suite.Empty(items)
}

// TestDoctorNotInitialized tests Doctor on uninitialized repo
func (suite *CoreTestSuite) TestDoctorNotInitialized() {
	result, err := suite.lnk.Doctor()
	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "Lnk repository not initialized")
}

// TestDoctorEmptyLnkFile tests Doctor with empty .lnk file
func (suite *CoreTestSuite) TestDoctorEmptyLnkFile() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Run doctor on empty tracking
	result, err := suite.lnk.Doctor()
	suite.Require().NoError(err)
	suite.False(result.HasIssues())
}

// TestDoctorFixesBrokenSymlinks tests Doctor detects and fixes broken symlinks
func (suite *CoreTestSuite) TestDoctorFixesBrokenSymlinks() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add a file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Remove the symlink to simulate a broken symlink scenario
	err = os.Remove(testFile)
	suite.Require().NoError(err)

	// Run doctor — should detect and fix the broken symlink
	result, err := suite.lnk.Doctor()
	suite.Require().NoError(err)
	suite.Equal([]string{".bashrc"}, result.BrokenSymlinks)
	suite.Empty(result.InvalidEntries)

	// Verify the symlink was restored
	info, err := os.Lstat(testFile)
	suite.Require().NoError(err)
	suite.Equal(os.ModeSymlink, info.Mode()&os.ModeSymlink)
}

// TestDoctorCombinedIssues tests Doctor handles combined issues
func (suite *CoreTestSuite) TestDoctorCombinedIssues() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add two files
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	testFile2 := filepath.Join(suite.tempDir, ".vimrc")
	err = os.WriteFile(testFile2, []byte("set number"), 0644)
	suite.Require().NoError(err)
	err = suite.lnk.Add(testFile2)
	suite.Require().NoError(err)

	lnkDir := filepath.Join(suite.tempDir, "lnk")

	// 1. Create an invalid entry: delete .vimrc from repo
	repoVimrc := filepath.Join(lnkDir, ".vimrc")
	err = os.Remove(repoVimrc)
	suite.Require().NoError(err)

	// 2. Create a broken symlink: remove .bashrc symlink
	err = os.Remove(testFile)
	suite.Require().NoError(err)

	// Run doctor
	result, err := suite.lnk.Doctor()
	suite.Require().NoError(err)
	suite.Equal([]string{".vimrc"}, result.InvalidEntries)
	suite.Equal([]string{".bashrc"}, result.BrokenSymlinks)
	suite.Equal(2, result.TotalIssues())
}

// TestPreviewDoctorNoInvalidEntries tests PreviewDoctor with no issues
func (suite *CoreTestSuite) TestPreviewDoctorNoInvalidEntries() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add a real file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)

	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Preview doctor — nothing should be invalid
	result, err := suite.lnk.PreviewDoctor()
	suite.Require().NoError(err)
	suite.False(result.HasIssues())
}

// TestPreviewDoctorReturnsInvalidEntries tests PreviewDoctor returns invalid entries without modifying state
func (suite *CoreTestSuite) TestPreviewDoctorReturnsInvalidEntries() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add two files
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	testFile2 := filepath.Join(suite.tempDir, ".vimrc")
	err = os.WriteFile(testFile2, []byte("set number"), 0644)
	suite.Require().NoError(err)
	err = suite.lnk.Add(testFile2)
	suite.Require().NoError(err)

	// Delete .vimrc from the repo to create an invalid entry
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	repoVimrc := filepath.Join(lnkDir, ".vimrc")
	err = os.Remove(repoVimrc)
	suite.Require().NoError(err)

	// Get commit count before preview
	commitsBefore, err := suite.lnk.GetCommits()
	suite.Require().NoError(err)

	// Preview doctor — should report .vimrc as invalid
	result, err := suite.lnk.PreviewDoctor()
	suite.Require().NoError(err)
	suite.Equal([]string{".vimrc"}, result.InvalidEntries)

	// Verify NO mutation: .lnk file still contains both entries
	items, err := suite.lnk.getManagedItems()
	suite.Require().NoError(err)
	suite.Equal([]string{".bashrc", ".vimrc"}, items)

	// Verify NO git commit was created
	commitsAfter, err := suite.lnk.GetCommits()
	suite.Require().NoError(err)
	suite.Equal(len(commitsBefore), len(commitsAfter))
}

// TestPreviewDoctorPathTraversalEntries tests PreviewDoctor with path traversal entries
func (suite *CoreTestSuite) TestPreviewDoctorPathTraversalEntries() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Manually write .lnk file with a path traversal entry
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	lnkFile := filepath.Join(lnkDir, ".lnk")
	err = os.WriteFile(lnkFile, []byte("../../etc/passwd\n"), 0644)
	suite.Require().NoError(err)

	// Stage the .lnk file
	cmd := exec.Command("git", "add", ".lnk")
	cmd.Dir = lnkDir
	err = cmd.Run()
	suite.Require().NoError(err)

	cmd = exec.Command("git", "-c", "user.email=test@test.com", "-c", "user.name=Test", "commit", "-m", "add .lnk")
	cmd.Dir = lnkDir
	err = cmd.Run()
	suite.Require().NoError(err)

	// Preview doctor — should detect the traversal entry
	result, err := suite.lnk.PreviewDoctor()
	suite.Require().NoError(err)
	suite.Equal([]string{"../../etc/passwd"}, result.InvalidEntries)

	// Verify NO mutation: .lnk file still contains the traversal entry
	items, err := suite.lnk.getManagedItems()
	suite.Require().NoError(err)
	suite.Equal([]string{"../../etc/passwd"}, items)
}

// TestPreviewDoctorNotInitialized tests PreviewDoctor on uninitialized repo
func (suite *CoreTestSuite) TestPreviewDoctorNotInitialized() {
	result, err := suite.lnk.PreviewDoctor()
	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "Lnk repository not initialized")
}

// TestPreviewDoctorEmptyLnkFile tests PreviewDoctor with empty .lnk file
func (suite *CoreTestSuite) TestPreviewDoctorEmptyLnkFile() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Preview doctor on empty tracking
	result, err := suite.lnk.PreviewDoctor()
	suite.Require().NoError(err)
	suite.False(result.HasIssues())
}

// TestPreviewDoctorWithHost tests PreviewDoctor with host-specific configuration
func (suite *CoreTestSuite) TestPreviewDoctorWithHost() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add a host-specific file
	hostLnk := NewLnk(WithHost("work"))

	testFile := filepath.Join(suite.tempDir, ".vimrc")
	err = os.WriteFile(testFile, []byte("set number"), 0644)
	suite.Require().NoError(err)
	err = hostLnk.Add(testFile)
	suite.Require().NoError(err)

	// Delete from host storage to create invalid entry
	lnkDir := filepath.Join(suite.tempDir, "lnk")
	hostStoredFile := filepath.Join(lnkDir, "work.lnk", ".vimrc")
	err = os.Remove(hostStoredFile)
	suite.Require().NoError(err)

	// Preview doctor with host — should detect the invalid entry
	result, err := hostLnk.PreviewDoctor()
	suite.Require().NoError(err)
	suite.Equal([]string{".vimrc"}, result.InvalidEntries)

	// Verify NO mutation: .lnk.work file still contains the entry
	items, err := hostLnk.getManagedItems()
	suite.Require().NoError(err)
	suite.Equal([]string{".vimrc"}, items)
}

// TestPreviewDoctorDetectsBrokenSymlinks tests PreviewDoctor detects broken symlinks
func (suite *CoreTestSuite) TestPreviewDoctorDetectsBrokenSymlinks() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add a file
	testFile := filepath.Join(suite.tempDir, ".bashrc")
	err = os.WriteFile(testFile, []byte("export PATH=/usr/local/bin:$PATH"), 0644)
	suite.Require().NoError(err)
	err = suite.lnk.Add(testFile)
	suite.Require().NoError(err)

	// Remove the symlink to simulate a broken symlink
	err = os.Remove(testFile)
	suite.Require().NoError(err)

	// Get commit count before preview
	commitsBefore, err := suite.lnk.GetCommits()
	suite.Require().NoError(err)

	// Preview doctor — should detect the broken symlink
	result, err := suite.lnk.PreviewDoctor()
	suite.Require().NoError(err)
	suite.Equal([]string{".bashrc"}, result.BrokenSymlinks)
	suite.Empty(result.InvalidEntries)

	// Verify NO mutation: symlink was not restored
	_, err = os.Lstat(testFile)
	suite.True(os.IsNotExist(err))

	// Verify NO git commit was created
	commitsAfter, err := suite.lnk.GetCommits()
	suite.Require().NoError(err)
	suite.Equal(len(commitsBefore), len(commitsAfter))
}
