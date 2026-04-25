package cmd

import (
	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/lnk"
)

func newDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "diff",
		Short:         "📝 Show uncommitted changes in the repository",
		Long:          "Displays a diff of uncommitted changes in the lnk repository, equivalent to running git diff inside the lnk repo.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			l := lnk.NewLnk()

			// Determine color mode based on terminal detection
			useColor := isTerminal()

			output, err := l.Diff(useColor)
			if err != nil {
				return err
			}

			w := GetWriter(cmd)

			if output == "" {
				w.Writeln(Success("No uncommitted changes")).
					WriteString("   ").
					Writeln(Message{Text: "Your dotfiles are clean", Emoji: "📁"})
				return w.Err()
			}

			// Write diff output directly to command's stdout
			cmd.Print(output)
			return nil
		},
	}
}
