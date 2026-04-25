package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/lnk"
)

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm <file>",
		Short: "🗑️ Remove a file from lnk management",
		Long: `Removes a symlink and restores the original file from the lnk repository.

Use --force for tracking cleanup only: it removes the entry from the .lnk index
and the stored file from the repo without restoring anything in your home
directory. This is intended for cases where the symlink is already missing
(e.g., you deleted it manually) so the regular rm flow cannot run. --force
does NOT recreate or move any file back into place.`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			host, _ := cmd.Flags().GetString("host")
			force, _ := cmd.Flags().GetBool("force")
			l := lnk.NewLnk(lnk.WithHost(host))
			w := GetWriter(cmd)

			if force {
				if err := l.RemoveForce(filePath); err != nil {
					return err
				}

				basename := filepath.Base(filePath)
				if host != "" {
					w.Writeln(Message{Text: fmt.Sprintf("Force removed %s from lnk (host: %s)", basename, host), Emoji: "🗑️", Bold: true})
				} else {
					w.Writeln(Message{Text: fmt.Sprintf("Force removed %s from lnk", basename), Emoji: "🗑️", Bold: true})
				}
				w.WriteString("   ").
					Writeln(Message{Text: "Tracking cleanup only — no file was restored to your home directory", Emoji: "📋"})

				return w.Err()
			}

			if err := l.Remove(filePath); err != nil {
				return err
			}

			basename := filepath.Base(filePath)
			if host != "" {
				w.Writeln(Message{Text: fmt.Sprintf("Removed %s from lnk (host: %s)", basename, host), Emoji: "🗑️", Bold: true})
			} else {
				w.Writeln(Message{Text: fmt.Sprintf("Removed %s from lnk", basename), Emoji: "🗑️", Bold: true})
			}
			w.WriteString("   ").
				Write(Message{Text: lnk.FormatManagedPath(host, filePath), Emoji: "↩️"}).
				WriteString(" → ").
				Writeln(Colored(filePath, ColorCyan))

			w.WriteString("   ").
				Writeln(Message{Text: "Original file restored", Emoji: "📄"})

			return w.Err()
		},
	}

	cmd.Flags().StringP("host", "H", "", "Remove file from specific host configuration (default: common configuration)")
	cmd.Flags().BoolP("force", "f", false, "Tracking cleanup only: drop the entry and stored file without restoring anything in your home directory")
	return cmd
}
