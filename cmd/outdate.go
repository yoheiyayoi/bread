package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"yoheiyayoi/bread/breadTypes"
	"yoheiyayoi/bread/utils"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var outdatedCmd = &cobra.Command{
	Use:   "outdated",
	Short: "Check for outdated dependencies",
	Run: func(cmd *cobra.Command, args []string) {
		if err := checkOutdated(); err != nil {
			log.Error("Failed to check outdated packages", "error", err)
			return
		}
	},
}

type outdatedPackage struct {
	Name           string
	CurrentVersion string
	LatestVersion  string
	Realm          string
}

func checkOutdated() error {
	manifest, err := loadManifest()
	if err != nil {
		return err
	}

	lockfile, err := loadLockfile()
	if err != nil {
		return err
	}

	outdated := findOutdatedPackages(manifest, lockfile)
	displayOutdatedResults(outdated)
	return nil
}

func loadManifest() (*breadTypes.Config, error) {
	projectPath, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting current directory: %w", err)
	}

	tomlPath := filepath.Join(projectPath, "bread.toml")
	var manifest breadTypes.Config
	if _, err := toml.DecodeFile(tomlPath, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

func loadLockfile() (map[string][]breadTypes.LockedPackage, error) {
	lockfilePath := filepath.Join(".", "bread.lock")
	if _, err := os.Stat(lockfilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no bread.lock found. Run 'bread install' first")
	}

	var lockfileData struct {
		Registry string                     `toml:"registry"`
		Packages []breadTypes.LockedPackage `toml:"package"`
	}

	if _, err := toml.DecodeFile(lockfilePath, &lockfileData); err != nil {
		return nil, fmt.Errorf("failed to parse lockfile: %w", err)
	}

	lockfileMap := make(map[string][]breadTypes.LockedPackage)
	for _, pkg := range lockfileData.Packages {
		lockfileMap[pkg.Name] = append(lockfileMap[pkg.Name], pkg)
	}

	return lockfileMap, nil
}

// findOutdatedPackages checks all dependencies for updates
func findOutdatedPackages(manifest *breadTypes.Config, lockfile map[string][]breadTypes.LockedPackage) []outdatedPackage {
	checker := utils.NewVersionChecker()
	outdated := []outdatedPackage{}

	depGroups := map[string]map[string]string{
		"shared": manifest.Dependencies,
		"server": manifest.ServerDependencies,
		"dev":    manifest.DevDependencies,
	}

	for realm, deps := range depGroups {
		for name, constraint := range deps {
			if pkg := checkPackageVersion(name, constraint, realm, lockfile, checker); pkg != nil {
				outdated = append(outdated, *pkg)
			}
		}
	}

	return outdated
}

// checkPackageVersion checks if a single package is outdated
func checkPackageVersion(name, constraint, realm string, lockfile map[string][]breadTypes.LockedPackage, checker *utils.VersionChecker) *outdatedPackage {
	pkgName, _ := utils.ParsePackageSpec(name, constraint)

	currentVersion := checker.GetCurrentVersion(lockfile, pkgName)
	if currentVersion == "" {
		log.Warn("Package not found in lockfile", "package", pkgName)
		return nil
	}

	latestVersion, err := utils.GetPackageVersion(pkgName)
	if err != nil {
		log.Warn("Failed to resolve latest version", "package", pkgName, "error", err)
		return nil
	}

	if checker.IsOutdated(currentVersion, latestVersion[0]) {
		return &outdatedPackage{
			Name:           pkgName,
			CurrentVersion: currentVersion,
			LatestVersion:  latestVersion[0],
			Realm:          realm,
		}
	}

	return nil
}

// displayOutdatedResults shows the results to the user
func displayOutdatedResults(outdated []outdatedPackage) {
	if len(outdated) == 0 {
		log.Infof("%s All packages are up to date!", utils.Check)
		return
	}

	log.Warn(fmt.Sprintf("Found %d outdated package(s):", len(outdated)))
	fmt.Println()

	for _, pkg := range outdated {
		fmt.Printf("  📦 %s [%s]\n", pkg.Name, pkg.Realm)
		fmt.Printf("     Current: %s → Latest: %s\n", color.RedString(pkg.CurrentVersion), color.GreenString(pkg.LatestVersion))
	}
}

func init() {
	rootCmd.AddCommand(outdatedCmd)
}
