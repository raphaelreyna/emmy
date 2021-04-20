package cmd

import (
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "emmy",
		Short: "Automate the creation of Docker network topologies and analyze them.",
	}
)

func Execute() error {
	return rootCmd.Execute()
}
