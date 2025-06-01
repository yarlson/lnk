package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	
	"github.com/yarlson/lnk/internal/service"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "📋 List files managed by lnk",
		Long:         "Display all files and directories currently managed by lnk.",
		SilenceUsage: true,
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
	ctx := context.Background()
	lnkService, err := service.New()
	if err != nil {
		return wrapServiceError("initialize lnk service", err)
	}

	managedFiles, err := lnkService.ListManagedFiles(ctx, "")
	if err != nil {
		return formatError(err)
	}

	if len(managedFiles) == 0 {
		printf(cmd, "📋 \033[1mNo files currently managed by lnk (common)\033[0m\n")
		printf(cmd, "   💡 Use \033[1mlnk add <file>\033[0m to start managing files\n")
		return nil
	}

	printf(cmd, "📋 \033[1mFiles managed by lnk (common)\033[0m (\033[36m%d item", len(managedFiles))
	if len(managedFiles) > 1 {
		printf(cmd, "s")
	}
	printf(cmd, "\033[0m):\n\n")

	for _, file := range managedFiles {
		printf(cmd, "   🔗 \033[36m%s\033[0m\n", file.RelativePath)
	}

	printf(cmd, "\n💡 Use \033[1mlnk status\033[0m to check sync status\n")
	return nil
}

func listHostConfig(cmd *cobra.Command, host string) error {
	ctx := context.Background()
	lnkService, err := service.New()
	if err != nil {
		return wrapServiceError("initialize lnk service", err)
	}

	managedFiles, err := lnkService.ListManagedFiles(ctx, host)
	if err != nil {
		return formatError(err)
	}

	if len(managedFiles) == 0 {
		printf(cmd, "📋 \033[1mNo files currently managed by lnk (host: %s)\033[0m\n", host)
		printf(cmd, "   💡 Use \033[1mlnk add --host %s <file>\033[0m to start managing files\n", host)
		return nil
	}

	printf(cmd, "📋 \033[1mFiles managed by lnk (host: %s)\033[0m (\033[36m%d item", host, len(managedFiles))
	if len(managedFiles) > 1 {
		printf(cmd, "s")
	}
	printf(cmd, "\033[0m):\n\n")

	for _, file := range managedFiles {
		printf(cmd, "   🔗 \033[36m%s\033[0m\n", file.RelativePath)
	}

	printf(cmd, "\n💡 Use \033[1mlnk status\033[0m to check sync status\n")
	return nil
}

func listAllConfigs(cmd *cobra.Command) error {
	ctx := context.Background()
	lnkService, err := service.New()
	if err != nil {
		return wrapServiceError("initialize lnk service", err)
	}

	// List common configuration
	printf(cmd, "📋 \033[1mAll configurations managed by lnk\033[0m\n\n")

	commonFiles, err := lnkService.ListManagedFiles(ctx, "")
	if err != nil {
		return formatError(err)
	}

	printf(cmd, "🌐 \033[1mCommon configuration\033[0m (\033[36m%d item", len(commonFiles))
	if len(commonFiles) > 1 {
		printf(cmd, "s")
	}
	printf(cmd, "\033[0m):\n")

	if len(commonFiles) == 0 {
		printf(cmd, "   \033[90m(no files)\033[0m\n")
	} else {
		for _, file := range commonFiles {
			printf(cmd, "   🔗 \033[36m%s\033[0m\n", file.RelativePath)
		}
	}

	// Find all host-specific configurations
	hosts, err := findHostConfigs(lnkService)
	if err != nil {
		return formatError(err)
	}

	for _, host := range hosts {
		printf(cmd, "\n🖥️  \033[1mHost: %s\033[0m", host)

		hostFiles, err := lnkService.ListManagedFiles(ctx, host)
		if err != nil {
			printf(cmd, " \033[31m(error: %v)\033[0m\n", err)
			continue
		}

		printf(cmd, " (\033[36m%d item", len(hostFiles))
		if len(hostFiles) > 1 {
			printf(cmd, "s")
		}
		printf(cmd, "\033[0m):\n")

		if len(hostFiles) == 0 {
			printf(cmd, "   \033[90m(no files)\033[0m\n")
		} else {
			for _, file := range hostFiles {
				printf(cmd, "   🔗 \033[36m%s\033[0m\n", file.RelativePath)
			}
		}
	}

	printf(cmd, "\n💡 Use \033[1mlnk list --host <hostname>\033[0m to see specific host configuration\n")
	return nil
}

func findHostConfigs(service *service.Service) ([]string, error) {
	repoPath := service.GetRepoPath()

	// Check if repo exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read repository directory: %w", err)
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
