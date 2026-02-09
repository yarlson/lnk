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

			restored, err := lnk.Pull()
			if err != nil {
				return err
			}

			if len(restored) > 0 {
				var successMsg string
				if host != "" {
					successMsg = fmt.Sprintf("Successfully pulled changes (host: %s)", host)
				} else {
					successMsg = "Successfully pulled changes"
				}

				symlinkText := fmt.Sprintf("Restored %d symlink", len(restored))
				if len(restored) > 1 {
					symlinkText += "s"
				}
				symlinkText += ":"

				w.Writeln(Message{Text: successMsg, Emoji: "⬇️", Color: ColorBrightGreen, Bold: true}).
					WriteString("   ").
					Writeln(Link(symlinkText))

				for _, file := range restored {
					w.WriteString("      ").
						Writeln(Sparkles(file))
				}

				w.WritelnString("").
					WriteString("   ").
					Writeln(Message{Text: "Your dotfiles are synced and ready!", Emoji: "🎉"})
			} else {
				var successMsg string
				if host != "" {
					successMsg = fmt.Sprintf("Successfully pulled changes (host: %s)", host)
				} else {
					successMsg = "Successfully pulled changes"
				}

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
