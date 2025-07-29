package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <file>...",
		Short: "✨ Add files to lnk management",
		Long: `Moves files to the lnk repository and creates symlinks in their place. Supports multiple files.

Examples:
  lnk add ~/.bashrc ~/.vimrc          # Add multiple files at once
  lnk add --recursive ~/.config/nvim  # Add directory contents individually  
  lnk add --dry-run ~/.gitconfig      # Preview what would be added
  lnk add --host work ~/.ssh/config   # Add host-specific configuration

The --recursive flag processes directory contents individually instead of treating 
the directory as a single unit. This is useful for configuration directories where
you want each file managed separately.

The --dry-run flag shows you exactly what files would be added without making any
changes to your system - perfect for verification before bulk operations.`,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			recursive, _ := cmd.Flags().GetBool("recursive")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			lnk := core.NewLnk(core.WithHost(host))

			// Handle dry-run mode
			if dryRun {
				files, err := lnk.PreviewAdd(args, recursive)
				if err != nil {
					return err
				}

				// Display preview output
				if recursive {
					printf(cmd, "🔍 \033[1mWould add %d files recursively:\033[0m\n", len(files))
				} else {
					printf(cmd, "🔍 \033[1mWould add %d files:\033[0m\n", len(files))
				}

				// List files that would be added
				for _, file := range files {
					basename := filepath.Base(file)
					printf(cmd, "   📄 \033[90m%s\033[0m\n", basename)
				}

				printf(cmd, "\n💡 \033[33mTo proceed:\033[0m run without --dry-run flag\n")
				return nil
			}

			// Handle recursive mode
			if recursive {
				// Get preview to count files first for better output
				previewFiles, err := lnk.PreviewAdd(args, recursive)
				if err != nil {
					return err
				}

				// Create progress callback for CLI display
				progressCallback := func(current, total int, currentFile string) {
					printf(cmd, "\r⏳ Processing %d/%d: %s", current, total, currentFile)
				}

				if err := lnk.AddRecursiveWithProgress(args, progressCallback); err != nil {
					return err
				}

				// Clear progress line and show completion
				printf(cmd, "\r")

				// Store processed file count for display
				args = previewFiles // Replace args with actual files for display
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
				// Recursive mode - show enhanced message with count
				if host != "" {
					printf(cmd, "✨ \033[1mAdded %d files recursively to lnk (host: %s)\033[0m\n", len(args), host)
				} else {
					printf(cmd, "✨ \033[1mAdded %d files recursively to lnk\033[0m\n", len(args))
				}

				// Show some of the files that were added (limit to first few for readability)
				filesToShow := len(args)
				if filesToShow > 5 {
					filesToShow = 5
				}

				for i := 0; i < filesToShow; i++ {
					basename := filepath.Base(args[i])
					if host != "" {
						printf(cmd, "   🔗 \033[90m%s\033[0m → \033[36m~/.config/lnk/%s.lnk/...\033[0m\n", basename, host)
					} else {
						printf(cmd, "   🔗 \033[90m%s\033[0m → \033[36m~/.config/lnk/...\033[0m\n", basename)
					}
				}

				if len(args) > 5 {
					printf(cmd, "   \033[90m... and %d more files\033[0m\n", len(args)-5)
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
	cmd.Flags().BoolP("dry-run", "n", false, "Show what would be added without making changes")
	return cmd
}
