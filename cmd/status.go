package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/core"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "status",
		Short:         "📊 Show repository sync status",
		Long:          "Display how many commits ahead/behind the local repository is relative to the remote and check for uncommitted changes.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			lnk := core.NewLnk()
			status, err := lnk.Status()
			if err != nil {
				return err
			}

			if status.Dirty {
				displayDirtyStatus(cmd, status)
				return nil
			}

			if status.Ahead == 0 && status.Behind == 0 {
				displayUpToDateStatus(cmd, status)
				return nil
			}

			displaySyncStatus(cmd, status)
			return nil
		},
	}
}

func displayDirtyStatus(cmd *cobra.Command, status *core.StatusInfo) {
	w := GetWriter(cmd)

	w.Writeln(Warning("Repository has uncommitted changes")).
		WriteString("   ").
		Write(Message{Text: "Remote: ", Emoji: "📡"}).
		Writeln(Colored(status.Remote, ColorCyan))

	if status.Ahead == 0 && status.Behind == 0 {
		w.WritelnString("").
			Write(Info("Run ")).
			Write(Bold("git add && git commit")).
			WriteString(" in ").
			Write(Colored("~/.config/lnk", ColorCyan)).
			WriteString(" or ").
			Write(Bold("lnk push")).
			WritelnString(" to commit changes")
		return
	}

	w.WritelnString("")
	displayAheadBehindInfo(cmd, status, true)
	w.WritelnString("").
		Write(Info("Run ")).
		Write(Bold("git add && git commit")).
		WriteString(" in ").
		Write(Colored("~/.config/lnk", ColorCyan)).
		WriteString(" or ").
		Write(Bold("lnk push")).
		WritelnString(" to commit changes")
}

func displayUpToDateStatus(cmd *cobra.Command, status *core.StatusInfo) {
	w := GetWriter(cmd)

	w.Writeln(Success("Repository is up to date")).
		WriteString("   ").
		Write(Message{Text: "Synced with ", Emoji: "📡"}).
		Writeln(Colored(status.Remote, ColorCyan))
}

func displaySyncStatus(cmd *cobra.Command, status *core.StatusInfo) {
	w := GetWriter(cmd)

	w.Writeln(Message{Text: "Repository Status", Emoji: "📊", Bold: true}).
		WriteString("   ").
		Write(Message{Text: "Remote: ", Emoji: "📡"}).
		Writeln(Colored(status.Remote, ColorCyan)).
		WritelnString("")

	displayAheadBehindInfo(cmd, status, false)

	if status.Ahead > 0 && status.Behind == 0 {
		w.WritelnString("").
			Write(Info("Run ")).
			Write(Bold("lnk push")).
			WritelnString(" to sync your changes")
	} else if status.Behind > 0 {
		w.WritelnString("").
			Write(Info("Run ")).
			Write(Bold("lnk pull")).
			WritelnString(" to get latest changes")
	}
}

func displayAheadBehindInfo(cmd *cobra.Command, status *core.StatusInfo, isDirty bool) {
	w := GetWriter(cmd)

	if status.Ahead > 0 {
		commitText := getCommitText(status.Ahead)
		if isDirty {
			w.WriteString("   ").
				Write(Message{Text: fmt.Sprintf("%d %s ahead", status.Ahead, commitText), Emoji: "⬆️", Color: ColorBrightYellow, Bold: true}).
				WritelnString(" (excluding uncommitted changes)")
		} else {
			w.WriteString("   ").
				Write(Message{Text: fmt.Sprintf("%d %s ahead", status.Ahead, commitText), Emoji: "⬆️", Color: ColorBrightYellow, Bold: true}).
				WritelnString(" - ready to push")
		}
	}

	if status.Behind > 0 {
		commitText := getCommitText(status.Behind)
		w.WriteString("   ").
			Write(Message{Text: fmt.Sprintf("%d %s behind", status.Behind, commitText), Emoji: "⬇️", Color: ColorBrightRed, Bold: true}).
			WriteString(" - run ").
			Write(Bold("lnk pull")).
			WritelnString("")
	}
}

func getCommitText(count int) string {
	if count == 1 {
		return "commit"
	}
	return "commits"
}
