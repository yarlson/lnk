package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "lnk",
	Short: "Dotfiles, linked. No fluff.",
	Long:  "Lnk is a minimalist CLI tool for managing dotfiles using symlinks and Git.",
}

// SetVersion sets the version information for the CLI
func SetVersion(v, bt string) {
	version = v
	buildTime = bt
	rootCmd.Version = fmt.Sprintf("%s (built %s)", version, buildTime)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
