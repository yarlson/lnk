package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/lnk"
)

func newPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "pull",
		Short:         "⬇️ Pull changes from remote and restore symlinks",
		Long:          "Fetches changes from remote repository and automatically restores symlinks for all managed files.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			lnk := lnk.NewLnk(lnk.WithHost(host))
			w := GetWriter(cmd)

			result, err := lnk.Pull()
			if err != nil {
				return err
			}

			var successMsg string
			if host != "" {
				successMsg = fmt.Sprintf("Successfully pulled changes (host: %s)", host)
			} else {
				successMsg = "Successfully pulled changes"
			}

			if len(result.Restored) > 0 {
				symlinkText := fmt.Sprintf("Restored %d symlink", len(result.Restored))
				if len(result.Restored) > 1 {
					symlinkText += "s"
				}
				symlinkText += ":"

				w.Writeln(Message{Text: successMsg, Emoji: "⬇️", Color: ColorBrightGreen, Bold: true}).
					WriteString("   ").
					Writeln(Link(symlinkText))

				for _, file := range result.Restored {
					w.WriteString("      ").
						Writeln(Sparkles(file))
				}

				writeBackupNotice(w, result.BackedUp)

				w.WritelnString("").
					WriteString("   ").
					Writeln(Message{Text: "Your dotfiles are synced and ready!", Emoji: "🎉"})
			} else {
				w.Writeln(Message{Text: successMsg, Emoji: "⬇️", Color: ColorBrightGreen, Bold: true}).
					WriteString("   ").
					Writeln(Success("All symlinks already in place")).
					WriteString("   ").
					Writeln(Message{Text: "Everything is up to date!", Emoji: "🎉"})
			}

			return w.Err()
		},
	}

	cmd.Flags().StringP("host", "H", "", "Pull and restore symlinks for specific host (default: common configuration)")
	return cmd
}

// writeBackupNotice renders a section listing files that were renamed to
// <path>.lnk-backup so the user can decide what to do with them. No-op when
// no backups occurred.
func writeBackupNotice(w *Writer, backedUp []string) {
	if len(backedUp) == 0 {
		return
	}

	noun := "file"
	if len(backedUp) > 1 {
		noun = "files"
	}

	w.WritelnString("").
		WriteString("   ").
		Writeln(Warning(fmt.Sprintf("Backed up %d existing %s to .lnk-backup:", len(backedUp), noun)))

	for _, file := range backedUp {
		w.WriteString("      ").
			Write(Plain("~/" + file)).
			WriteString(" → ").
			Writeln(Colored("~/"+file+".lnk-backup", ColorYellow))
	}
}
