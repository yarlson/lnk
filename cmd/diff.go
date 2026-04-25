package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/lnk"
)

// errDiffHasChanges is returned by `lnk diff --quiet` when the repo has
// uncommitted changes. It exists only to drive a non-zero exit code;
// quiet mode suppresses error display, so callers see only the exit code.
var errDiffHasChanges = errors.New("repository has uncommitted changes")

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

			// In quiet mode, avoid materializing the patch — probe for changes
			// via `git diff --quiet` and signal dirty state through the exit
			// code (errDiffHasChanges → exit 1).
			if w.Quiet() {
				dirty, err := l.HasDiff()
				if err != nil {
					return err
				}
				if dirty {
					return errDiffHasChanges
				}
				return nil
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
