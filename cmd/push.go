package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "push [message]",
		Short:        "🚀 Push local changes to remote repository",
		Long:         "Stages all changes, creates a sync commit with the provided message, and pushes to remote.",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			message := "lnk: sync configuration files"
			if len(args) > 0 {
				message = args[0]
			}

			lnk := core.NewLnk()
			if err := lnk.Push(message); err != nil {
				return fmt.Errorf("failed to push changes: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "🚀 \033[1;32mSuccessfully pushed changes\033[0m\n")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "   💾 Commit: \033[90m%s\033[0m\n", message)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "   📡 Synced to remote\n")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "   ✨ Your dotfiles are up to date!\n")
			return nil
		},
	}
}
