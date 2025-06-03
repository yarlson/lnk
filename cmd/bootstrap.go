package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

func newBootstrapCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "bootstrap",
		Short:         "🚀 Run the bootstrap script to set up your environment",
		Long:          "Executes the bootstrap script from your dotfiles repository to install dependencies and configure your system.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			lnk := core.NewLnk()

			scriptPath, err := lnk.FindBootstrapScript()
			if err != nil {
				return err
			}

			if scriptPath == "" {
				printf(cmd, "💡 \033[33mNo bootstrap script found\033[0m\n")
				printf(cmd, "   📝 Create a \033[1mbootstrap.sh\033[0m file in your dotfiles repository:\n")
				printf(cmd, "      \033[90m#!/bin/bash\033[0m\n")
				printf(cmd, "      \033[90mecho \"Setting up environment...\"\033[0m\n")
				printf(cmd, "      \033[90m# Your setup commands here\033[0m\n")
				return nil
			}

			printf(cmd, "🚀 \033[1mRunning bootstrap script\033[0m\n")
			printf(cmd, "   📄 Script: \033[36m%s\033[0m\n", scriptPath)
			printf(cmd, "\n")

			if err := lnk.RunBootstrapScript(scriptPath); err != nil {
				return err
			}

			printf(cmd, "\n✅ \033[1;32mBootstrap completed successfully!\033[0m\n")
			printf(cmd, "   🎉 Your environment is ready to use\n")
			return nil
		},
	}
}
