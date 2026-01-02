package cmd

import (
	"fmt"
	"yoheiyayoi/bread/config"

	"github.com/spf13/cobra"
)

var Version = config.Version

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print current version of bread",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🥖 Bread version", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
