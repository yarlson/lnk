package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

func newPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "pull",
		Short:        "⬇️ Pull changes from remote and restore symlinks",
		Long:         "Fetches changes from remote repository and automatically restores symlinks for all managed files.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			lnk := core.NewLnk()
			restored, err := lnk.Pull()
			if err != nil {
				return fmt.Errorf("failed to pull changes: %w", err)
			}

			if len(restored) > 0 {
				printf(cmd, "⬇️  \033[1;32mSuccessfully pulled changes\033[0m\n")
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
				printf(cmd, "⬇️  \033[1;32mSuccessfully pulled changes\033[0m\n")
				printf(cmd, "   ✅ All symlinks already in place\n")
				printf(cmd, "   🎉 Everything is up to date!\n")
			}

			return nil
		},
	}
}
