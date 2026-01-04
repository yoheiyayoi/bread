package commands

import (
	"os"
	"path/filepath"
	breadTypes "yoheiyayoi/bread/pkg/bread_type"
	"yoheiyayoi/bread/pkg/utils"

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
	Run:     runRemove,
}

func runRemove(cmd *cobra.Command, args []string) {
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

	if err := saveManifest(tomlPath, config); err != nil {
		log.Error("Failed to save bread.toml:", err)
		return
	}

	log.Infof("Removed %s from dependencies", packageName)

	if err := reinstallPackages(projectPath); err != nil {
		log.Error("Reinstallation failed:", err)
	}
}

func removeDependency(config *breadTypes.Config, packageName string) bool {
	deps := []*map[string]string{
		&config.Dependencies,
		&config.ServerDependencies,
		&config.DevDependencies,
	}

	for _, depMap := range deps {
		if *depMap != nil {
			if _, exists := (*depMap)[packageName]; exists {
				delete(*depMap, packageName)
				return true
			}
		}
	}

	return false
}

func reinstallPackages(projectPath string) error {
	log.Info("Reinstalling packages...")

	installation := utils.NewInstaller(projectPath)
	if installation == nil {
		return nil
	}

	if err := installation.Clean(); err != nil {
		return err
	}

	return installation.InstallAll()
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
