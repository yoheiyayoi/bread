package commands

import (
	"fmt"
	"strings"

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
	},
}

// Functions
func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().String("name", "user/test", "Name of the project")
}
