package cmd

import (
	"fmt"
	"yoheiyayoi/bread/src/config"

	"github.com/spf13/cobra"
)

var Version = config.Version

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print current version of bread",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("bread version", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
