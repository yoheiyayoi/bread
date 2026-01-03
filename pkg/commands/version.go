package commands

import (
	"fmt"
	"yoheiyayoi/bread/pkg/config"

	"github.com/spf13/cobra"
)

// Local
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print current version of bread",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🥖 Bread version v", config.Version)
	},
}

// Functions
func init() {
	rootCmd.AddCommand(versionCmd)
}
