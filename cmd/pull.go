package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull changes from remote and restore symlinks",
	Long:  "Fetches changes from remote repository and automatically restores symlinks for all managed files.",
	RunE: func(cmd *cobra.Command, args []string) error {
		lnk := core.NewLnk()
		restored, err := lnk.Pull()
		if err != nil {
			return fmt.Errorf("failed to pull changes: %w", err)
		}

		if len(restored) > 0 {
			fmt.Printf("Successfully pulled changes and restored %d symlink(s):\n", len(restored))
			for _, file := range restored {
				fmt.Printf("  - %s\n", file)
			}
		} else {
			fmt.Println("Successfully pulled changes (no symlinks needed restoration)")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
}
