package utils

import (
	"fmt"
	"strings"
	breadTypes "yoheiyayoi/bread/pkg/bread_type"

	"golang.org/x/mod/semver"
)

// IsOutdated checks if current version is older than latest version
func IsOutdated(current, latest string) bool {
	currVer := normalizeVersion(current)
	latestVer := normalizeVersion(latest)

	if !semver.IsValid(currVer) || !semver.IsValid(latestVer) {
		return false
	}

	return semver.Compare(currVer, latestVer) < 0
}

// GetCurrentVersion retrieves the current version from lockfile
func GetCurrentVersion(lockfile map[string][]breadTypes.LockedPackage, packageName string) string {
	if packages, ok := lockfile[packageName]; ok && len(packages) > 0 {
		return packages[0].Version
	}
	return ""
}

// CompareVersions compares two versions and returns:
// -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func CompareVersions(v1, v2 string) int {
	ver1 := normalizeVersion(v1)
	ver2 := normalizeVersion(v2)

	if !semver.IsValid(ver1) || !semver.IsValid(ver2) {
		return 0
	}

	return semver.Compare(ver1, ver2)
}

// GetVersionInfo returns formatted version comparison info
func GetVersionInfo(current, latest string) string {
	if current == latest {
		return fmt.Sprintf("✓ %s (up to date)", current)
	}
	return fmt.Sprintf("%s → %s (update available)", current, latest)
}

// normalizeVersion adds 'v' prefix if not present
func normalizeVersion(version string) string {
	if version == "" || strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}
