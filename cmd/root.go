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

// NewRootCommand creates a new root command (testable)
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "lnk",
		Short: "🔗 Dotfiles, linked. No fluff.",
		Long: `🔗 Lnk - Git-native dotfiles management that doesn't suck.

Move your dotfiles to ~/.config/lnk, symlink them back, and use Git like normal.
Supports both common configurations and host-specific setups.

✨ Examples:
  lnk init                         # Fresh start
  lnk init -r <repo-url>           # Clone existing dotfiles
  lnk add ~/.vimrc ~/.bashrc       # Start managing common files
  lnk add --host work ~/.ssh/config # Manage host-specific files
  lnk list --all                  # Show all configurations
  lnk pull --host work             # Pull host-specific changes
  lnk push "setup complete"        # Sync to remote

🎯 Simple, fast, Git-native, and multi-host ready.`,
		SilenceUsage: true,
		Version:      fmt.Sprintf("%s (built %s)", version, buildTime),
	}

	// Add subcommands
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newAddCmd())
	rootCmd.AddCommand(newRemoveCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newPushCmd())
	rootCmd.AddCommand(newPullCmd())

	return rootCmd
}

// SetVersion sets the version information for the CLI
func SetVersion(v, bt string) {
	version = v
	buildTime = bt
}

func Execute() {
	rootCmd := NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
