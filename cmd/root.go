package cmd

import (
	stderrors "errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/core"
	"github.com/yarlson/lnk/internal/errors"
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

	// Handle structured errors from core package
	var lnkErr *core.LnkError
	if stderrors.As(err, &lnkErr) {
		w.Write(Error(lnkErr.Message))
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

	// Handle fs/git errors with paths
	var fileNotExistsErr *errors.FileNotExistsError
	if stderrors.As(err, &fileNotExistsErr) {
		w.Write(Error(err.Error())).
			WritelnString("").
			WriteString("   ").
			Write(Colored(fileNotExistsErr.Path, ColorRed)).
			WritelnString("")
		return
	}

	var unsupportedFileErr *errors.UnsupportedFileTypeError
	if stderrors.As(err, &unsupportedFileErr) {
		w.Write(Error(err.Error())).
			WritelnString("").
			WriteString("   ").
			Write(Colored(unsupportedFileErr.Path, ColorRed))
		if unsupportedFileErr.Suggestion != "" {
			w.WritelnString("").
				WriteString("   ").
				Write(Info(unsupportedFileErr.Suggestion))
		}
		w.WritelnString("")
		return
	}

	var notManagedErr *errors.NotManagedByLnkError
	if stderrors.As(err, &notManagedErr) {
		w.Write(Error(err.Error())).
			WritelnString("").
			WriteString("   ").
			Write(Colored(notManagedErr.Path, ColorRed))
		if notManagedErr.Suggestion != "" {
			w.WritelnString("").
				WriteString("   ").
				Write(Info(notManagedErr.Suggestion))
		}
		w.WritelnString("")
		return
	}

	var dirCreationErr *errors.DirectoryCreationError
	if stderrors.As(err, &dirCreationErr) {
		w.Write(Error(err.Error())).
			WritelnString("").
			WriteString("   ").
			Write(Colored(dirCreationErr.Path, ColorRed)).
			WritelnString("")
		return
	}

	var dirRemovalErr *errors.DirectoryRemovalError
	if stderrors.As(err, &dirRemovalErr) {
		w.Write(Error(err.Error())).
			WritelnString("").
			WriteString("   ").
			Write(Colored(dirRemovalErr.Path, ColorRed)).
			WritelnString("")
		return
	}

	// Handle git errors with remotes
	var remoteExistsErr *errors.RemoteExistsError
	if stderrors.As(err, &remoteExistsErr) {
		w.Write(Error(err.Error())).
			WritelnString("").
			WriteString("   Remote: ").
			Write(Colored(remoteExistsErr.Remote, ColorCyan)).
			WritelnString("")
		return
	}

	var remoteNotFoundErr *errors.RemoteNotFoundError
	if stderrors.As(err, &remoteNotFoundErr) {
		w.Write(Error(err.Error())).
			WritelnString("").
			WriteString("   Remote: ").
			Write(Colored(remoteNotFoundErr.Remote, ColorCyan)).
			WritelnString("")
		return
	}

	// Handle git errors with reasons
	var pushErr *errors.PushError
	if stderrors.As(err, &pushErr) {
		w.Write(Error(err.Error()))
		if pushErr.Reason != "" {
			w.WritelnString("").
				WriteString("   Reason: ").
				Write(Colored(pushErr.Reason, ColorYellow))
		}
		w.WritelnString("")
		return
	}

	var pullErr *errors.PullError
	if stderrors.As(err, &pullErr) {
		w.Write(Error(err.Error()))
		if pullErr.Reason != "" {
			w.WritelnString("").
				WriteString("   Reason: ").
				Write(Colored(pullErr.Reason, ColorYellow))
		}
		w.WritelnString("")
		return
	}

	// Handle other common errors
	var fileCheckErr *errors.FileCheckError
	if stderrors.As(err, &fileCheckErr) {
		w.Writeln(Error(err.Error()))
		return
	}

	var symlinkReadErr *errors.SymlinkReadError
	if stderrors.As(err, &symlinkReadErr) {
		w.Writeln(Error(err.Error()))
		return
	}

	var relPathErr *errors.RelativePathCalculationError
	if stderrors.As(err, &relPathErr) {
		w.Writeln(Error(err.Error()))
		return
	}

	var gitInitErr *errors.GitInitError
	if stderrors.As(err, &gitInitErr) {
		w.Writeln(Error(err.Error()))
		return
	}

	var branchSetupErr *errors.BranchSetupError
	if stderrors.As(err, &branchSetupErr) {
		w.Writeln(Error(err.Error()))
		return
	}

	var gitCmdErr *errors.GitCommandError
	if stderrors.As(err, &gitCmdErr) {
		w.Writeln(Error(err.Error()))
		return
	}

	var noRemoteErr *errors.NoRemoteError
	if stderrors.As(err, &noRemoteErr) {
		w.Writeln(Error(err.Error()))
		return
	}

	var gitConfigErr *errors.GitConfigError
	if stderrors.As(err, &gitConfigErr) {
		w.Writeln(Error(err.Error()))
		return
	}

	var uncommittedErr *errors.UncommittedChangesError
	if stderrors.As(err, &uncommittedErr) {
		w.Writeln(Error(err.Error()))
		return
	}

	// Handle generic errors
	w.Writeln(Error(err.Error()))
}
