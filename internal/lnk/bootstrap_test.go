package lnk

import (
	"fmt"
	"os"
	"path/filepath"
)

// TestFindBootstrapScript tests bootstrap script detection
func (suite *CoreTestSuite) TestFindBootstrapScript() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Test with no bootstrap script
	scriptPath, err := suite.lnk.FindBootstrapScript()
	suite.NoError(err)
	suite.Empty(scriptPath)

	// Test with bootstrap.sh
	bootstrapScript := filepath.Join(suite.tempDir, "lnk", "bootstrap.sh")
	err = os.WriteFile(bootstrapScript, []byte("#!/bin/bash\necho 'test'"), 0644)
	suite.Require().NoError(err)

	scriptPath, err = suite.lnk.FindBootstrapScript()
	suite.NoError(err)
	suite.Equal("bootstrap.sh", scriptPath)
}

// TestRunBootstrapScript tests bootstrap script execution
func (suite *CoreTestSuite) TestRunBootstrapScript() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create a test script that creates a marker file
	bootstrapScript := filepath.Join(suite.tempDir, "lnk", "test.sh")
	markerFile := filepath.Join(suite.tempDir, "lnk", "bootstrap-executed.txt")
	scriptContent := fmt.Sprintf("#!/bin/bash\ntouch %s\necho 'Bootstrap executed'", markerFile)

	err = os.WriteFile(bootstrapScript, []byte(scriptContent), 0755)
	suite.Require().NoError(err)

	// Run the bootstrap script
	err = suite.lnk.RunBootstrapScript("test.sh", os.Stdout, os.Stderr, os.Stdin)
	suite.NoError(err)

	// Verify the marker file was created
	suite.FileExists(markerFile)
}

// TestRunBootstrapScriptWithError tests bootstrap script execution with error
func (suite *CoreTestSuite) TestRunBootstrapScriptWithError() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Create a script that will fail
	bootstrapScript := filepath.Join(suite.tempDir, "lnk", "failing.sh")
	scriptContent := "#!/bin/bash\nexit 1"

	err = os.WriteFile(bootstrapScript, []byte(scriptContent), 0755)
	suite.Require().NoError(err)

	// Run the bootstrap script - should fail
	err = suite.lnk.RunBootstrapScript("failing.sh", os.Stdout, os.Stderr, os.Stdin)
	suite.Error(err)
	suite.Contains(err.Error(), "Bootstrap script failed")
}

// TestRunBootstrapScriptNotFound tests running bootstrap on non-existent script
func (suite *CoreTestSuite) TestRunBootstrapScriptNotFound() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Try to run non-existent script
	err = suite.lnk.RunBootstrapScript("nonexistent.sh", os.Stdout, os.Stderr, os.Stdin)
	suite.Error(err)
	suite.Contains(err.Error(), "Bootstrap script not found")
}
