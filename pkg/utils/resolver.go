package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/Masterminds/semver/v3"
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
	metadataMu    sync.RWMutex
)

func ResolveVersion(name, constraintStr string) (string, error) {
	// fetch versions
	versions, err := GetPackageVersions(name)
	if err != nil {
		return "", err
	}

	if constraintStr == "" {
		constraintStr = "*"
	}

	c, err := semver.NewConstraint(constraintStr)
	if err != nil {
		return "", fmt.Errorf("invalid constraint: %w", err)
	}

	// parse and sort available versions (descending skibiding)
	var vs []*semver.Version
	for _, v := range versions {
		if parsed, err := semver.NewVersion(v); err == nil {
			vs = append(vs, parsed)
		}
	}
	sort.Sort(sort.Reverse(semver.Collection(vs)))

	// find the first highest version that matches
	for _, v := range vs {
		if c.Check(v) {
			return v.String(), nil
		}
	}

	return "", fmt.Errorf("no version found for %s satisfying %s", name, constraintStr)
}

func GetPackageVersions(name string) ([]string, error) {
	metadataMu.RLock()
	if versions, ok := metadataCache[name]; ok {
		metadataMu.RUnlock()
		return versions, nil
	}
	metadataMu.RUnlock()

	url := fmt.Sprintf("https://api.wally.run/v1/package-metadata/%s", name)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch metadata: %s", resp.Status)
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

func MatchConstraint(version, constraint string) (bool, error) {
	if constraint == "" {
		constraint = "*"
	}
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false, err
	}
	v, err := semver.NewVersion(version)
	if err != nil {
		return false, err
	}
	return c.Check(v), nil
}
