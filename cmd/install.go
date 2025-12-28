package cmd

import (
	"os"
	"yoheiyayoi/bread/utils"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install",
	Aliases: []string{"i"},
	Short:   "Install project dependencies",
	Long:    "Install project dependencies (And you can use 'bread i' instead of 'bread install')",
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, err := os.Getwd()
		if err != nil {
			log.Error("Error getting current directory:", err)
			return
		}

		var installation = utils.NewInstaller(projectPath, nil, nil)
		if installation == nil {
			return
		}

		if err := installation.Install(); err != nil {
			log.Error("Installation failed:", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
