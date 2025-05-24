package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

var rmCmd = &cobra.Command{
	Use:   "rm <file>",
	Short: "Remove a file from lnk management",
	Long:  "Removes a symlink and restores the original file from the lnk repository.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		lnk := core.NewLnk()
		if err := lnk.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove file: %w", err)
		}

		basename := filepath.Base(filePath)
		fmt.Printf("Removed %s from lnk\n", basename)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
