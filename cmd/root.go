// Package cmd implements the CLI commands for lnk.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/lnkerr"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

// NewRootCommand creates a new root command (testable)
func NewRootCommand() *cobra.Command {
	var (
		colors  string
		emoji   bool
		noEmoji bool
	)

	rootCmd := &cobra.Command{
		Use:   "lnk",
		Short: "🔗 Dotfiles, linked. No fluff.",
		Long: `🔗 Lnk - Git-native dotfiles management that doesn't suck.

Move your dotfiles to ~/.config/lnk, symlink them back, and use Git like normal.
Supports both common configurations, host-specific setups, and bulk operations for multiple files.

✨ Examples:
  lnk init                           # Fresh start
  lnk init -r <repo-url>             # Clone existing dotfiles (runs bootstrap automatically)
  lnk add ~/.vimrc ~/.bashrc         # Start managing common files
  lnk add --recursive ~/.config/nvim # Add directory contents individually
  lnk add --dry-run ~/.gitconfig     # Preview changes without applying
  lnk add --host work ~/.ssh/config  # Manage host-specific files
  lnk list --all                     # Show all configurations
  lnk pull --host work               # Pull host-specific changes
  lnk push "setup complete"          # Sync to remote
  lnk bootstrap                      # Run bootstrap script manually

🚀 Bootstrap Support:
  Automatically runs bootstrap.sh when cloning a repository.
  Use --no-bootstrap to disable.

🎯 Simple, fast, Git-native, and multi-host ready.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       fmt.Sprintf("%s (built %s)", version, buildTime),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Handle emoji flag logic
			emojiEnabled := emoji
			if noEmoji {
				emojiEnabled = false
			}
			err := SetGlobalConfig(colors, emojiEnabled)
			if err != nil {
				return err
			}

			return nil
		},
	}

	// Add global flags for output formatting
	rootCmd.PersistentFlags().StringVar(&colors, "colors", "auto", "when to use colors (auto, always, never)")
	rootCmd.PersistentFlags().BoolVar(&emoji, "emoji", true, "enable emoji in output")
	rootCmd.PersistentFlags().BoolVar(&noEmoji, "no-emoji", false, "disable emoji in output")

	// Mark emoji flags as mutually exclusive
	rootCmd.MarkFlagsMutuallyExclusive("emoji", "no-emoji")

	// Add subcommands
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newAddCmd())
	rootCmd.AddCommand(newRemoveCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newDiffCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newPushCmd())
	rootCmd.AddCommand(newPullCmd())
	rootCmd.AddCommand(newBootstrapCmd())

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
		DisplayError(err)
		os.Exit(1)
	}
}

// DisplayError formats and displays an error with appropriate styling
func DisplayError(err error) {
	w := GetErrorWriter()

	var lnkErr *lnkerr.Error
	if errors.As(err, &lnkErr) {
		w.Write(Error(lnkErr.Err.Error()))
		if lnkErr.Path != "" {
			w.WritelnString("").
				WriteString("   ").
				Write(Colored(lnkErr.Path, ColorRed))
		}
		if lnkErr.Suggestion != "" {
			w.WritelnString("").
				WriteString("   ").
				Write(Info(lnkErr.Suggestion))
		}
		w.WritelnString("")
		return
	}

	w.Writeln(Error(err.Error()))
}
