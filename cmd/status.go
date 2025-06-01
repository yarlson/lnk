package cmd

import (
	"context"

	"github.com/spf13/cobra"
	
	"github.com/yarlson/lnk/internal/models"
	"github.com/yarlson/lnk/internal/service"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "status",
		Short:        "📊 Show repository sync status",
		Long:         "Display how many commits ahead/behind the local repository is relative to the remote and check for uncommitted changes.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			lnkService, err := service.New()
			if err != nil {
				return wrapServiceError("initialize lnk service", err)
			}

			ctx := context.Background()
			status, err := lnkService.GetStatus(ctx)
			if err != nil {
				return formatError(err)
			}

			if status.Dirty {
				displayDirtyStatus(cmd, status)
				return nil
			}

			if status.Ahead == 0 && status.Behind == 0 {
				displayUpToDateStatus(cmd, status)
				return nil
			}

			displaySyncStatus(cmd, status)
			return nil
		},
	}
}

func displayDirtyStatus(cmd *cobra.Command, status *models.SyncStatus) {
	printf(cmd, "⚠️  \033[1;33mRepository has uncommitted changes\033[0m\n")
	printf(cmd, "   📡 Remote: \033[36m%s\033[0m\n", getRemoteDisplay(status))

	if status.Ahead == 0 && status.Behind == 0 {
		printf(cmd, "\n💡 Run \033[1mgit add && git commit\033[0m in \033[36m~/.config/lnk\033[0m or \033[1mlnk push\033[0m to commit changes\n")
		return
	}

	printf(cmd, "\n")
	displayAheadBehindInfo(cmd, status, true)
	printf(cmd, "\n💡 Run \033[1mgit add && git commit\033[0m in \033[36m~/.config/lnk\033[0m or \033[1mlnk push\033[0m to commit changes\n")
}

func displayUpToDateStatus(cmd *cobra.Command, status *models.SyncStatus) {
	printf(cmd, "✅ \033[1;32mRepository is up to date\033[0m\n")
	printf(cmd, "   📡 Synced with \033[36m%s\033[0m\n", getRemoteDisplay(status))
}

func displaySyncStatus(cmd *cobra.Command, status *models.SyncStatus) {
	printf(cmd, "📊 \033[1mRepository Status\033[0m\n")
	printf(cmd, "   📡 Remote: \033[36m%s\033[0m\n", getRemoteDisplay(status))
	printf(cmd, "\n")

	displayAheadBehindInfo(cmd, status, false)

	if status.Ahead > 0 && status.Behind == 0 {
		printf(cmd, "\n💡 Run \033[1mlnk push\033[0m to sync your changes\n")
	} else if status.Behind > 0 {
		printf(cmd, "\n💡 Run \033[1mlnk pull\033[0m to get latest changes\n")
	}
}

func displayAheadBehindInfo(cmd *cobra.Command, status *models.SyncStatus, isDirty bool) {
	if status.Ahead > 0 {
		commitText := getCommitText(status.Ahead)
		if isDirty {
			printf(cmd, "   ⬆️ \033[1;33m%d %s ahead\033[0m (excluding uncommitted changes)\n", status.Ahead, commitText)
		} else {
			printf(cmd, "   ⬆️ \033[1;33m%d %s ahead\033[0m - ready to push\n", status.Ahead, commitText)
		}
	}

	if status.Behind > 0 {
		commitText := getCommitText(status.Behind)
		printf(cmd, "   ⬇️ \033[1;31m%d %s behind\033[0m - run \033[1mlnk pull\033[0m\n", status.Behind, commitText)
	}
}

func getCommitText(count int) string {
	if count == 1 {
		return "commit"
	}
	return "commits"
}

func getRemoteDisplay(status *models.SyncStatus) string {
	if status.HasRemote && status.RemoteBranch != "" {
		return status.RemoteBranch
	}
	if status.HasRemote && status.RemoteURL != "" {
		return status.RemoteURL
	}
	return "no remote configured"
}
