package utils

import (
	"fmt"
	"yoheiyayoi/bread/breadTypes"

	"golang.org/x/mod/semver"
)

// VersionChecker provides utilities for checking package versions
type VersionChecker struct{}

// NewVersionChecker creates a new version checker instance
func NewVersionChecker() *VersionChecker {
	return &VersionChecker{}
}

// IsOutdated checks if current version is older than latest version
func (vc *VersionChecker) IsOutdated(current, latest string) bool {
	// Normalize versions
	currVer := ensureVersionPrefix(current)
	latestVer := ensureVersionPrefix(latest)

	if !semver.IsValid(currVer) || !semver.IsValid(latestVer) {
		return false
	}

	return semver.Compare(currVer, latestVer) < 0
}

// GetCurrentVersion retrieves the current version from lockfile
func (vc *VersionChecker) GetCurrentVersion(lockfile map[string][]breadTypes.LockedPackage, packageName string) string {
	if packages, ok := lockfile[packageName]; ok && len(packages) > 0 {
		return packages[0].Version
	}
	return ""
}

// CompareVersions compares two versions and returns:
// -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func (vc *VersionChecker) CompareVersions(v1, v2 string) int {
	ver1 := ensureVersionPrefix(v1)
	ver2 := ensureVersionPrefix(v2)

	if !semver.IsValid(ver1) || !semver.IsValid(ver2) {
		return 0
	}

	return semver.Compare(ver1, ver2)
}

// GetVersionInfo returns formatted version comparison info
func (vc *VersionChecker) GetVersionInfo(current, latest string) string {
	if current == latest {
		return fmt.Sprintf("✓ %s (up to date)", current)
	}
	return fmt.Sprintf("%s → %s (update available)", current, latest)
}

// ensureVersionPrefix adds 'v' prefix if not present
func ensureVersionPrefix(version string) string {
	if version == "" {
		return ""
	}
	if version[0] == 'v' {
		return version
	}
	return "v" + version
}
