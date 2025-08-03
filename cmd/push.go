package cmd

import (
	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/core"
)

func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "push [message]",
		Short:         "🚀 Push local changes to remote repository",
		Long:          "Stages all changes, creates a sync commit with the provided message, and pushes to remote.",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			message := "lnk: sync configuration files"
			if len(args) > 0 {
				message = args[0]
			}

			lnk := core.NewLnk()
			w := GetWriter(cmd)

			if err := lnk.Push(message); err != nil {
				return err
			}

			w.Writeln(Rocket("Successfully pushed changes")).
				WriteString("   ").
				Write(Message{Text: "Commit: ", Emoji: "💾"}).
				Writeln(Colored(message, ColorGray)).
				WriteString("   ").
				Writeln(Message{Text: "Synced to remote", Emoji: "📡"}).
				WriteString("   ").
				Writeln(Sparkles("Your dotfiles are up to date!"))

			return w.Err()
		},
	}
}
