package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

var pushCmd = &cobra.Command{
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

		fmt.Printf("🚀 \033[1;32mSuccessfully pushed changes\033[0m\n")
		fmt.Printf("   💾 Commit: \033[90m%s\033[0m\n", message)
		fmt.Printf("   📡 Synced to remote\n")
		fmt.Printf("   ✨ Your dotfiles are up to date!\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
