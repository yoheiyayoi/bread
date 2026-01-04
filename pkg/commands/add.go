package commands

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strings"

	breadTypes "yoheiyayoi/bread/pkg/bread_type"
	"yoheiyayoi/bread/pkg/utils"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var allRealms = []string{"shared", "server", "dev"}

var addCmd = &cobra.Command{
	Use:   "add <package>",
	Short: "Add project dependencies",
	Long:  "Add project dependencies (Auto install after adding)",
	Args:  cobra.MinimumNArgs(1),
	Run:   runAdd,
}

func runAdd(cmd *cobra.Command, args []string) {
	projectPath, err := os.Getwd()
	if err != nil {
		log.Errorf("Error getting current directory: %s", err)
		return
	}

	packageSpec := args[0]
	depType, _ := cmd.Flags().GetString("realm")
	pkgNameFlag, _ := cmd.Flags().GetString("name")

	if !slices.Contains(allRealms, depType) {
		log.Error("Invalid package realm")
		fmt.Printf("  → You typed '%s'\n  → Expected one of: shared, server, dev\n", depType)
		return
	}

	tomlPath, err := utils.ManifestPath()
	if err != nil {
		log.Errorf("Failed to get bread.toml path: %s", err)
		return
	}

	var config breadTypes.Config
	if _, err := toml.DecodeFile(tomlPath, &config); err != nil {
		log.Errorf("Failed to read bread.toml: %s", err)
		return
	}

	packageName := pkgNameFlag
	if packageName == "" {
		if packageName, err = extractPackageName(packageSpec); err != nil {
			log.Errorf("Failed to extract package name: %s", err)
			return
		}
	}

	targetDeps := getTargetDeps(&config, depType)
	if _, exists := targetDeps[packageName]; exists {
		log.Errorf("Package %s already exists in %s realm", packageName, depType)
		return
	}

	targetDeps[packageName] = packageSpec

	if err := saveManifest(tomlPath, config); err != nil {
		log.Errorf("Failed to save bread.toml: %s", err)
		return
	}

	log.Infof("Added %s (%s) to %s realm", packageName, packageSpec, depType)

	if err := installPackage(projectPath, packageName, packageSpec, depType); err != nil {
		log.Errorf("Installation failed: %v", err)
		rollbackManifest(tomlPath, targetDeps, packageName, config)
		return
	}
}

func getTargetDeps(config *breadTypes.Config, depType string) map[string]string {
	switch strings.ToLower(depType) {
	case "server":
		if config.ServerDependencies == nil {
			config.ServerDependencies = make(map[string]string)
		}
		return config.ServerDependencies
	case "dev":
		if config.DevDependencies == nil {
			config.DevDependencies = make(map[string]string)
		}
		return config.DevDependencies
	default:
		if config.Dependencies == nil {
			config.Dependencies = make(map[string]string)
		}
		return config.Dependencies
	}
}

func saveManifest(path string, config breadTypes.Config) error {
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	encoder.Indent = "  "
	if err := encoder.Encode(config); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func installPackage(projectPath, packageName, packageSpec, depType string) error {
	installation := utils.NewInstaller(projectPath)
	if installation == nil {
		return fmt.Errorf("failed to create installer")
	}

	realmConst := utils.RealmShared
	switch strings.ToLower(depType) {
	case "server":
		realmConst = utils.RealmServer
	case "dev":
		realmConst = utils.RealmDev
	}

	packagesArray := [][]string{{fmt.Sprintf("%s@%s", packageName, packageSpec), realmConst}}
	return utils.RunPackageInstallation(installation, packagesArray)
}

func rollbackManifest(path string, targetDeps map[string]string, packageName string, config breadTypes.Config) {
	delete(targetDeps, packageName)
	if err := saveManifest(path, config); err != nil {
		log.Errorf("Failed to rollback bread.toml: %v", err)
	} else {
		log.Warn("Rolled back bread.toml change due to installation failure")
	}
}

func extractPackageName(spec string) (string, error) {
	parts := strings.Split(spec, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid package spec: must be scoped (author/package)")
	}

	base := strings.Split(parts[1], "@")[0]
	if base == "" {
		return "", fmt.Errorf("invalid package name")
	}

	// Convert hyphenated names to CamelCase (bridge-net2 → BridgeNet2)
	subparts := strings.Split(base, "-")
	for i, part := range subparts {
		if len(part) > 0 {
			subparts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(subparts, ""), nil
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringP("realm", "r", "shared", "Realm to add the package to (shared, server, dev)")
	addCmd.Flags().String("name", "", "Override the dependency name in bread.toml (default: CamelCase from package name)")
}
