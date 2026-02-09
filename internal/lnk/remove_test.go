package lnk

import (
	"os"
	"path/filepath"
)

// TestRemove tests Remove() function with various scenarios
func (suite *CoreTestSuite) TestRemove() {
	tests := []struct {
		name        string
		setupFunc   func() (string, error)
		wantErr     bool
		errContains string
		verifyFunc  func(filePath string)
	}{
		{
			name: "remove managed file successfully",
			setupFunc: func() (string, error) {
				err := suite.lnk.Init()
				if err != nil {
					return "", err
				}
				testFile := filepath.Join(suite.tempDir, ".testrc")
				err = os.WriteFile(testFile, []byte("test content"), 0644)
				if err != nil {
					return "", err
				}
				err = suite.lnk.Add(testFile)
				return testFile, err
			},
			wantErr: false,
			verifyFunc: func(filePath string) {
				// File should be regular file again
				info, err := os.Lstat(filePath)
				suite.NoError(err)
				suite.Equal(os.FileMode(0), info.Mode()&os.ModeSymlink, "Should not be symlink")

				// Content should be preserved
				content, err := os.ReadFile(filePath)
				suite.NoError(err)
				suite.Equal("test content", string(content))
			},
		},
		{
			name: "remove unmanaged file",
			setupFunc: func() (string, error) {
				err := suite.lnk.Init()
				if err != nil {
					return "", err
				}
				testFile := filepath.Join(suite.tempDir, "unmanaged.txt")
				err = os.WriteFile(testFile, []byte("content"), 0644)
				return testFile, err
			},
			wantErr:     true,
			errContains: "File is not managed by lnk",
		},
		{
			name: "remove nonexistent file",
			setupFunc: func() (string, error) {
				err := suite.lnk.Init()
				return filepath.Join(suite.tempDir, "nonexistent"), err
			},
			wantErr:     true,
			errContains: "File or directory not found",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			filePath, err := tt.setupFunc()
			suite.Require().NoError(err, "Setup failed for test: %s", tt.name)

			// Execute Remove
			err = suite.lnk.Remove(filePath)

			// Verify
			if tt.wantErr {
				suite.Error(err, "Expected error for test: %s", tt.name)
				if tt.errContains != "" {
					suite.Contains(err.Error(), tt.errContains, "Error message mismatch for: %s", tt.name)
				}
			} else {
				suite.NoError(err, "Unexpected error for test: %s", tt.name)
				if tt.verifyFunc != nil {
					tt.verifyFunc(filePath)
				}
			}
		})
	}
}

// TestRemoveForce tests force removal when symlink is missing
func (suite *CoreTestSuite) TestRemoveForce() {
	tests := []struct {
		name        string
		setupFunc   func() (string, error)
		wantErr     bool
		errContains string
		verifyFunc  func(filePath string)
	}{
		{
			name: "force remove when symlink is missing",
			setupFunc: func() (string, error) {
				// Initialize and add a file
				if err := suite.lnk.Init(); err != nil {
					return "", err
				}

				filePath := filepath.Join(suite.tempDir, ".bashrc")
				if err := os.WriteFile(filePath, []byte("export PATH"), 0644); err != nil {
					return "", err
				}

				if err := suite.lnk.Add(filePath); err != nil {
					return "", err
				}

				// Now delete the symlink directly (simulating user mistake)
				if err := os.Remove(filePath); err != nil {
					return "", err
				}

				return filePath, nil
			},
			wantErr: false,
			verifyFunc: func(filePath string) {
				// Verify file is no longer in tracking
				items, err := suite.lnk.getManagedItems()
				suite.NoError(err)
				suite.NotContains(items, ".bashrc")
			},
		},
		{
			name: "force remove when symlink still exists",
			setupFunc: func() (string, error) {
				// Initialize and add a file
				if err := suite.lnk.Init(); err != nil {
					return "", err
				}

				filePath := filepath.Join(suite.tempDir, ".vimrc")
				if err := os.WriteFile(filePath, []byte("set number"), 0644); err != nil {
					return "", err
				}

				if err := suite.lnk.Add(filePath); err != nil {
					return "", err
				}

				return filePath, nil
			},
			wantErr: false,
			verifyFunc: func(filePath string) {
				// Verify file is no longer in tracking
				items, err := suite.lnk.getManagedItems()
				suite.NoError(err)
				suite.NotContains(items, ".vimrc")

				// Symlink should be removed
				_, err = os.Lstat(filePath)
				suite.True(os.IsNotExist(err))
			},
		},
		{
			name: "force remove untracked file fails",
			setupFunc: func() (string, error) {
				if err := suite.lnk.Init(); err != nil {
					return "", err
				}
				return filepath.Join(suite.tempDir, ".untracked"), nil
			},
			wantErr:     true,
			errContains: "not managed",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			filePath, err := tt.setupFunc()
			suite.Require().NoError(err, "Setup failed for: %s", tt.name)

			err = suite.lnk.RemoveForce(filePath)

			if tt.wantErr {
				suite.Error(err, "Expected error for test: %s", tt.name)
				if tt.errContains != "" {
					suite.Contains(err.Error(), tt.errContains)
				}
			} else {
				suite.NoError(err, "Unexpected error for test: %s", tt.name)
				if tt.verifyFunc != nil {
					tt.verifyFunc(filePath)
				}
			}
		})
	}
}
