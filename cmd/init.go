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

			// Show warning when force is used and there are managed files to overwrite
			if force && remote != "" && lnk.HasUserContent() {
				printf(cmd, "⚠️  \033[33mUsing --force flag: This will overwrite existing managed files\033[0m\n")
				printf(cmd, "   💡 Only use this if you understand the risks\n\n")
			}

			if err := lnk.InitWithRemoteForce(remote, force); err != nil {
				return err
			}

			if remote != "" {
				printf(cmd, "🎯 \033[1mInitialized lnk repository\033[0m\n")
				printf(cmd, "   📦 Cloned from: \033[36m%s\033[0m\n", remote)
				printf(cmd, "   📁 Location: \033[90m~/.config/lnk\033[0m\n")

				// Try to run bootstrap script if not disabled
				if !noBootstrap {
					printf(cmd, "\n🔍 \033[1mLooking for bootstrap script...\033[0m\n")

					scriptPath, err := lnk.FindBootstrapScript()
					if err != nil {
						return err
					}

					if scriptPath != "" {
						printf(cmd, "   ✅ Found bootstrap script: \033[36m%s\033[0m\n", scriptPath)
						printf(cmd, "\n🚀 \033[1mRunning bootstrap script...\033[0m\n")
						printf(cmd, "\n")

						if err := lnk.RunBootstrapScript(scriptPath); err != nil {
							printf(cmd, "\n⚠️  \033[33mBootstrap script failed, but repository was initialized successfully\033[0m\n")
							printf(cmd, "   💡 You can run it manually with: \033[1mlnk bootstrap\033[0m\n")
							printf(cmd, "   🔧 Error: %v\n", err)
						} else {
							printf(cmd, "\n✅ \033[1;32mBootstrap completed successfully!\033[0m\n")
						}
					} else {
						printf(cmd, "   💡 No bootstrap script found\n")
					}
				}

				printf(cmd, "\n💡 \033[33mNext steps:\033[0m\n")
				printf(cmd, "   • Run \033[1mlnk pull\033[0m to restore symlinks\n")
				printf(cmd, "   • Use \033[1mlnk add <file>\033[0m to manage new files\n")
			} else {
				printf(cmd, "🎯 \033[1mInitialized empty lnk repository\033[0m\n")
				printf(cmd, "   📁 Location: \033[90m~/.config/lnk\033[0m\n")
				printf(cmd, "\n💡 \033[33mNext steps:\033[0m\n")
				printf(cmd, "   • Run \033[1mlnk add <file>\033[0m to start managing dotfiles\n")
				printf(cmd, "   • Add a remote with: \033[1mgit remote add origin <url>\033[0m\n")
			}

			return nil
		},
	}

	cmd.Flags().StringP("remote", "r", "", "Clone from remote URL instead of creating empty repository")
	cmd.Flags().Bool("no-bootstrap", false, "Skip automatic execution of bootstrap script after cloning")
	cmd.Flags().Bool("force", false, "Force initialization even if directory contains managed files (WARNING: This will overwrite existing content)")
	return cmd
}
