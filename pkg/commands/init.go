package commands

import (
	"fmt"
	"strings"
	"yoheiyayoi/bread/pkg/utils"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

// Local
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Bread project 🥖",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")

		if !strings.Contains(name, "/") {
			log.Error("Invalid project name")
			fmt.Println("  → Missing '/' in project name")
			fmt.Println("  → Expected format: username/project_name")
			return
		}

		if err := utils.CreateManifest(name); err != nil {
			log.Errorf("Failed to initialize project: %v", err)
			return
		}

		log.Infof("%s Init project successfully!", CheckIcon)
	},
}

// Functions
func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().String("name", "user/test", "Name of the project")
}
