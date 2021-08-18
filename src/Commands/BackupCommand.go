package Commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gobackup/src/Model"
	"gobackup/src/Services"
	"gobackup/src/Utils"
	"os"
)

func BackupCommand() *cobra.Command {
	bc := &cobra.Command{
		Use:   "backup",
		Short: "Backup with restic",
		Long:  "Backup with restic, list of folders to backup in argument",
		Args:  cobra.MinimumNArgs(1),
		Run:   RunBackup,
	}

	bc.Flags().StringP("repo", "r", "", "Restic repository name")
	bc.Flags().String("metrics-file", "backup.prom", "Export metrics file as Prometheus format")

	return bc
}

func RunBackup(cmd *cobra.Command, args []string) {
	var repositoryName string
	var folders = args
	var metricsFilename string
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		switch flag.Name {
		case "repo":
			repositoryName = flag.Value.String()
		case "metrics-file":
			metricsFilename = flag.Value.String()
		default:
			break
		}
	})

	Model.GetConfig().GetResticPassword()

	err := _checkIfFoldersExists(folders)
	Utils.HaltOnError(Utils.GetLogger(), err, "")

	email, err := Services.NewEmailServer(Model.GetConfig())
	Utils.HaltOnError(Utils.GetLogger(), err, "")

	Services.InitBackupManager(Model.GetConfig(), repositoryName, folders)
	bm := Services.GetBackupManager()
	_, err = bm.ExecutePreCommand()
	Utils.WarnOnError(Utils.GetLogger(), err, "Error during Pre-Command", nil)
	bm.InitRepo()
	bm.StartBackup()
	bm.Cleanup()
	bm.CheckRepoIntegrity()
	_, err = bm.ExecutePostCommand()
	Utils.WarnOnError(Utils.GetLogger(), err, "Error during Pre-Command", nil)
	bm.GetResults()
	if metricsFilename != "" {
		metrics := bm.GetMetrics()
		err := Utils.ExportMetricsToFile(metricsFilename, metrics)
		Utils.WarnOnError(Utils.GetLogger(), err, "Error while exporting metrics to prometheus", nil)
	}
	if Model.GetConfig().BackupConfig.Email.Enabled {
		body := bm.MakeEmailBody()
		mail := &Services.Email{
			From: Model.GetConfig().BackupConfig.Email.Sender,
			To:   Model.GetConfig().BackupConfig.Email.To,
			Body: body,
		}
		if err := email.Send(mail); err != nil {
			Utils.GetLogger().Error("Email can't be send !", err.Error())
		}
	}
}

func _checkIfFoldersExists(folders []string) error {
	for _, folder := range folders {
		if _, err := os.Stat(folder); err != nil {
			return err
		}
	}
	return nil
}
