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
	"sync/atomic"
	"time"
	"yoheiyayoi/bread/breadTypes"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
	"github.com/fatih/color"
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

var (
	Check = color.GreenString("✓")
	Info  = color.BlueString("ℹ")
)

// Helper struct for installation state
type installSession struct {
	wg           sync.WaitGroup
	installed    sync.Map // concurrent map for visited packages
	errChan      chan error
	successCount atomic.Int32
	totalCount   atomic.Int32
}

func newInstallSession() *installSession {
	return &installSession{
		errChan: make(chan error, 1000),
	}
}

// Functions
func init() {
	log.SetReportTimestamp(false)
}

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
	session := newInstallSession()

	// Create directories and install packages for each realm
	realms := []struct {
		realm Realm
		deps  map[string]string
	}{
		{RealmShared, ic.Manifest.Dependencies},
		{RealmServer, ic.Manifest.ServerDependencies},
		{RealmDev, ic.Manifest.DevDependencies},
	}

	log.Info("Installing packages...")

	// 1. Download Phase
	for _, r := range realms {
		if len(r.deps) == 0 {
			continue
		}

		// Create index directory
		if err := os.MkdirAll(ic.getIndexDir(r.realm), 0755); err != nil {
			return err
		}

		// Download packages recursively
		for name, versionSpec := range r.deps {
			ic.installPackageRecursive(name, versionSpec, r.realm, session)
		}
	}

	session.wg.Wait()
	close(session.errChan)

	// Check for errors
	for err := range session.errChan {
		if err != nil {
			return err
		}
	}

	// 2. Linking Phase
	for _, r := range realms {
		if len(r.deps) > 0 {
			if err := ic.writeRootPackageLinks(r.realm, r.deps); err != nil {
				return err
			}
		}
	}

	elapsed := time.Since(start)
	log.Infof("%s Installed %d packages in %dms", Check, session.successCount.Load(), elapsed.Milliseconds())
	return nil
}

func (ic *InstallationContext) installPackageRecursive(name, versionSpec string, realm Realm, session *installSession) {
	pkgName, versionConstraint := parsePackageSpec(name, versionSpec)

	// Resolve version constraint to concrete version
	version, err := ResolveVersion(pkgName, versionConstraint)
	if err != nil {
		session.errChan <- fmt.Errorf("failed to resolve version for %s@%s: %v", pkgName, versionConstraint, err)
		return
	}

	pkgID := fmt.Sprintf("%s:%s@%s", realm, pkgName, version)

	// Check if already installed/processing
	if _, loaded := session.installed.LoadOrStore(pkgID, true); loaded {
		return
	}

	session.totalCount.Add(1)
	session.wg.Add(1)

	go func() {
		defer session.wg.Done()

		if err := ic.downloadPackage(pkgName, version, realm); err != nil {
			session.errChan <- err
			return
		}

		// Log success only
		fmt.Printf("%s Downloaded %s@%s\n", Check, pkgName, version)
		session.successCount.Add(1)

		// Read dependencies from the downloaded package
		deps, err := ic.getPackageDependencies(pkgName, version, realm)
		if err != nil {
			log.Errorf("Failed to read dependencies for %s@%s: %v", pkgName, version, err)
			return
		}

		for depName, depVersion := range deps {
			ic.installPackageRecursive(depName, depVersion, realm, session)
		}
	}()
}

func (ic *InstallationContext) getPackageDependencies(name, version string, realm Realm) (map[string]string, error) {
	fullName := packageIDFileName(name, version)
	shortName := getPackageName(name)
	packageDir := filepath.Join(ic.getIndexDir(realm), fullName, shortName)

	// Check for wally.toml or bread.toml
	for _, fname := range []string{"wally.toml", "bread.toml"} {
		configPath := filepath.Join(packageDir, fname)
		if _, err := os.Stat(configPath); err == nil {
			var config breadTypes.Config
			if _, err := toml.DecodeFile(configPath, &config); err != nil {
				return nil, err
			}
			return config.Dependencies, nil
		}
	}

	return nil, nil // No config found
}

func (ic *InstallationContext) InstallSinglePackage(name, versionSpec string, realm Realm) error {
	start := time.Now()

	// Create index directory
	if err := os.MkdirAll(ic.getIndexDir(realm), 0755); err != nil {
		return err
	}

	// Parse and download the package
	pkgName, versionConstraint := parsePackageSpec(name, versionSpec)
	version, err := ResolveVersion(pkgName, versionConstraint)
	if err != nil {
		return err
	}

	if err := ic.downloadPackage(pkgName, version, realm); err != nil {
		return err
	}

	// Write root package link for this single package
	baseDir := ic.getRealmDir(realm)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	if err := ic.writeLinkFile(baseDir, pkgName, version, realm); err != nil {
		return err
	}

	fmt.Printf("%s Downloaded %s@%s\n", Check, pkgName, version)

	elapsed := time.Since(start)
	log.Infof("%s Installed %s@%s in %dms", Check, pkgName, version, elapsed.Milliseconds())
	return nil
}

func parsePackageSpec(alias, versionSpec string) (packageName, version string) {
	if parts := strings.SplitN(versionSpec, "@", 2); len(parts) == 2 {
		return parts[0], parts[1]
	}
	// If versionSpec looks like "author/repo", assume it's the package name and version is empty (latest)
	if strings.Contains(versionSpec, "/") {
		return versionSpec, ""
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

	resp, err := http.DefaultClient.Do(req)
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
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return err
	}

	packageDirName := packageIDFileName(name, version)
	targetDir := filepath.Join(ic.getIndexDir(realm), packageDirName)

	return unzipPackage(tmpFile.Name(), targetDir, name)
}

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
		return fmt.Sprintf("return %s\n", requirePath)
	}

	// Log the types being re-exported
	log.Debugf("Re-exporting %d types from %s", len(types), shortName)

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

		// Basic Zip Slip protection
		if !strings.HasPrefix(fpath, filepath.Clean(packageDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		if err := extractFile(f, fpath); err != nil {
			return err
		}
	}
	return nil
}

func extractFile(f *zip.File, destPath string) error {
	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = io.Copy(outFile, rc)
	return err
}
