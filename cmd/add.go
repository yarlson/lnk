package cmd

import (
	"context"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/service"
)

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "add <file>",
		Short:        "✨ Add a file to lnk management",
		Long:         "Moves a file to the lnk repository and creates a symlink in its place.",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			host, _ := cmd.Flags().GetString("host")

			// Create service instance
			lnkService, err := service.New()
			if err != nil {
				return wrapServiceError("initialize lnk service", err)
			}

			// Add file using service layer
			ctx := context.Background()
			managedFile, err := lnkService.AddFile(ctx, filePath, host)
			if err != nil {
				return formatError(err)
			}

			// Display success message
			basename := filepath.Base(filePath)
			if host != "" {
				printf(cmd, "✨ \033[1mAdded %s to lnk (host: %s)\033[0m\n", basename, host)
				printf(cmd, "   🔗 \033[90m%s\033[0m → \033[36m~/.config/lnk/%s.lnk/%s\033[0m\n", managedFile.OriginalPath, host, managedFile.RelativePath)
			} else {
				printf(cmd, "✨ \033[1mAdded %s to lnk\033[0m\n", basename)
				printf(cmd, "   🔗 \033[90m%s\033[0m → \033[36m~/.config/lnk/%s\033[0m\n", managedFile.OriginalPath, managedFile.RelativePath)
			}
			printf(cmd, "   📝 Use \033[1mlnk push\033[0m to sync to remote\n")
			return nil
		},
	}

	cmd.Flags().StringP("host", "H", "", "Manage file for specific host (default: common configuration)")
	return cmd
}
