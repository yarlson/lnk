package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/lnk"
)

func newHomeCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "home",
		Short:         "🏠 Print your lnk home path",
		Long:          "Print the lnk repository directory (LNK_HOME, XDG_CONFIG_HOME/lnk, or ~/.config/lnk).",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoPath := lnk.GetRepoPath()
			if err := ensureLnkHome(repoPath); err != nil {
				return err
			}

			GetWriter(cmd).Writeln(Plain(repoPath))
			return nil
		},
	}
}

func ensureLnkHome(repoPath string) error {
	info, err := os.Stat(repoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("lnk home does not exist: %s (run lnk init first)", repoPath)
		}
		return fmt.Errorf("failed to access lnk home: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("lnk home is not a directory: %s", repoPath)
	}
	return nil
}
