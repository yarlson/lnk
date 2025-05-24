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
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "⬇️  \033[1;32mSuccessfully pulled changes\033[0m\n")
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "   🔗 Restored \033[1m%d symlink", len(restored))
				if len(restored) > 1 {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "s")
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\033[0m:\n")
				for _, file := range restored {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "      ✨ \033[36m%s\033[0m\n", file)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n   🎉 Your dotfiles are synced and ready!\n")
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "⬇️  \033[1;32mSuccessfully pulled changes\033[0m\n")
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "   ✅ All symlinks already in place\n")
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "   🎉 Everything is up to date!\n")
			}

			return nil
		},
	}
}
