package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/mod/semver"
)

type PackageMetadata struct {
	Versions []struct {
		Package struct {
			Version string `json:"version"`
		} `json:"package"`
	} `json:"versions"`
}

var (
	metadataCache = make(map[string][]string)
	metadataMu    sync.Mutex
)

func ResolveVersion(name, constraint string) (string, error) {
	// Clean up constraint
	constraint = strings.TrimSpace(constraint)

	// Fetch available versions
	versions, err := getPackageVersions(name)
	if err != nil {
		return "", err
	}

	// Find the best match
	bestMatch := ""
	for _, v := range versions {
		// Ensure v starts with v for semver package
		semverV := v
		if !strings.HasPrefix(semverV, "v") {
			semverV = "v" + v
		}

		if matchConstraint(semverV, constraint) {
			if bestMatch == "" || semver.Compare(semverV, bestMatch) > 0 {
				bestMatch = semverV
			}
		}
	}

	if bestMatch == "" {
		return "", fmt.Errorf("no version found for %s satisfying %s", name, constraint)
	}

	return strings.TrimPrefix(bestMatch, "v"), nil
}

func getPackageVersions(name string) ([]string, error) {
	metadataMu.Lock()
	if versions, ok := metadataCache[name]; ok {
		metadataMu.Unlock()
		return versions, nil
	}
	metadataMu.Unlock()

	url := fmt.Sprintf("https://api.wally.run/v1/package-metadata/%s", name)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch metadata for %s: %s", name, resp.Status)
	}

	var meta PackageMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, err
	}

	var versions []string
	for _, v := range meta.Versions {
		versions = append(versions, v.Package.Version)
	}

	metadataMu.Lock()
	metadataCache[name] = versions
	metadataMu.Unlock()

	return versions, nil
}

func matchConstraint(version, constraint string) bool {
	// Basic implementation of semver constraint matching
	// Supports: ^, exact version, empty (latest)

	// Normalize version to have 'v' prefix
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	if !semver.IsValid(version) {
		return false
	}

	// Handle empty constraint (latest)
	if constraint == "" {
		return true
	}

	// Handle ^ constraint
	if strings.HasPrefix(constraint, "^") {
		baseVer := strings.TrimPrefix(constraint, "^")
		// If baseVer is just "1", treat as "1.0.0"
		if !strings.Contains(baseVer, ".") {
			baseVer += ".0.0"
		} else if strings.Count(baseVer, ".") == 1 {
			baseVer += ".0"
		}

		baseVer = "v" + baseVer
		if !semver.IsValid(baseVer) {
			return false
		}

		// ^1.2.3 := >=1.2.3 <2.0.0
		// ^0.2.3 := >=0.2.3 <0.3.0
		// ^0.0.3 := >=0.0.3 <0.0.4

		major := semver.Major(baseVer)

		// Check lower bound
		if semver.Compare(version, baseVer) < 0 {
			return false
		}

		// Check upper bound
		if major == "v0" {
			// Special handling for 0.x.x
			// If 0.x.x, then < 0.(x+1).0
			// If 0.0.x, then < 0.0.(x+1)

			parts := strings.Split(strings.TrimPrefix(baseVer, "v"), ".")
			if len(parts) >= 2 && parts[1] != "0" {
				// 0.1.2 -> < 0.2.0
				return strings.HasPrefix(version, fmt.Sprintf("v0.%s.", parts[1]))
			} else if len(parts) >= 3 && parts[1] == "0" {
				// 0.0.1 -> < 0.0.2
				return version == baseVer
			}

			// 0.x -> < 1.0.0
			return semver.Major(version) == "v0"
		} else {
			// >= 1.0.0
			// Must have same major version
			return semver.Major(version) == major
		}
	}

	// Handle exact version or partial version (e.g. "1", "1.2")
	// In Wally/npm, "1" means "^1.0.0"

	// Check if it's a partial version
	if !strings.Contains(constraint, ".") {
		// "1" -> "^1.0.0"
		return matchConstraint(version, "^"+constraint+".0.0")
	} else if strings.Count(constraint, ".") == 1 {
		// "1.2" -> "^1.2.0"
		return matchConstraint(version, "^"+constraint+".0")
	}

	// Exact version
	cVer := constraint
	if !strings.HasPrefix(cVer, "v") {
		cVer = "v" + cVer
	}

	if semver.IsValid(cVer) {
		return version == cVer
	}

	return false
}
