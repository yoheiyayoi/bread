package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"yoheiyayoi/bread/breadTypes"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
)

// Types
type InstallationContext struct {
	Manifest   breadTypes.Config
	SharedDir  string
	ServerDir  string
	DevDir     string
	SharedPath *string
	ServerPath *string
}

type Realm string

const (
	RealmShared  Realm = "shared"
	RealmServer  Realm = "server"
	RealmDev     Realm = "dev"
	IndexDirName       = "_Index"
)

// Functions
func NewInstaller(projectPath string, sharedPath *string, serverPath *string) *InstallationContext {
	configPath := filepath.Join(projectPath, "bread.toml")
	var config breadTypes.Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		log.Error("bread.toml not found!, Run bread init first")
		return nil
	}

	getDir := func(configDir, defaultName string) string {
		if configDir != "" {
			return filepath.Join(projectPath, configDir)
		}
		return filepath.Join(projectPath, defaultName)
	}

	return &InstallationContext{
		Manifest:   config,
		SharedDir:  getDir(config.BreadConfig.PackagesDir, "Packages"),
		ServerDir:  getDir(config.BreadConfig.ServerDir, "ServerPackages"),
		DevDir:     getDir(config.BreadConfig.DevDir, "DevPackages"),
		SharedPath: sharedPath,
		ServerPath: serverPath,
	}
}

func (ic *InstallationContext) Clean() error {
	for _, dir := range []string{ic.SharedDir, ic.ServerDir, ic.DevDir} {
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (ic *InstallationContext) getRealmDir(realm Realm) string {
	switch realm {
	case RealmServer:
		return ic.ServerDir
	case RealmDev:
		return ic.DevDir
	default:
		return ic.SharedDir
	}
}

func (ic *InstallationContext) getIndexDir(realm Realm) string {
	return filepath.Join(ic.getRealmDir(realm), IndexDirName)
}

func (ic *InstallationContext) Install() error {
	start := time.Now()

	// Create directories and install packages for each realm
	realms := []struct {
		realm Realm
		deps  map[string]string
	}{
		{RealmShared, ic.Manifest.Dependencies},
		{RealmServer, ic.Manifest.ServerDependencies},
		{RealmDev, ic.Manifest.DevDependencies},
	}

	var wg sync.WaitGroup
	totalDeps := len(ic.Manifest.Dependencies) + len(ic.Manifest.ServerDependencies) + len(ic.Manifest.DevDependencies)
	errChan := make(chan error, totalDeps)

	// Add counters
	var successCount, totalCount int32
	var mu sync.Mutex

	for _, r := range realms {
		if len(r.deps) == 0 {
			continue
		}

		// Create index directory
		if err := os.MkdirAll(ic.getIndexDir(r.realm), 0755); err != nil {
			return err
		}

		// Download packages
		for name, versionSpec := range r.deps {
			wg.Add(1)
			go func(n, vs string, realm Realm) {
				defer wg.Done()
				mu.Lock()
				totalCount++
				mu.Unlock()

				pkgName, version := parsePackageSpec(n, vs)
				log.Infof("[◆] Downloading %s@%s", pkgName, version)

				if err := ic.downloadPackage(pkgName, version, realm); err != nil {
					errChan <- err
				} else {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}(name, versionSpec, r.realm)
		}
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	// Write root package links
	for _, r := range realms {
		if len(r.deps) > 0 {
			if err := ic.writeRootPackageLinks(r.realm, r.deps); err != nil {
				return err
			}
		}
	}

	log.Infof("Installation complete! %d/%d packages installed successfully", successCount, totalCount)
	elapsed := time.Since(start)
	log.Infof("Time taken: %dms", time.Duration(elapsed).Milliseconds())
	return nil
}

func (ic *InstallationContext) InstallSinglePackage(name, versionSpec string, realm Realm) error {
	start := time.Now()

	// Create index directory
	if err := os.MkdirAll(ic.getIndexDir(realm), 0755); err != nil {
		return err
	}

	// Parse and download the package
	pkgName, version := parsePackageSpec(name, versionSpec)
	log.Infof("[◆] Downloading %s@%s", pkgName, version)
	if err := ic.downloadPackage(pkgName, version, realm); err != nil {
		return err
	}

	// Write root package link for this single package
	baseDir := ic.getRealmDir(realm)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	shortName := getPackageName(pkgName)
	linkPath := filepath.Join(baseDir, shortName+".lua")
	content := ic.linkRootSameIndex(pkgName, version, realm)
	if err := os.WriteFile(linkPath, []byte(content), 0644); err != nil {
		return err
	}

	log.Infof("Package %s@%s installed successfully", pkgName, version)
	elapsed := time.Since(start)
	log.Infof("Time taken: %dms", time.Duration(elapsed).Milliseconds())
	return nil
}

func parsePackageSpec(alias, versionSpec string) (packageName, version string) {
	if strings.Contains(versionSpec, "@") {
		parts := strings.SplitN(versionSpec, "@", 2)
		return parts[0], parts[1]
	}
	return alias, versionSpec
}

func (ic *InstallationContext) downloadPackage(name, version string, realm Realm) error {
	url := fmt.Sprintf("https://api.wally.run/v1/package-contents/%s/%s", name, version)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "bread/1.0")
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("Wally-Version", "0.3.2")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("package %s@%s not found", name, version)
		}
		return fmt.Errorf("failed to download %s@%s: HTTP %d", name, version, resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "package-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return err
	}
	tmpFile.Close()

	packageDirName := packageIDFileName(name, version)
	targetDir := filepath.Join(ic.getIndexDir(realm), packageDirName)

	if err := unzipPackage(tmpFile.Name(), targetDir, name); err != nil {
		return err
	}

	// Keep this one - it shows success per package
	log.Infof("✓ %s@%s", name, version)
	return nil
}

func (ic *InstallationContext) writeRootPackageLinks(realm Realm, dependencies map[string]string) error {
	baseDir := ic.getRealmDir(realm)

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	for depName, versionSpec := range dependencies {
		pkgName, version := parsePackageSpec(depName, versionSpec)
		shortName := getPackageName(pkgName)
		linkPath := filepath.Join(baseDir, shortName+".lua")
		content := ic.linkRootSameIndex(pkgName, version, realm)
		if err := os.WriteFile(linkPath, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
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
		return fmt.Sprintf("return %s\n", requirePath)
	}

	// Log the types being re-exported
	typeNames := make([]string, len(types))
	for i, t := range types {
		typeNames[i] = t.Name
	}
	log.Debugf("Re-exporting %d types from %s: %v", len(types), shortName, typeNames)

	// Generate link file with type re-exports
	return typeExtractor.GenerateLinkFileWithTypes(requirePath, types, "_Package")
}

func packageIDFileName(name, version string) string {
	parts := strings.Split(name, "/")
	if len(parts) == 2 {
		return fmt.Sprintf("%s_%s@%s", parts[0], parts[1], version)
	}
	return fmt.Sprintf("%s@%s", name, version)
}

func getPackageName(name string) string {
	if parts := strings.Split(name, "/"); len(parts) == 2 {
		return parts[1]
	}
	return name
}

func unzipPackage(src, dest, packageName string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	packageDir := filepath.Join(dest, getPackageName(packageName))
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return err
	}

	for _, f := range r.File {
		fpath := filepath.Join(packageDir, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		outFile, err := os.Create(fpath)
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
