package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lnk",
	Short: "Dotfiles, linked. No fluff.",
	Long:  "Lnk is a minimalist CLI tool for managing dotfiles using symlinks and Git.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
