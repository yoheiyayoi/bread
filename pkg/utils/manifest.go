package utils

import (
	"fmt"
	"os"
	"path/filepath"
	breadTypes "yoheiyayoi/bread/pkg/bread_type"

	"github.com/BurntSushi/toml"
)

// Local
const ManifestFileName = "bread.toml"

// Functions

// Get bread.toml path
func ManifestPath() (string, error) {
	projectPath, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting current directory: %w", err)
	}

	return filepath.Join(projectPath, ManifestFileName), nil
}

func CreateManifest(name string) error {
	// Exist file check
	if _, err := os.Stat(ManifestFileName); err == nil {
		return fmt.Errorf("%s already exist", ManifestFileName)
	}

	file, err := os.Create(ManifestFileName)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	defer file.Close()

	configData := breadTypes.Config{
		Package: breadTypes.Package{
			Name:     name,
			Version:  "0.1.0",
			Registry: "https://github.com/UpliftGames/wally-index",
			Realm:    "shared",
		},

		BreadConfig: breadTypes.BreadConfig{
			PackagesDir: "Packages",
			ServerDir:   "ServerPackages",
			DevDir:      "DevPackages",
		},

		Dependencies: map[string]string{},
	}

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(configData); err != nil {
		return fmt.Errorf("failed to encode data to TOML: %w", err)
	}

	return nil
}
