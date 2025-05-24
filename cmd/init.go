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
		lnk := core.NewLnk()
		if err := lnk.Init(); err != nil {
			return fmt.Errorf("failed to initialize lnk: %w", err)
		}
		fmt.Println("Initialized lnk repository")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
