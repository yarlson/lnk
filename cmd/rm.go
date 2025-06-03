package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "rm <file>",
		Short:         "🗑️ Remove a file from lnk management",
		Long:          "Removes a symlink and restores the original file from the lnk repository.",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			host, _ := cmd.Flags().GetString("host")

			lnk := core.NewLnk(core.WithHost(host))

			if err := lnk.Remove(filePath); err != nil {
				return err
			}

			basename := filepath.Base(filePath)
			if host != "" {
				printf(cmd, "🗑️  \033[1mRemoved %s from lnk (host: %s)\033[0m\n", basename, host)
				printf(cmd, "   ↩️  \033[90m~/.config/lnk/%s.lnk/%s\033[0m → \033[36m%s\033[0m\n", host, basename, filePath)
			} else {
				printf(cmd, "🗑️  \033[1mRemoved %s from lnk\033[0m\n", basename)
				printf(cmd, "   ↩️  \033[90m~/.config/lnk/%s\033[0m → \033[36m%s\033[0m\n", basename, filePath)
			}
			printf(cmd, "   📄 Original file restored\n")
			return nil
		},
	}

	cmd.Flags().StringP("host", "H", "", "Remove file from specific host configuration (default: common configuration)")
	return cmd
}
