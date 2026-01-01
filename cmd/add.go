package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"yoheiyayoi/bread/breadTypes"
	"yoheiyayoi/bread/utils"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <package>",
	Short: "Add project dependencies",
	Long:  "Add project dependencies (Auto install after adding)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, err := os.Getwd()
		if err != nil {
			log.Errorf("Error getting current directory: %s", err)
			return
		}

		packageSpec := args[0]
		depType, _ := cmd.Flags().GetString("types")
		pkgName, _ := cmd.Flags().GetString("name")

		tomlPath := filepath.Join(projectPath, "bread.toml")
		var config breadTypes.Config

		if _, err := toml.DecodeFile(tomlPath, &config); err != nil {
			log.Errorf("Failed to read bread.toml: %s", err)
			return
		}

		var packageName string
		if pkgName != "" {
			packageName = pkgName
		} else {
			if packageName, err = extractPackageName(packageSpec); err != nil {
				log.Errorf("Failed to extract package name: %s", err)
				return
			}
		}

		if !addDependency(&config, depType, packageName, packageSpec) {
			return
		}

		// Write back with proper formatting
		var buf bytes.Buffer
		encoder := toml.NewEncoder(&buf)
		encoder.Indent = "  "
		if err := encoder.Encode(config); err != nil {
			log.Errorf("Failed to encode bread.toml: %s", err)
			return
		}

		if err := os.WriteFile(tomlPath, buf.Bytes(), 0644); err != nil {
			log.Errorf("Failed to write bread.toml: %s", err)
			return
		}

		log.Infof("Added %s to dependencies", packageSpec)

		// Install all packages to ensure dependencies are resolved and lockfile is updated
		installation := utils.NewInstaller(projectPath, nil, nil)
		if installation == nil {
			return
		}

		if err := installation.InstallSinglePackage(packageName, packageSpec, utils.Realm(depType)); err != nil {
			log.Errorf("Installation failed: %s", err)
		}
	},
}

func addDependency(config *breadTypes.Config, depType string, packageName, packageSpec string) bool {
	var deps map[string]string
	var section string

	switch depType {
	case "server":
		if config.ServerDependencies == nil {
			config.ServerDependencies = make(map[string]string)
		}
		deps = config.ServerDependencies
		section = "server_dependencies"
	case "dev":
		if config.DevDependencies == nil {
			config.DevDependencies = make(map[string]string)
		}
		deps = config.DevDependencies
		section = "dev_dependencies"
	default:
		if config.Dependencies == nil {
			config.Dependencies = make(map[string]string)
		}
		deps = config.Dependencies
		section = "dependencies"
	}

	if _, exists := deps[packageName]; exists {
		log.Errorf("Package %s already exists in %s", packageName, section)
		return false
	}

	deps[packageName] = packageSpec
	return true
}

func extractPackageName(spec string) (string, error) {
	parts := strings.Split(spec, "/")
	if len(parts) < 2 {
		return "", errors.New("Invalid package!")
	}
	// Extract name part (e.g. "author/name" -> "name")
	name := strings.Split(parts[1], "@")[0]

	// Capitalize first letter
	// btw i simplify this shit
	if name == "" {
		return "", errors.New("invalid package name")
	}
	return strings.ToUpper(name[:1]) + name[1:], nil
}

func mapDepTypeToRealm(depType string) utils.Realm {
	switch depType {
	case "server":
		return utils.RealmServer
	case "dev":
		return utils.RealmDev
	default:
		return utils.RealmShared
	}
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().String("types", "shared", "Specify the package types to add (shared, server, dev)")
	addCmd.Flags().String("name", "", "Specify package name")
}
