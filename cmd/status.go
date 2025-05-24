package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show repository sync status",
	Long:  "Display how many commits ahead/behind the local repository is relative to the remote.",
	RunE: func(cmd *cobra.Command, args []string) error {
		lnk := core.NewLnk()
		status, err := lnk.Status()
		if err != nil {
			return fmt.Errorf("failed to get status: %w", err)
		}

		if status.Ahead == 0 && status.Behind == 0 {
			fmt.Println("Repository is up to date with remote")
		} else {
			if status.Ahead > 0 {
				fmt.Printf("Your branch is ahead of '%s' by %d commit(s)\n", status.Remote, status.Ahead)
			}
			if status.Behind > 0 {
				fmt.Printf("Your branch is behind '%s' by %d commit(s)\n", status.Remote, status.Behind)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
