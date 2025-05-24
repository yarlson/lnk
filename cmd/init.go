package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "init",
		Short:        "🎯 Initialize a new lnk repository",
		Long:         "Creates the lnk directory and initializes a Git repository for managing dotfiles.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			remote, _ := cmd.Flags().GetString("remote")

			lnk := core.NewLnk()
			if err := lnk.InitWithRemote(remote); err != nil {
				return fmt.Errorf("failed to initialize lnk: %w", err)
			}

			if remote != "" {
				printf(cmd, "🎯 \033[1mInitialized lnk repository\033[0m\n")
				printf(cmd, "   📦 Cloned from: \033[36m%s\033[0m\n", remote)
				printf(cmd, "   📁 Location: \033[90m~/.config/lnk\033[0m\n")
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
	return cmd
}
