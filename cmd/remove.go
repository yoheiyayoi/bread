package cmd

import (
	"bytes"
	"os"
	"path/filepath"

	"yoheiyayoi/bread/breadTypes"
	"yoheiyayoi/bread/utils"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <package>",
	Aliases: []string{"rm", "uninstall"},
	Short:   "Remove project dependencies [aliases: rm]",
	Long:    "Remove project dependencies and reinstall remaining packages",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, err := os.Getwd()
		if err != nil {
			log.Error("Error getting current directory:", err)
			return
		}

		packageName := args[0]
		tomlPath := filepath.Join(projectPath, "bread.toml")
		var config breadTypes.Config

		if _, err := toml.DecodeFile(tomlPath, &config); err != nil {
			log.Error("Failed to read bread.toml:", err)
			return
		}

		if !removeDependency(&config, packageName) {
			log.Errorf("Package %s not found in dependencies", packageName)
			return
		}

		// Write back with proper formatting
		var buf bytes.Buffer
		encoder := toml.NewEncoder(&buf)
		encoder.Indent = "  "
		if err := encoder.Encode(config); err != nil {
			log.Error("Failed to encode bread.toml:", err)
			return
		}

		if err := os.WriteFile(tomlPath, buf.Bytes(), 0644); err != nil {
			log.Error("Failed to write bread.toml:", err)
			return
		}

		log.Infof("Removed %s from dependencies", packageName)

		// Reinstall packages to clean up shit
		log.Info("Reinstalling packages...")
		installation := utils.NewInstaller(projectPath, nil, nil)
		if installation == nil {
			return
		}

		if err := installation.Clean(); err != nil {
			log.Error("Failed to clean packages:", err)
			return
		}

		if err := installation.Install(); err != nil {
			log.Error("Installation failed:", err)
			return
		}
	},
}

func removeDependency(config *breadTypes.Config, packageName string) bool {
	found := false

	if config.Dependencies != nil {
		if _, exists := config.Dependencies[packageName]; exists {
			delete(config.Dependencies, packageName)
			found = true
		}
	}

	if config.ServerDependencies != nil {
		if _, exists := config.ServerDependencies[packageName]; exists {
			delete(config.ServerDependencies, packageName)
			found = true
		}
	}

	if config.DevDependencies != nil {
		if _, exists := config.DevDependencies[packageName]; exists {
			delete(config.DevDependencies, packageName)
			found = true
		}
	}

	return found
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
