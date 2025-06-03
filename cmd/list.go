package cmd

import (
	"os"
	"path/filepath"
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
	managedItems, err := lnk.List()
	if err != nil {
		return err
	}

	if len(managedItems) == 0 {
		printf(cmd, "📋 \033[1mNo files currently managed by lnk (common)\033[0m\n")
		printf(cmd, "   💡 Use \033[1mlnk add <file>\033[0m to start managing files\n")
		return nil
	}

	printf(cmd, "📋 \033[1mFiles managed by lnk (common)\033[0m (\033[36m%d item", len(managedItems))
	if len(managedItems) > 1 {
		printf(cmd, "s")
	}
	printf(cmd, "\033[0m):\n\n")

	for _, item := range managedItems {
		printf(cmd, "   🔗 \033[36m%s\033[0m\n", item)
	}

	printf(cmd, "\n💡 Use \033[1mlnk status\033[0m to check sync status\n")
	return nil
}

func listHostConfig(cmd *cobra.Command, host string) error {
	lnk := core.NewLnk(core.WithHost(host))
	managedItems, err := lnk.List()
	if err != nil {
		return err
	}

	if len(managedItems) == 0 {
		printf(cmd, "📋 \033[1mNo files currently managed by lnk (host: %s)\033[0m\n", host)
		printf(cmd, "   💡 Use \033[1mlnk add --host %s <file>\033[0m to start managing files\n", host)
		return nil
	}

	printf(cmd, "📋 \033[1mFiles managed by lnk (host: %s)\033[0m (\033[36m%d item", host, len(managedItems))
	if len(managedItems) > 1 {
		printf(cmd, "s")
	}
	printf(cmd, "\033[0m):\n\n")

	for _, item := range managedItems {
		printf(cmd, "   🔗 \033[36m%s\033[0m\n", item)
	}

	printf(cmd, "\n💡 Use \033[1mlnk status\033[0m to check sync status\n")
	return nil
}

func listAllConfigs(cmd *cobra.Command) error {
	// List common configuration
	printf(cmd, "📋 \033[1mAll configurations managed by lnk\033[0m\n\n")

	lnk := core.NewLnk()
	commonItems, err := lnk.List()
	if err != nil {
		return err
	}

	printf(cmd, "🌐 \033[1mCommon configuration\033[0m (\033[36m%d item", len(commonItems))
	if len(commonItems) > 1 {
		printf(cmd, "s")
	}
	printf(cmd, "\033[0m):\n")

	if len(commonItems) == 0 {
		printf(cmd, "   \033[90m(no files)\033[0m\n")
	} else {
		for _, item := range commonItems {
			printf(cmd, "   🔗 \033[36m%s\033[0m\n", item)
		}
	}

	// Find all host-specific configurations
	hosts, err := findHostConfigs()
	if err != nil {
		return err
	}

	for _, host := range hosts {
		printf(cmd, "\n🖥️  \033[1mHost: %s\033[0m", host)

		hostLnk := core.NewLnk(core.WithHost(host))
		hostItems, err := hostLnk.List()
		if err != nil {
			printf(cmd, " \033[31m(error: %v)\033[0m\n", err)
			continue
		}

		printf(cmd, " (\033[36m%d item", len(hostItems))
		if len(hostItems) > 1 {
			printf(cmd, "s")
		}
		printf(cmd, "\033[0m):\n")

		if len(hostItems) == 0 {
			printf(cmd, "   \033[90m(no files)\033[0m\n")
		} else {
			for _, item := range hostItems {
				printf(cmd, "   🔗 \033[36m%s\033[0m\n", item)
			}
		}
	}

	printf(cmd, "\n💡 Use \033[1mlnk list --host <hostname>\033[0m to see specific host configuration\n")
	return nil
}

func findHostConfigs() ([]string, error) {
	repoPath := getRepoPath()

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

func getRepoPath() string {
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			xdgConfig = "."
		} else {
			xdgConfig = filepath.Join(homeDir, ".config")
		}
	}
	return filepath.Join(xdgConfig, "lnk")
}
