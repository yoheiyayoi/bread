package cmd

import (
	"os"
	"strings"
	"yoheiyayoi/bread/breadTypes"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

// Functions
func createProject(name string) {
	// Exist file check
	if _, err := os.Stat("bread.toml"); err == nil {
		log.Error("bread.toml already exists!")
		return
	}

	file, err := os.Create("bread.toml")
	if err != nil {
		log.Error("Failed to create project:", err)
		return
	}

	defer file.Close()

	configData := breadTypes.Config{
		Package: breadTypes.Package{
			Name:     name,
			Version:  "0.1.0",
			Registry: "https://github.com/UpliftGames/wally-index",
			Realm:    "shared",
		},

		BreadConfig: breadTypes.BreadConfig{
			PackagesDir: "Packages",
			ServerDir:   "ServerPackages",
			DevDir:      "DevPackages",
		},
		Dependencies: map[string]string{},
	}

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(configData); err != nil {
		log.Errorf("Failed to encode data to TOML: %v", err)
		return
	}

	log.Info("Init project successfully!")
}

// Command Init
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Bread project",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")

		if !strings.Contains(name, "/") {
			log.Error("Project name must be in the format 'username/project_name")
			return
		}

		createProject(name)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().String("name", "user/test", "Name of the project")
}
