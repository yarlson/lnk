package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/core"
)

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm <file>",
		Short: "🗑️ Remove a file from lnk management",
		Long: `Removes a symlink and restores the original file from the lnk repository.

Use --force to remove a file from tracking even if the symlink no longer exists
(e.g., if you accidentally deleted the symlink without using lnk rm).`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			host, _ := cmd.Flags().GetString("host")
			force, _ := cmd.Flags().GetBool("force")
			lnk := core.NewLnk(core.WithHost(host))
			w := GetWriter(cmd)

			if force {
				if err := lnk.RemoveForce(filePath); err != nil {
					return err
				}

				basename := filepath.Base(filePath)
				if host != "" {
					w.Writeln(Message{Text: fmt.Sprintf("Force removed %s from lnk (host: %s)", basename, host), Emoji: "🗑️", Bold: true})
				} else {
					w.Writeln(Message{Text: fmt.Sprintf("Force removed %s from lnk", basename), Emoji: "🗑️", Bold: true})
				}
				w.WriteString("   ").
					Writeln(Message{Text: "File removed from tracking", Emoji: "📋"})

				return w.Err()
			}

			if err := lnk.Remove(filePath); err != nil {
				return err
			}

			basename := filepath.Base(filePath)
			if host != "" {
				w.Writeln(Message{Text: fmt.Sprintf("Removed %s from lnk (host: %s)", basename, host), Emoji: "🗑️", Bold: true}).
					WriteString("   ").
					Write(Message{Text: fmt.Sprintf("~/.config/lnk/%s.lnk/%s", host, basename), Emoji: "↩️"}).
					WriteString(" → ").
					Writeln(Colored(filePath, ColorCyan))
			} else {
				w.Writeln(Message{Text: fmt.Sprintf("Removed %s from lnk", basename), Emoji: "🗑️", Bold: true}).
					WriteString("   ").
					Write(Message{Text: fmt.Sprintf("~/.config/lnk/%s", basename), Emoji: "↩️"}).
					WriteString(" → ").
					Writeln(Colored(filePath, ColorCyan))
			}

			w.WriteString("   ").
				Writeln(Message{Text: "Original file restored", Emoji: "📄"})

			return w.Err()
		},
	}

	cmd.Flags().StringP("host", "H", "", "Remove file from specific host configuration (default: common configuration)")
	cmd.Flags().BoolP("force", "f", false, "Force removal from tracking even if symlink is missing")
	return cmd
}
