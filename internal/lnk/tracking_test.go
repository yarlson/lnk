package lnk

import (
	"os"
	"path/filepath"
)

// TestAddManagedItem tests managed items tracking
func (suite *CoreTestSuite) TestAddManagedItem() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	tests := []struct {
		name         string
		relativePath string
		wantErr      bool
	}{
		{
			name:         "add new item",
			relativePath: ".bashrc",
			wantErr:      false,
		},
		{
			name:         "add duplicate item (should be idempotent)",
			relativePath: ".bashrc",
			wantErr:      false,
		},
		{
			name:         "add another item",
			relativePath: ".vimrc",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := suite.lnk.tracker.AddManagedItem(tt.relativePath)
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
		})
	}

	// Verify all items are tracked
	items, err := suite.lnk.tracker.GetManagedItems()
	suite.NoError(err)
	suite.Len(items, 2) // Only 2 unique items
	suite.Contains(items, ".bashrc")
	suite.Contains(items, ".vimrc")
}

// TestGetManagedItems tests getting managed items
func (suite *CoreTestSuite) TestGetManagedItems() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	tests := []struct {
		name          string
		setupFunc     func() error
		expectedCount int
		expectedItems []string
	}{
		{
			name: "empty tracking file",
			setupFunc: func() error {
				return nil
			},
			expectedCount: 0,
			expectedItems: []string{},
		},
		{
			name: "with tracked items",
			setupFunc: func() error {
				lnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk")
				return os.WriteFile(lnkFile, []byte(".bashrc\n.vimrc\n"), 0644)
			},
			expectedCount: 2,
			expectedItems: []string{".bashrc", ".vimrc"},
		},
		{
			name: "with empty lines",
			setupFunc: func() error {
				lnkFile := filepath.Join(suite.tempDir, "lnk", ".lnk")
				return os.WriteFile(lnkFile, []byte(".bashrc\n\n.vimrc\n  \n"), 0644)
			},
			expectedCount: 2,
			expectedItems: []string{".bashrc", ".vimrc"},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := tt.setupFunc()
			suite.Require().NoError(err)

			items, err := suite.lnk.tracker.GetManagedItems()
			suite.NoError(err)
			suite.Len(items, tt.expectedCount)

			for _, expected := range tt.expectedItems {
				suite.Contains(items, expected)
			}
		})
	}
}

// TestRemoveManagedItem tests removing items from tracking
func (suite *CoreTestSuite) TestRemoveManagedItem() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	// Add some items first
	_ = suite.lnk.tracker.AddManagedItem(".bashrc")
	_ = suite.lnk.tracker.AddManagedItem(".vimrc")
	_ = suite.lnk.tracker.AddManagedItem(".gitconfig")

	tests := []struct {
		name         string
		relativePath string
		wantErr      bool
	}{
		{
			name:         "remove existing item",
			relativePath: ".bashrc",
			wantErr:      false,
		},
		{
			name:         "remove another item",
			relativePath: ".vimrc",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := suite.lnk.tracker.RemoveManagedItem(tt.relativePath)
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
		})
	}

	// Verify only one item remains
	items, err := suite.lnk.tracker.GetManagedItems()
	suite.NoError(err)
	suite.Len(items, 1)
	suite.Contains(items, ".gitconfig")
}

// TestWriteManagedItems tests writing managed items
func (suite *CoreTestSuite) TestWriteManagedItems() {
	err := suite.lnk.Init()
	suite.Require().NoError(err)

	tests := []struct {
		name  string
		items []string
	}{
		{
			name:  "write multiple items",
			items: []string{".bashrc", ".vimrc", ".gitconfig"},
		},
		{
			name:  "write empty list",
			items: []string{},
		},
		{
			name:  "write single item",
			items: []string{".bashrc"},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := suite.lnk.tracker.WriteManagedItems(tt.items)
			suite.NoError(err)

			// Verify by reading back
			items, err := suite.lnk.tracker.GetManagedItems()
			suite.NoError(err)
			suite.ElementsMatch(tt.items, items)
		})
	}
}
