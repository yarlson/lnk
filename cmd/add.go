package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "add <file>",
		Short:         "✨ Add a file to lnk management",
		Long:          "Moves a file to the lnk repository and creates a symlink in its place.",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			host, _ := cmd.Flags().GetString("host")

			lnk := core.NewLnk(core.WithHost(host))

			if err := lnk.Add(filePath); err != nil {
				return err
			}

			basename := filepath.Base(filePath)
			if host != "" {
				printf(cmd, "✨ \033[1mAdded %s to lnk (host: %s)\033[0m\n", basename, host)
				printf(cmd, "   🔗 \033[90m%s\033[0m → \033[36m~/.config/lnk/%s.lnk/%s\033[0m\n", filePath, host, filePath)
			} else {
				printf(cmd, "✨ \033[1mAdded %s to lnk\033[0m\n", basename)
				printf(cmd, "   🔗 \033[90m%s\033[0m → \033[36m~/.config/lnk/%s\033[0m\n", filePath, filePath)
			}
			printf(cmd, "   📝 Use \033[1mlnk push\033[0m to sync to remote\n")
			return nil
		},
	}

	cmd.Flags().StringP("host", "H", "", "Manage file for specific host (default: common configuration)")
	return cmd
}
