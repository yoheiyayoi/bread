package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
)

func (ic *InstallationContext) writeRootPackageLinks(realm Realm, dependencies map[string]string) error {
	baseDir := ic.getRealmDir(realm)

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	for depName, versionSpec := range dependencies {
		pkgName, version := parsePackageSpec(depName, versionSpec)
		if err := ic.writeLinkFile(baseDir, pkgName, version, realm); err != nil {
			return err
		}
	}

	return nil
}

func (ic *InstallationContext) writeLinkFile(baseDir, pkgName, version string, realm Realm) error {
	shortName := getPackageName(pkgName)
	linkPath := filepath.Join(baseDir, shortName+".lua")
	content := ic.linkRootSameIndex(pkgName, version, realm)
	return os.WriteFile(linkPath, []byte(content), 0644)
}

func (ic *InstallationContext) linkRootSameIndex(name, version string, realm Realm) string {
	fullName := packageIDFileName(name, version)
	shortName := getPackageName(name)
	requirePath := fmt.Sprintf("require(script.Parent.%s[\"%s\"][\"%s\"])", IndexDirName, fullName, shortName)

	// Try to extract types from the package
	packageDir := filepath.Join(ic.getIndexDir(realm), fullName, shortName)
	typeExtractor := NewTypeExtractor()
	types, err := typeExtractor.ExtractTypesFromPackage(packageDir, shortName)

	if err != nil || len(types) == 0 {
		// No types found, use simple require
		return fmt.Sprintf("--Bread\n--%s\nreturn %s\n", fullName, requirePath)
	}

	// Log the types being re-exported
	log.Debugf("Re-exporting %d types from %s", len(types), shortName)

	// Generate link file with type re-exports
	return typeExtractor.GenerateLinkFileWithTypes(requirePath, types, "_Package", fullName)
}
