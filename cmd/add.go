package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "add <file>...",
		Short:         "✨ Add files to lnk management",
		Long:          "Moves files to the lnk repository and creates symlinks in their place. Supports multiple files.",
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			recursive, _ := cmd.Flags().GetBool("recursive")
			lnk := core.NewLnk(core.WithHost(host))

			// Handle recursive mode
			if recursive {
				// Create progress callback for CLI display
				progressCallback := func(current, total int, currentFile string) {
					printf(cmd, "\r⏳ Processing %d/%d: %s", current, total, currentFile)
				}
				
				if err := lnk.AddRecursiveWithProgress(args, progressCallback); err != nil {
					return err
				}
				
				// Clear progress line and show completion
				printf(cmd, "\r")
			} else {
				// Use appropriate method based on number of files
				if len(args) == 1 {
					// Single file - use existing Add method for backward compatibility
					if err := lnk.Add(args[0]); err != nil {
						return err
					}
				} else {
					// Multiple files - use AddMultiple for atomic operation
					if err := lnk.AddMultiple(args); err != nil {
						return err
					}
				}
			}

			// Display results
			if recursive {
				// Recursive mode - show different message
				if host != "" {
					printf(cmd, "✨ \033[1mAdded files recursively to lnk (host: %s)\033[0m\n", host)
				} else {
					printf(cmd, "✨ \033[1mAdded files recursively to lnk\033[0m\n")
				}
			} else if len(args) == 1 {
				// Single file - maintain existing output format for backward compatibility
				filePath := args[0]
				basename := filepath.Base(filePath)
				if host != "" {
					printf(cmd, "✨ \033[1mAdded %s to lnk (host: %s)\033[0m\n", basename, host)
					printf(cmd, "   🔗 \033[90m%s\033[0m → \033[36m~/.config/lnk/%s.lnk/%s\033[0m\n", filePath, host, filePath)
				} else {
					printf(cmd, "✨ \033[1mAdded %s to lnk\033[0m\n", basename)
					printf(cmd, "   🔗 \033[90m%s\033[0m → \033[36m~/.config/lnk/%s\033[0m\n", filePath, filePath)
				}
			} else {
				// Multiple files - show summary
				if host != "" {
					printf(cmd, "✨ \033[1mAdded %d items to lnk (host: %s)\033[0m\n", len(args), host)
				} else {
					printf(cmd, "✨ \033[1mAdded %d items to lnk\033[0m\n", len(args))
				}

				// List each added file
				for _, filePath := range args {
					basename := filepath.Base(filePath)
					if host != "" {
						printf(cmd, "   🔗 \033[90m%s\033[0m → \033[36m~/.config/lnk/%s.lnk/...\033[0m\n", basename, host)
					} else {
						printf(cmd, "   🔗 \033[90m%s\033[0m → \033[36m~/.config/lnk/...\033[0m\n", basename)
					}
				}
			}

			printf(cmd, "   📝 Use \033[1mlnk push\033[0m to sync to remote\n")
			return nil
		},
	}

	cmd.Flags().StringP("host", "H", "", "Manage file for specific host (default: common configuration)")
	cmd.Flags().BoolP("recursive", "r", false, "Add directory contents individually instead of the directory as a whole")
	return cmd
}
