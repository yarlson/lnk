package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

var addCmd = &cobra.Command{
	Use:   "add <file>",
	Short: "Add a file to lnk management",
	Long:  "Moves a file to the lnk repository and creates a symlink in its place.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		lnk := core.NewLnk()
		if err := lnk.Add(filePath); err != nil {
			return fmt.Errorf("failed to add file: %w", err)
		}

		basename := filepath.Base(filePath)
		fmt.Printf("Added %s to lnk\n", basename)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
