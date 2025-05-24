package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yarlson/lnk/internal/core"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new lnk repository",
	Long:  "Creates the lnk directory and initializes a Git repository for managing dotfiles.",
	RunE: func(cmd *cobra.Command, args []string) error {
		remote, _ := cmd.Flags().GetString("remote")

		lnk := core.NewLnk()
		if err := lnk.Init(); err != nil {
			return fmt.Errorf("failed to initialize lnk: %w", err)
		}

		if remote != "" {
			if err := lnk.AddRemote("origin", remote); err != nil {
				return fmt.Errorf("failed to add remote: %w", err)
			}
			fmt.Printf("Initialized lnk repository with remote: %s\n", remote)
		} else {
			fmt.Println("Initialized lnk repository")
		}

		return nil
	},
}

func init() {
	initCmd.Flags().StringP("remote", "r", "", "Add origin remote URL to the repository")
	rootCmd.AddCommand(initCmd)
}
