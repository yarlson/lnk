package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

var rmCmd = &cobra.Command{
	Use:          "rm <file>",
	Short:        "🗑️ Remove a file from lnk management",
	Long:         "Removes a symlink and restores the original file from the lnk repository.",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		lnk := core.NewLnk()
		if err := lnk.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove file: %w", err)
		}

		basename := filepath.Base(filePath)
		fmt.Printf("🗑️  \033[1mRemoved %s from lnk\033[0m\n", basename)
		fmt.Printf("   ↩️  \033[90m~/.config/lnk/%s\033[0m → \033[36m%s\033[0m\n", basename, filePath)
		fmt.Printf("   📄 Original file restored\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
