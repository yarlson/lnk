package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yarlson/lnk/internal/core"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "list",
		Short:         "📋 List files managed by lnk",
		Long:          "Display all files and directories currently managed by lnk.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			all, _ := cmd.Flags().GetBool("all")

			if host != "" {
				// Show specific host configuration
				return listHostConfig(cmd, host)
			}

			if all {
				// Show all configurations (common + all hosts)
				return listAllConfigs(cmd)
			}

			// Default: show common configuration
			return listCommonConfig(cmd)
		},
	}

	cmd.Flags().StringP("host", "H", "", "List files for specific host")
	cmd.Flags().BoolP("all", "a", false, "List files for all hosts and common configuration")
	return cmd
}

func listCommonConfig(cmd *cobra.Command) error {
	lnk := core.NewLnk()
	w := GetWriter(cmd)

	managedItems, err := lnk.List()
	if err != nil {
		return err
	}

	if len(managedItems) == 0 {
		w.Writeln(Message{Text: "No files currently managed by lnk (common)", Emoji: "📋", Bold: true}).
			WriteString("   ").
			Write(Info("Use ")).
			Write(Bold("lnk add <file>")).
			WritelnString(" to start managing files")
		return w.Err()
	}

	countText := fmt.Sprintf("Files managed by lnk (common) (%d item", len(managedItems))
	if len(managedItems) > 1 {
		countText += "s"
	}
	countText += "):"

	w.Writeln(Message{Text: countText, Emoji: "📋", Bold: true}).
		WritelnString("")

	for _, item := range managedItems {
		w.WriteString("   ").
			Writeln(Link(item))
	}

	w.WritelnString("").
		Write(Info("Use ")).
		Write(Bold("lnk status")).
		WritelnString(" to check sync status")
	return w.Err()
}

func listHostConfig(cmd *cobra.Command, host string) error {
	lnk := core.NewLnk(core.WithHost(host))
	w := GetWriter(cmd)

	managedItems, err := lnk.List()
	if err != nil {
		return err
	}

	if len(managedItems) == 0 {
		w.Writeln(Message{Text: fmt.Sprintf("No files currently managed by lnk (host: %s)", host), Emoji: "📋", Bold: true}).
			WriteString("   ").
			Write(Info("Use ")).
			Write(Bold(fmt.Sprintf("lnk add --host %s <file>", host))).
			WritelnString(" to start managing files")
		return w.Err()
	}

	countText := fmt.Sprintf("Files managed by lnk (host: %s) (%d item", host, len(managedItems))
	if len(managedItems) > 1 {
		countText += "s"
	}
	countText += "):"

	w.Writeln(Message{Text: countText, Emoji: "📋", Bold: true}).
		WritelnString("")

	for _, item := range managedItems {
		w.WriteString("   ").
			Writeln(Link(item))
	}

	w.WritelnString("").
		Write(Info("Use ")).
		Write(Bold("lnk status")).
		WritelnString(" to check sync status")
	return w.Err()
}

func listAllConfigs(cmd *cobra.Command) error {
	w := GetWriter(cmd)

	// List common configuration
	w.Writeln(Message{Text: "All configurations managed by lnk", Emoji: "📋", Bold: true}).
		WritelnString("")

	lnk := core.NewLnk()
	commonItems, err := lnk.List()
	if err != nil {
		return err
	}

	countText := fmt.Sprintf("Common configuration (%d item", len(commonItems))
	if len(commonItems) > 1 {
		countText += "s"
	}
	countText += "):"

	w.Writeln(Message{Text: countText, Emoji: "🌐", Bold: true})

	if len(commonItems) == 0 {
		w.WriteString("   ").
			Writeln(Colored("(no files)", ColorGray))
	} else {
		for _, item := range commonItems {
			w.WriteString("   ").
				Writeln(Link(item))
		}
	}

	// Find all host-specific configurations
	hosts, err := findHostConfigs()
	if err != nil {
		return err
	}

	for _, host := range hosts {
		w.WritelnString("").
			Write(Message{Text: fmt.Sprintf("Host: %s", host), Emoji: "🖥️", Bold: true})

		hostLnk := core.NewLnk(core.WithHost(host))
		hostItems, err := hostLnk.List()
		if err != nil {
			w.WriteString(" ").
				Writeln(Colored(fmt.Sprintf("(error: %v)", err), ColorRed))
			continue
		}

		countText := fmt.Sprintf(" (%d item", len(hostItems))
		if len(hostItems) > 1 {
			countText += "s"
		}
		countText += "):"

		w.WriteString(countText).
			WritelnString("")

		if len(hostItems) == 0 {
			w.WriteString("   ").
				Writeln(Colored("(no files)", ColorGray))
		} else {
			for _, item := range hostItems {
				w.WriteString("   ").
					Writeln(Link(item))
			}
		}
	}

	w.WritelnString("").
		Write(Info("Use ")).
		Write(Bold("lnk list --host <hostname>")).
		WritelnString(" to see specific host configuration")
	return w.Err()
}

func findHostConfigs() ([]string, error) {
	repoPath := core.GetRepoPath()

	// Check if repo exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return nil, err
	}

	var hosts []string
	for _, entry := range entries {
		name := entry.Name()
		// Look for .lnk.<hostname> files
		if strings.HasPrefix(name, ".lnk.") && name != ".lnk" {
			host := strings.TrimPrefix(name, ".lnk.")
			hosts = append(hosts, host)
		}
	}

	return hosts, nil
}
