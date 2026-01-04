package commands

import (
	"fmt"
	breadTypes "yoheiyayoi/bread/pkg/bread_type"
	"yoheiyayoi/bread/pkg/utils"

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
	var manifest breadTypes.Config
	if err := utils.DecodeManifest(&manifest); err != nil {
		return err
	}

	lockfile, err := loadLockfile()
	if err != nil {
		return err
	}

	outdated := findOutdatedPackages(&manifest, lockfile)
	displayOutdatedResults(outdated)
	return nil
}

func loadLockfile() (map[string][]breadTypes.LockedPackage, error) {
	var lockfileData breadTypes.Lockfile
	if err := utils.LoadLockfile(&lockfileData); err != nil {
		return nil, err
	}

	lockfileMap := make(map[string][]breadTypes.LockedPackage)
	for _, pkg := range lockfileData.Packages {
		lockfileMap[pkg.Name] = append(lockfileMap[pkg.Name], pkg)
	}

	return lockfileMap, nil
}

// findOutdatedPackages checks all dependencies for updates
func findOutdatedPackages(manifest *breadTypes.Config, lockfile map[string][]breadTypes.LockedPackage) []outdatedPackage {
	outdated := []outdatedPackage{}

	depGroups := map[string]map[string]string{
		"shared": manifest.Dependencies,
		"server": manifest.ServerDependencies,
		"dev":    manifest.DevDependencies,
	}

	for realm, deps := range depGroups {
		for name, constraint := range deps {
			if pkg := checkPackageVersion(name, constraint, realm, lockfile); pkg != nil {
				outdated = append(outdated, *pkg)
			}
		}
	}

	return outdated
}

// checkPackageVersion checks if a single package is outdated
func checkPackageVersion(name, constraint, realm string, lockfile map[string][]breadTypes.LockedPackage) *outdatedPackage {
	pkgName, _ := utils.ParsePackageSpec(name, constraint)

	currentVersion := utils.GetCurrentVersion(lockfile, name)
	if currentVersion == "" {
		// Try with pkgName as fallback
		currentVersion = utils.GetCurrentVersion(lockfile, pkgName)
	}

	if currentVersion == "" {
		log.Warn("Package not found in lockfile", "package", name, "full_name", pkgName)
		return nil
	}

	latestVersion, err := utils.GetPackageVersions(pkgName)
	if err != nil {
		log.Warn("Failed to resolve latest version", "package", pkgName, "error", err)
		return nil
	}

	if utils.IsOutdated(currentVersion, latestVersion[0]) {
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
