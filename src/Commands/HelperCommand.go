package Commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gobackup/src/Model"
	"gobackup/src/Services"
	"os"
	"strings"
)

func HelperCommand() *cobra.Command {
	bc := &cobra.Command{
		Use:   "restic",
		Short: "Restic helper command",
		Long:  "Restic helper command, allows you to use restic using the config file",
		Args:  cobra.MinimumNArgs(1),
		Run:   RunRestic,
	}

	bc.Flags().StringP("repo", "r", "", "Restic repository name")

	return bc
}

func RunRestic(cmd *cobra.Command, args []string) {
	var repositoryName string
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		switch flag.Name {
		case "repo":
			repositoryName = flag.Value.String()
		default:
			break
		}
	})
	Model.GetConfig().GetResticPassword()

	Services.InitBackupManager(Model.GetConfig(), repositoryName, []string{})
	bm := Services.GetBackupManager()
	res, err := bm.ExecuteRestic(strings.Join(args, " "))

	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, res.Output)
	} else {
		_, _ = fmt.Fprintln(os.Stdout, res.Output)
	}

}
