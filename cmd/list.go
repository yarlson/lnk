package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "list",
		Short:        "📋 List files managed by lnk",
		Long:         "Display all files and directories currently managed by lnk.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			lnk := core.NewLnk()
			managedItems, err := lnk.List()
			if err != nil {
				return fmt.Errorf("failed to list managed items: %w", err)
			}

			if len(managedItems) == 0 {
				printf(cmd, "📋 \033[1mNo files currently managed by lnk\033[0m\n")
				printf(cmd, "   💡 Use \033[1mlnk add <file>\033[0m to start managing files\n")
				return nil
			}

			printf(cmd, "📋 \033[1mFiles managed by lnk\033[0m (\033[36m%d item", len(managedItems))
			if len(managedItems) > 1 {
				printf(cmd, "s")
			}
			printf(cmd, "\033[0m):\n\n")

			for _, item := range managedItems {
				printf(cmd, "   🔗 \033[36m%s\033[0m\n", item)
			}

			printf(cmd, "\n💡 Use \033[1mlnk status\033[0m to check sync status\n")
			return nil
		},
	}
}
