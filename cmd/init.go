package cmd

import (
	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/core"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "init",
		Short:         "🎯 Initialize a new lnk repository",
		Long:          "Creates the lnk directory and initializes a Git repository for managing dotfiles.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			remote, _ := cmd.Flags().GetString("remote")
			noBootstrap, _ := cmd.Flags().GetBool("no-bootstrap")
			force, _ := cmd.Flags().GetBool("force")

			lnk := core.NewLnk()
			w := GetWriter(cmd)

			// Show warning when force is used and there are managed files to overwrite
			if force && remote != "" && lnk.HasUserContent() {
				w.Writeln(Warning("Using --force flag: This will overwrite existing managed files")).
					WriteString("   ").
					Writeln(Info("Only use this if you understand the risks")).
					WritelnString("")
				if err := w.Err(); err != nil {
					return err
				}
			}

			if err := lnk.InitWithRemoteForce(remote, force); err != nil {
				return err
			}

			if remote != "" {
				w.Writeln(Target("Initialized lnk repository")).
					WriteString("   ").
					Write(Message{Text: "Cloned from: ", Emoji: "📦"}).
					Writeln(Colored(remote, ColorCyan)).
					WriteString("   ").
					Write(Message{Text: "Location: ", Emoji: "📁"}).
					Writeln(Colored("~/.config/lnk", ColorGray))

				if err := w.Err(); err != nil {
					return err
				}

				// Try to run bootstrap script if not disabled
				if !noBootstrap {
					w.WritelnString("").
						Writeln(Message{Text: "Looking for bootstrap script...", Emoji: "🔍", Bold: true})

					if err := w.Err(); err != nil {
						return err
					}

					scriptPath, err := lnk.FindBootstrapScript()
					if err != nil {
						return err
					}

					if scriptPath != "" {
						w.WriteString("   ").
							Write(Success("Found bootstrap script: ")).
							Writeln(Colored(scriptPath, ColorCyan)).
							WritelnString("").
							Writeln(Rocket("Running bootstrap script...")).
							WritelnString("")

						if err := w.Err(); err != nil {
							return err
						}

						if err := lnk.RunBootstrapScript(scriptPath); err != nil {
							w.WritelnString("").
								Writeln(Warning("Bootstrap script failed, but repository was initialized successfully")).
								WriteString("   ").
								Write(Info("You can run it manually with: ")).
								Writeln(Bold("lnk bootstrap")).
								WriteString("   ").
								Write(Message{Text: "Error: ", Emoji: "🔧"}).
								Writeln(Plain(err.Error()))
						} else {
							w.WritelnString("").
								Writeln(Success("Bootstrap completed successfully!"))
						}

						if err := w.Err(); err != nil {
							return err
						}
					} else {
						w.WriteString("   ").
							Writeln(Info("No bootstrap script found"))
						if err := w.Err(); err != nil {
							return err
						}
					}
				}

				w.WritelnString("").
					Writeln(Info("Next steps:")).
					WriteString("   • Run ").
					Write(Bold("lnk pull")).
					Writeln(Plain(" to restore symlinks")).
					WriteString("   • Use ").
					Write(Bold("lnk add <file>")).
					Writeln(Plain(" to manage new files"))

				return w.Err()
			} else {
				w.Writeln(Target("Initialized empty lnk repository")).
					WriteString("   ").
					Write(Message{Text: "Location: ", Emoji: "📁"}).
					Writeln(Colored("~/.config/lnk", ColorGray)).
					WritelnString("").
					Writeln(Info("Next steps:")).
					WriteString("   • Run ").
					Write(Bold("lnk add <file>")).
					Writeln(Plain(" to start managing dotfiles")).
					WriteString("   • Add a remote with: ").
					Writeln(Bold("git remote add origin <url>"))

				return w.Err()
			}
		},
	}

	cmd.Flags().StringP("remote", "r", "", "Clone from remote URL instead of creating empty repository")
	cmd.Flags().Bool("no-bootstrap", false, "Skip automatic execution of bootstrap script after cloning")
	cmd.Flags().Bool("force", false, "Force initialization even if directory contains managed files (WARNING: This will overwrite existing content)")
	return cmd
}
