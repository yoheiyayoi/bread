package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"yoheiyayoi/bread/breadTypes"

	"github.com/BurntSushi/toml"
)

func (ic *InstallationContext) getPackageDependencies(name, version string, realm Realm) (map[string]string, error) {
	fullName := packageIDFileName(name, version)
	shortName := getPackageName(name)
	packageDir := filepath.Join(ic.getIndexDir(realm), fullName, shortName)

	configPath := filepath.Join(packageDir, "bread.toml")
	if _, err := os.Stat(configPath); err == nil {
		var config breadTypes.Config
		if _, err := toml.DecodeFile(configPath, &config); err != nil {
			return nil, err
		}
		return config.Dependencies, nil
	}

	return nil, nil // No config found
}

func ParsePackageSpec(alias, versionSpec string) (packageName, version string) {
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
	downloadLimit <- struct{}{}
	defer func() { <-downloadLimit }()

	url := fmt.Sprintf("https://api.wally.run/v1/package-contents/%s/%s", name, version)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "bread/1.0")
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("Wally-Version", "0.3.2")

	resp, err := ic.Client.Do(req)
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
