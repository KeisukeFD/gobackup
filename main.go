package main

import (
	"github.com/spf13/cobra"
	"gobackup/src/Commands"
	"gobackup/src/Model"
	"gobackup/src/Utils"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	Model.GetConfig()
	Utils.InitLogger(&Model.GetConfig().LoggerLevel)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		os.Exit(1)
	}()

	cmds := []*cobra.Command{
		Commands.BackupCommand(),
		Commands.HelperCommand(),
	}

	var rootCmd = Commands.RootCommand()
	for _, cmd := range cmds {
		rootCmd.AddCommand(cmd)
	}
	rootCmd.Execute()
}
