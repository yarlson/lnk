package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/lnk"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "status",
		Short:         "📊 Show repository sync status",
		Long:          "Display how many commits ahead/behind the local repository is relative to the remote and check for uncommitted changes.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			l := lnk.NewLnk()
			status, err := l.Status()
			if err != nil {
				return err
			}

			if status.Remote == "" {
				displayNoRemoteStatus(cmd, status)
				return nil
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

func displayDirtyStatus(cmd *cobra.Command, status *lnk.StatusInfo) {
	w := GetWriter(cmd)

	repoDisplay := lnk.DisplayPath(lnk.GetRepoPath())

	w.Writeln(Warning("Repository has uncommitted changes")).
		WriteString("   ").
		Write(Message{Text: "Remote: ", Emoji: "📡"}).
		Writeln(Colored(status.Remote, ColorCyan))

	if status.Ahead == 0 && status.Behind == 0 {
		w.WritelnString("").
			Write(Info("Run ")).
			Write(Bold("git add && git commit")).
			WriteString(" in ").
			Write(Colored(repoDisplay, ColorCyan)).
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
		Write(Colored(repoDisplay, ColorCyan)).
		WriteString(" or ").
		Write(Bold("lnk push")).
		WritelnString(" to commit changes")
}

func displayUpToDateStatus(cmd *cobra.Command, status *lnk.StatusInfo) {
	w := GetWriter(cmd)

	w.Writeln(Success("Repository is up to date")).
		WriteString("   ").
		Write(Message{Text: "Synced with ", Emoji: "📡"}).
		Writeln(Colored(status.Remote, ColorCyan))
}

func displaySyncStatus(cmd *cobra.Command, status *lnk.StatusInfo) {
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

func displayAheadBehindInfo(cmd *cobra.Command, status *lnk.StatusInfo, isDirty bool) {
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

// displayNoRemoteStatus renders status for a repository that has no remote
// configured. We still report local state (dirty / clean, local commit count)
// and guide the user toward adding a remote.
func displayNoRemoteStatus(cmd *cobra.Command, status *lnk.StatusInfo) {
	w := GetWriter(cmd)

	repoDisplay := lnk.DisplayPath(lnk.GetRepoPath())

	if status.Dirty {
		w.Writeln(Warning("Repository has uncommitted changes")).
			WriteString("   ").
			Writeln(Message{Text: "No remote configured", Emoji: "📡", Color: ColorGray})
	} else {
		w.Writeln(Success("Working tree is clean")).
			WriteString("   ").
			Writeln(Message{Text: "No remote configured", Emoji: "📡", Color: ColorGray})
	}

	if status.Ahead > 0 {
		commitText := getCommitText(status.Ahead)
		w.WriteString("   ").
			Writeln(Message{Text: fmt.Sprintf("%d local %s (no remote to compare against)", status.Ahead, commitText), Emoji: "⬆️", Color: ColorBrightYellow, Bold: true})
	}

	w.WritelnString("").
		Write(Info("Add a remote to enable ")).
		Write(Bold("lnk push")).
		WriteString(" / ").
		Write(Bold("lnk pull")).
		WritelnString("").
		WriteString("   ").
		Write(Bold("git remote add origin <url>")).
		WriteString(" in ").
		Writeln(Colored(repoDisplay, ColorCyan))
}
