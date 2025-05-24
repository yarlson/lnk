package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// printf is a helper function to simplify output formatting in commands
func printf(cmd *cobra.Command, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), format, args...)
}
