package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/lnk"
)

// bootstrapWriters returns the stdout/stderr targets to hand to the
// bootstrap script: the cobra command's writers in normal mode, or
// io.Discard in both slots when --quiet is set.
func bootstrapWriters(cmd *cobra.Command, w *Writer) (io.Writer, io.Writer) {
	if w.Quiet() {
		return io.Discard, io.Discard
	}
	return cmd.OutOrStdout(), cmd.ErrOrStderr()
}

func newBootstrapCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "bootstrap",
		Short:         "🚀 Run the bootstrap script to set up your environment",
		Long:          "Executes the bootstrap script from your dotfiles repository to install dependencies and configure your system.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			lnk := lnk.NewLnk()
			w := GetWriter(cmd)

			scriptPath, err := lnk.FindBootstrapScript()
			if err != nil {
				return err
			}

			if scriptPath == "" {
				w.Writeln(Info("No bootstrap script found")).
					WriteString("   ").
					Write(Message{Text: "Create a ", Emoji: "📝"}).
					Write(Bold("bootstrap.sh")).
					WritelnString(" file in your dotfiles repository:").
					WriteString("      ").
					Writeln(Colored("#!/bin/bash", ColorGray)).
					WriteString("      ").
					Writeln(Colored("echo \"Setting up environment...\"", ColorGray)).
					WriteString("      ").
					Writeln(Colored("# Your setup commands here", ColorGray))
				return w.Err()
			}

			w.Writeln(Rocket("Running bootstrap script")).
				WriteString("   ").
				Write(Message{Text: "Script: ", Emoji: "📄"}).
				Writeln(Colored(scriptPath, ColorCyan)).
				WritelnString("")

			if err := w.Err(); err != nil {
				return err
			}

			scriptOut, scriptErr := bootstrapWriters(cmd, w)
			if err := lnk.RunBootstrapScript(scriptPath, scriptOut, scriptErr, os.Stdin); err != nil {
				return err
			}

			w.WritelnString("").
				Writeln(Success("Bootstrap completed successfully!")).
				WriteString("   ").
				Writeln(Message{Text: "Your environment is ready to use", Emoji: "🎉"})

			return w.Err()
		},
	}
}
