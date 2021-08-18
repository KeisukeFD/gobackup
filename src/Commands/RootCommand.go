package Commands

import (
	"github.com/spf13/cobra"
	"gobackup/src/Model"
	"os"
)

func RootCommand() *cobra.Command {
	rc := &cobra.Command{
		Use:              os.Args[0],
		PersistentPreRun: Root,
	}
	rc.PersistentFlags().StringP("config", "c", "config.yml", "Configuration file in yaml format")

	return rc
}

func Root(cmd *cobra.Command, args []string) {
	filename, _ := cmd.Flags().GetString("config")
	Model.GetConfig().InitBackupConfig(filename)
}
