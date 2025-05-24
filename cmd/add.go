package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "add <file>",
		Short:        "✨ Add a file to lnk management",
		Long:         "Moves a file to the lnk repository and creates a symlink in its place.",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			lnk := core.NewLnk()
			if err := lnk.Add(filePath); err != nil {
				return fmt.Errorf("failed to add file: %w", err)
			}

			basename := filepath.Base(filePath)
			printf(cmd, "✨ \033[1mAdded %s to lnk\033[0m\n", basename)
			printf(cmd, "   🔗 \033[90m%s\033[0m → \033[36m~/.config/lnk/%s\033[0m\n", filePath, basename)
			printf(cmd, "   📝 Use \033[1mlnk push\033[0m to sync to remote\n")
			return nil
		},
	}
}
