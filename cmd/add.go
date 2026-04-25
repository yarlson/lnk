package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/fs"
	"github.com/yarlson/lnk/internal/lnk"
	error2 "github.com/yarlson/lnk/internal/lnkerror"
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
  lnk add ~/.foo --force              # If file doesn't exist create and then add 
  lnk add ~/.foo/bar/ --force         #   trailing slash indicates a folder should be created then added

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
			force, _ := cmd.Flags().GetBool("force")
			lnk := lnk.NewLnk(lnk.WithHost(host))
			w := GetWriter(cmd)

			// Handle dry-run mode
			if dryRun {
				files, err := lnk.PreviewAdd(args, recursive)
				if err != nil {
					return err
				}

				// Display preview output
				if recursive {
					w.Writeln(Message{Text: fmt.Sprintf("Would add %d files recursively:", len(files)), Emoji: "🔍", Bold: true})
				} else {
					w.Writeln(Message{Text: fmt.Sprintf("Would add %d files:", len(files)), Emoji: "🔍", Bold: true})
				}

				// List files that would be added
				for _, file := range files {
					basename := filepath.Base(file)
					w.WriteString("   ").
						Writeln(Message{Text: basename, Emoji: "📄"})
				}

				w.WritelnString("").
					Writeln(Info("To proceed: run without --dry-run flag"))

				return w.Err()
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
					w.WriteString(fmt.Sprintf("\r⏳ Processing %d/%d: %s", current, total, currentFile))
				}

				if err := lnk.AddRecursiveWithProgress(args, progressCallback); err != nil {
					return err
				}

				// Clear progress line and show completion
				w.WriteString("\r")

				// Store processed file count for display
				args = previewFiles // Replace args with actual files for display
			} else {
				// Use appropriate method based on number of files
				if len(args) == 1 {
					// Single file - use existing Add method for backward compatibility
					if err := lnk.Add(args[0]); err != nil {
						// If Error FileNotExists and Force - Create and Add
						var addErr *error2.Error
						if errors.As(err, &addErr) {
							if addErr.Err == fs.ErrFileNotExists && force {
								if err := lnk.Create(args[0]); err != nil {
									return err
								}
								w.Writeln(Colored(fmt.Sprintf("%s didn't exist - created", args[0]), ColorCyan))
							} else {
								return err
							}
						}
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
					w.Writeln(Sparkles(fmt.Sprintf("Added %d files recursively to lnk (host: %s)", len(args), host)))
				} else {
					w.Writeln(Sparkles(fmt.Sprintf("Added %d files recursively to lnk", len(args))))
				}

				// Show some of the files that were added (limit to first few for readability)
				filesToShow := len(args)
				if filesToShow > 5 {
					filesToShow = 5
				}

				for i := 0; i < filesToShow; i++ {
					basename := filepath.Base(args[i])
					if host != "" {
						w.WriteString("   ").
							Write(Link(basename)).
							WriteString(" → ").
							Writeln(Colored(fmt.Sprintf("~/.config/lnk/%s.lnk/...", host), ColorCyan))
					} else {
						w.WriteString("   ").
							Write(Link(basename)).
							WriteString(" → ").
							Writeln(Colored("~/.config/lnk/...", ColorCyan))
					}
				}

				if len(args) > 5 {
					w.WriteString("   ").
						Writeln(Colored(fmt.Sprintf("... and %d more files", len(args)-5), ColorGray))
				}
			} else if len(args) == 1 {
				// Single file - maintain existing output format for backward compatibility
				filePath := args[0]
				basename := filepath.Base(filePath)
				if host != "" {
					w.Writeln(Sparkles(fmt.Sprintf("Added %s to lnk (host: %s)", basename, host)))
					w.WriteString("   ").
						Write(Link(filePath)).
						WriteString(" → ").
						Writeln(Colored(fmt.Sprintf("~/.config/lnk/%s.lnk/%s", host, filePath), ColorCyan))
				} else {
					w.Writeln(Sparkles(fmt.Sprintf("Added %s to lnk", basename)))
					w.WriteString("   ").
						Write(Link(filePath)).
						WriteString(" → ").
						Writeln(Colored(fmt.Sprintf("~/.config/lnk/%s", filePath), ColorCyan))
				}
			} else {
				// Multiple files - show summary
				if host != "" {
					w.Writeln(Sparkles(fmt.Sprintf("Added %d items to lnk (host: %s)", len(args), host)))
				} else {
					w.Writeln(Sparkles(fmt.Sprintf("Added %d items to lnk", len(args))))
				}

				// List each added file
				for _, filePath := range args {
					basename := filepath.Base(filePath)
					if host != "" {
						w.WriteString("   ").
							Write(Link(basename)).
							WriteString(" → ").
							Writeln(Colored(fmt.Sprintf("~/.config/lnk/%s.lnk/...", host), ColorCyan))
					} else {
						w.WriteString("   ").
							Write(Link(basename)).
							WriteString(" → ").
							Writeln(Colored("~/.config/lnk/...", ColorCyan))
					}
				}
			}

			w.WriteString("   ").
				Write(Message{Text: "Use ", Emoji: "📝"}).
				Write(Bold("lnk push")).
				WritelnString(" to sync to remote")

			return w.Err()
		},
	}

	cmd.Flags().StringP("host", "H", "", "Manage file for specific host (default: common configuration)")
	cmd.Flags().BoolP("recursive", "r", false, "Add directory contents individually instead of the directory as a whole")
	cmd.Flags().BoolP("dry-run", "n", false, "Show what would be added without making changes")
	cmd.Flags().Bool("force", false, "Force add a file or directory even if it doesn't exist (WARNING: check file or directory permissions as needed)")
	return cmd
}
