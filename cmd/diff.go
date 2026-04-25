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
			w := GetWriter(cmd)

			// In quiet mode, avoid materializing the patch — only validate the
			// repo and probe for changes via `git diff --quiet`.
			if w.Quiet() {
				_, err := l.HasDiff()
				return err
			}

			output, err := l.Diff(w.Colors())
			if err != nil {
				return err
			}

			if output == "" {
				w.Writeln(Success("No uncommitted changes")).
					WriteString("   ").
					Writeln(Message{Text: "Your dotfiles are clean", Emoji: "📁"})
				return w.Err()
			}

			w.WriteString(output)
			return w.Err()
		},
	}
}
