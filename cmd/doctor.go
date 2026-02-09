package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/core"
)

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "🩺 Diagnose and fix repository health issues",
		Long: `Scans the lnk repository for health issues and fixes them.

Checks performed:
  • Invalid entries: .lnk entries whose stored files no longer exist
  • Broken symlinks: managed files whose symlinks are missing or broken

Use --host to check a specific host configuration instead of the common one.
Use --dry-run to preview what would be fixed without making changes.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			lnk := core.NewLnk(core.WithHost(host))
			w := GetWriter(cmd)

			// Handle dry-run mode
			if dryRun {
				result, err := lnk.PreviewDoctor()
				if err != nil {
					return err
				}

				if !result.HasIssues() {
					if host != "" {
						w.Writeln(Success(fmt.Sprintf("Repository is healthy (host: %s)", host)))
					} else {
						w.Writeln(Success("Repository is healthy"))
					}
					w.WriteString("   ").
						Writeln(Message{Text: "No issues found", Emoji: "📋"})
					return w.Err()
				}

				// Show summary
				hostSuffix := ""
				if host != "" {
					hostSuffix = fmt.Sprintf(" (host: %s)", host)
				}
				w.Writeln(Message{Text: fmt.Sprintf("Found %d issue%s%s:", result.TotalIssues(), pluralS(result.TotalIssues()), hostSuffix), Emoji: "🔍", Bold: true})

				// Show broken symlinks
				if len(result.BrokenSymlinks) > 0 {
					w.WritelnString("")
					w.WriteString("   ").
						Writeln(Message{Text: fmt.Sprintf("Would fix %d broken symlink%s:", len(result.BrokenSymlinks), pluralS(len(result.BrokenSymlinks))), Emoji: "🔗", Bold: true})
					for _, entry := range result.BrokenSymlinks {
						w.WriteString("      ").
							Writeln(Message{Text: entry, Color: ColorYellow, Emoji: "🔗"})
					}
				}

				// Show invalid entries
				if len(result.InvalidEntries) > 0 {
					w.WritelnString("")
					w.WriteString("   ").
						Writeln(Message{Text: fmt.Sprintf("Would remove %d invalid entr%s:", len(result.InvalidEntries), pluralY(len(result.InvalidEntries))), Emoji: "🗑️", Bold: true})
					for _, entry := range result.InvalidEntries {
						w.WriteString("      ").
							Writeln(Message{Text: entry, Color: ColorRed, Emoji: "🗑️"})
					}
				}

				w.WritelnString("").
					Writeln(Info("To proceed: run without --dry-run flag"))

				return w.Err()
			}

			result, err := lnk.Doctor()
			if err != nil {
				return err
			}

			if !result.HasIssues() {
				if host != "" {
					w.Writeln(Success(fmt.Sprintf("Repository is healthy (host: %s)", host)))
				} else {
					w.Writeln(Success("Repository is healthy"))
				}
				w.WriteString("   ").
					Writeln(Message{Text: "No issues found", Emoji: "📋"})
				return w.Err()
			}

			// Show summary
			hostSuffix := ""
			if host != "" {
				hostSuffix = fmt.Sprintf(" (host: %s)", host)
			}
			w.Writeln(Message{Text: fmt.Sprintf("Fixed %d issue%s%s", result.TotalIssues(), pluralS(result.TotalIssues()), hostSuffix), Emoji: "🩺", Bold: true})

			// Show fixed broken symlinks
			if len(result.BrokenSymlinks) > 0 {
				w.WritelnString("")
				w.WriteString("   ").
					Writeln(Message{Text: fmt.Sprintf("Restored %d broken symlink%s:", len(result.BrokenSymlinks), pluralS(len(result.BrokenSymlinks))), Emoji: "🔗", Bold: true})
				for _, entry := range result.BrokenSymlinks {
					w.WriteString("      ").
						Writeln(Message{Text: entry, Color: ColorCyan, Emoji: "🔗"})
				}
			}

			// Show removed invalid entries
			if len(result.InvalidEntries) > 0 {
				w.WritelnString("")
				w.WriteString("   ").
					Writeln(Message{Text: fmt.Sprintf("Removed %d invalid entr%s:", len(result.InvalidEntries), pluralY(len(result.InvalidEntries))), Emoji: "🗑️", Bold: true})
				for _, entry := range result.InvalidEntries {
					w.WriteString("      ").
						Writeln(Message{Text: entry, Color: ColorRed, Emoji: "🗑️"})
				}
			}

			w.WritelnString("").
				Write(Info("Use ")).
				Write(Bold("lnk push")).
				WritelnString(" to sync changes to remote")

			return w.Err()
		},
	}

	cmd.Flags().StringP("host", "H", "", "Check specific host configuration (default: common configuration)")
	cmd.Flags().BoolP("dry-run", "n", false, "Show what would be fixed without making changes")
	return cmd
}

// pluralS returns "s" for counts != 1, "" for count == 1.
func pluralS(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// pluralY returns "y" for count == 1, "ies" for count != 1.
func pluralY(count int) string {
	if count == 1 {
		return "y"
	}
	return "ies"
}
