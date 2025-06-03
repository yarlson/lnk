package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

func newPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "pull",
		Short:         "⬇️ Pull changes from remote and restore symlinks",
		Long:          "Fetches changes from remote repository and automatically restores symlinks for all managed files.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")

			lnk := core.NewLnk(core.WithHost(host))

			restored, err := lnk.Pull()
			if err != nil {
				return err
			}

			if len(restored) > 0 {
				if host != "" {
					printf(cmd, "⬇️  \033[1;32mSuccessfully pulled changes (host: %s)\033[0m\n", host)
				} else {
					printf(cmd, "⬇️  \033[1;32mSuccessfully pulled changes\033[0m\n")
				}
				printf(cmd, "   🔗 Restored \033[1m%d symlink", len(restored))
				if len(restored) > 1 {
					printf(cmd, "s")
				}
				printf(cmd, "\033[0m:\n")
				for _, file := range restored {
					printf(cmd, "      ✨ \033[36m%s\033[0m\n", file)
				}
				printf(cmd, "\n   🎉 Your dotfiles are synced and ready!\n")
			} else {
				if host != "" {
					printf(cmd, "⬇️  \033[1;32mSuccessfully pulled changes (host: %s)\033[0m\n", host)
				} else {
					printf(cmd, "⬇️  \033[1;32mSuccessfully pulled changes\033[0m\n")
				}
				printf(cmd, "   ✅ All symlinks already in place\n")
				printf(cmd, "   🎉 Everything is up to date!\n")
			}

			return nil
		},
	}

	cmd.Flags().StringP("host", "H", "", "Pull and restore symlinks for specific host (default: common configuration)")
	return cmd
}
