package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/lnk"
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
			l := lnk.NewLnk(lnk.WithHost(host))
			w := GetWriter(cmd)

			// Handle dry-run mode
			if dryRun {
				files, err := l.PreviewAdd(args, recursive)
				if err != nil {
					return err
				}

				// Display preview output
				if recursive {
					w.Writeln(Message{Text: fmt.Sprintf("Would add %d files recursively:", len(files)), Emoji: "🔍", Bold: true})
				} else {
					w.Writeln(Message{Text: fmt.Sprintf("Would add %d files:", len(files)), Emoji: "🔍", Bold: true})
				}

				// List files using home-relative paths so duplicate basenames remain distinguishable.
				// Dry-run is a preview for verification, so show all files.
				for _, file := range files {
					w.WriteString("   ").
						Writeln(Message{Text: displaySourcePath(file), Emoji: "📄"})
				}

				w.WritelnString("").
					Writeln(Info("To proceed: run without --dry-run flag"))

				return w.Err()
			}

			// Handle recursive mode
			if recursive {
				// Get preview to count files first for better output
				previewFiles, err := l.PreviewAdd(args, recursive)
				if err != nil {
					return err
				}

				// Only show carriage-return progress when output is a terminal;
				// in piped/non-TTY contexts the redraw becomes noise.
				var progressCallback lnk.ProgressCallback
				if w.IsTerminal() {
					progressCallback = func(current, total int, currentFile string) {
						w.WriteString(fmt.Sprintf("\r⏳ Processing %d/%d: %s", current, total, currentFile))
					}
				}

				if err := l.AddRecursiveWithProgress(args, progressCallback); err != nil {
					return err
				}

				if w.IsTerminal() {
					w.WriteString("\r")
				}

				// Store processed file count for display
				args = previewFiles // Replace args with actual files for display
			} else {
				// Use appropriate method based on number of files
				if len(args) == 1 {
					// Single file - use existing Add method for backward compatibility
					if err := l.Add(args[0]); err != nil {
						return err
					}
				} else {
					// Multiple files - use AddMultiple for atomic operation
					if err := l.AddMultiple(args); err != nil {
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
				if filesToShow > displayLimit {
					filesToShow = displayLimit
				}

				for i := 0; i < filesToShow; i++ {
					w.WriteString("   ").
						Write(Link(displaySourcePath(args[i]))).
						WriteString(" → ").
						Writeln(Colored(lnk.FormatManagedPath(host, args[i]), ColorCyan))
				}

				if len(args) > displayLimit {
					w.WriteString("   ").
						Writeln(Colored(fmt.Sprintf("... and %d more files", len(args)-displayLimit), ColorGray))
				}
			} else if len(args) == 1 {
				// Single file - maintain existing output format for backward compatibility
				filePath := args[0]
				basename := filepath.Base(filePath)
				if host != "" {
					w.Writeln(Sparkles(fmt.Sprintf("Added %s to lnk (host: %s)", basename, host)))
				} else {
					w.Writeln(Sparkles(fmt.Sprintf("Added %s to lnk", basename)))
				}
				w.WriteString("   ").
					Write(Link(filePath)).
					WriteString(" → ").
					Writeln(Colored(lnk.FormatManagedPath(host, filePath), ColorCyan))
			} else {
				// Multiple files - show summary
				if host != "" {
					w.Writeln(Sparkles(fmt.Sprintf("Added %d items to lnk (host: %s)", len(args), host)))
				} else {
					w.Writeln(Sparkles(fmt.Sprintf("Added %d items to lnk", len(args))))
				}

				// List each added file using home-relative source paths so same-basename
				// files in different directories stay distinguishable. Truncate large batches.
				filesToShow := len(args)
				if filesToShow > displayLimit {
					filesToShow = displayLimit
				}
				for i := 0; i < filesToShow; i++ {
					w.WriteString("   ").
						Write(Link(displaySourcePath(args[i]))).
						WriteString(" → ").
						Writeln(Colored(lnk.FormatManagedPath(host, args[i]), ColorCyan))
				}
				if len(args) > displayLimit {
					w.WriteString("   ").
						Writeln(Colored(fmt.Sprintf("... and %d more files", len(args)-displayLimit), ColorGray))
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
	return cmd
}

// displayLimit caps the number of per-file entries shown in batch summaries
// before collapsing the remainder into "... and N more files".
const displayLimit = 5

// displaySourcePath renders a path (relative or absolute) as a home-relative
// (~/foo) display string, falling back to the original input on resolution
// failure. Used so duplicate basenames in different directories remain
// distinguishable in preview and batch output. The Abs call is idempotent
// for paths already absolute (from PreviewAdd) and normalizes relative paths.
func displaySourcePath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return lnk.DisplayPath(abs)
}
