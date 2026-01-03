package commands

import (
	"yoheiyayoi/bread/pkg/utils"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

// Local
var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Update Bread package manager to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		if err := utils.DoSelfUpdate(); err != nil {
			log.Error(err)
			return
		}
	},
}

// Functions
func init() {
	rootCmd.AddCommand(selfUpdateCmd)
}
