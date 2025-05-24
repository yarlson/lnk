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
	Short: "🔗 Dotfiles, linked. No fluff.",
	Long: `🔗 Lnk - Git-native dotfiles management that doesn't suck.

Move your dotfiles to ~/.config/lnk, symlink them back, and use Git like normal.
That's it.

✨ Examples:
  lnk init                    # Fresh start
  lnk init -r <repo-url>      # Clone existing dotfiles
  lnk add ~/.vimrc ~/.bashrc  # Start managing files
  lnk push "setup complete"   # Sync to remote
  lnk pull                    # Get latest changes

🎯 Simple, fast, and Git-native.`,
	SilenceUsage: true,
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
