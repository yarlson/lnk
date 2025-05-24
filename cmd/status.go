package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "status",
		Short:        "📊 Show repository sync status",
		Long:         "Display how many commits ahead/behind the local repository is relative to the remote.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			lnk := core.NewLnk()
			status, err := lnk.Status()
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}

			if status.Ahead == 0 && status.Behind == 0 {
				printf(cmd, "✅ \033[1;32mRepository is up to date\033[0m\n")
				printf(cmd, "   📡 Synced with \033[36m%s\033[0m\n", status.Remote)
			} else {
				printf(cmd, "📊 \033[1mRepository Status\033[0m\n")
				printf(cmd, "   📡 Remote: \033[36m%s\033[0m\n", status.Remote)
				printf(cmd, "\n")

				if status.Ahead > 0 {
					commitText := "commit"
					if status.Ahead > 1 {
						commitText = "commits"
					}
					printf(cmd, "   ⬆️  \033[1;33m%d %s ahead\033[0m - ready to push\n", status.Ahead, commitText)
				}
				if status.Behind > 0 {
					commitText := "commit"
					if status.Behind > 1 {
						commitText = "commits"
					}
					printf(cmd, "   ⬇️  \033[1;31m%d %s behind\033[0m - run \033[1mlnk pull\033[0m\n", status.Behind, commitText)
				}

				if status.Ahead > 0 && status.Behind == 0 {
					printf(cmd, "\n💡 Run \033[1mlnk push\033[0m to sync your changes")
				} else if status.Behind > 0 {
					printf(cmd, "\n💡 Run \033[1mlnk pull\033[0m to get latest changes")
				}
			}

			return nil
		},
	}
}
