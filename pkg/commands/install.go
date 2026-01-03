package commands

import (
	"os"
	"yoheiyayoi/bread/pkg/utils"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install",
	Aliases: []string{"i"},
	Short:   "Install project dependencies [aliases: i]",
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, err := os.Getwd()
		if err != nil {
			log.Errorf("Error getting current directory: %s", err)
			return
		}

		installer := utils.NewInstaller(projectPath)
		if installer == nil {
			log.Error("Something went wrong during installer initialization.")
			return
		}

		if err := installer.InstallAll(); err != nil {
			log.Errorf("Installation failed: %s", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
