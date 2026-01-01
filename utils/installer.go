package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"
	"yoheiyayoi/bread/breadTypes"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
)

type installSession struct {
	wg           sync.WaitGroup
	visited      sync.Map
	packages     sync.Map
	downloaded   sync.Map
	errors       chan error
	successCount atomic.Int32
	total        int
}

func newInstallSession(total int) *installSession {
	return &installSession{
		errors: make(chan error, 1000),
		total:  total,
	}
}

func (s *installSession) collectErrors() error {
	close(s.errors)
	for err := range s.errors {
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *installSession) storePackage(name, version string, deps [][]string) {
	key := fmt.Sprintf("%s@%s", name, version)
	s.packages.Store(key, &breadTypes.LockedPackage{
		Name:         name,
		Version:      version,
		Dependencies: deps,
	})
}

// Install downloads and links all dependencies from the manifest.
func (ic *InstallationContext) Install() error {
	start := time.Now()

	realms := []struct {
		realm Realm
		deps  map[string]string
	}{
		{RealmShared, ic.Manifest.Dependencies},
		{RealmServer, ic.Manifest.ServerDependencies},
		{RealmDev, ic.Manifest.DevDependencies},
	}

	total := countDependencies(realms)
	if total == 0 {
		log.Info("No packages to install")
		return nil
	}

	log.Info("Installing packages...")
	session := newInstallSession(total)

	if err := ic.downloadAll(realms, session); err != nil {
		return err
	}

	session.wg.Wait()

	if err := session.collectErrors(); err != nil {
		return err
	}

	if err := ic.writeLockfile(session); err != nil {
		return err
	}

	if err := ic.linkAll(realms); err != nil {
		return err
	}

	elapsed := time.Since(start)
	log.Infof("%s Installed %d packages in %.2fs [%dms]", Check, session.successCount.Load(), elapsed.Seconds(), elapsed.Milliseconds())
	return nil
}

func countDependencies(realms []struct {
	realm Realm
	deps  map[string]string
}) int {
	total := 0
	for _, r := range realms {
		total += len(r.deps)
	}
	return total
}

func (ic *InstallationContext) downloadAll(realms []struct {
	realm Realm
	deps  map[string]string
}, session *installSession) error {
	for _, r := range realms {
		if len(r.deps) == 0 {
			continue
		}

		if err := os.MkdirAll(ic.getIndexDir(r.realm), 0755); err != nil {
			return err
		}

		for name, spec := range r.deps {
			ic.installPackage(name, spec, r.realm, session)
		}
	}
	return nil
}

func (ic *InstallationContext) linkAll(realms []struct {
	realm Realm
	deps  map[string]string
}) error {
	for _, r := range realms {
		if len(r.deps) > 0 {
			if err := ic.writeRootPackageLinks(r.realm, r.deps); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ic *InstallationContext) installPackage(name, spec string, realm Realm, session *installSession) {
	pkgName, constraint := parsePackageSpec(name, spec)

	version, err := ic.resolveVersion(pkgName, constraint)
	if err != nil {
		session.errors <- err
		return
	}

	pkgID := fmt.Sprintf("%s:%s@%s", realm, pkgName, version)
	if _, exists := session.visited.LoadOrStore(pkgID, true); exists {
		return
	}

	session.wg.Add(1)
	go func() {
		defer session.wg.Done()
		ic.downloadAndProcessPackage(pkgName, version, realm, session)
	}()
}

func (ic *InstallationContext) resolveVersion(name, constraint string) (string, error) {
	// Check lockfile first
	if locked, ok := ic.Lockfile[name]; ok {
		for _, pkg := range locked {
			if MatchConstraint(pkg.Version, constraint) {
				return pkg.Version, nil
			}
		}
	}

	version, err := ResolveVersion(name, constraint)
	if err != nil {
		return "", fmt.Errorf("failed to resolve %s@%s: %w", name, constraint, err)
	}
	return version, nil
}

func (ic *InstallationContext) downloadAndProcessPackage(name, version string, realm Realm, session *installSession) {
	if err := ic.downloadPackage(name, version, realm); err != nil {
		session.errors <- err
		return
	}

	n := session.successCount.Add(1)
	fmt.Printf("%s [%d/%d] Downloaded %s@%s\n", Check, n, session.total, name, version)
	deps, err := ic.getPackageDependencies(name, version, realm)
	if err != nil {
		log.Errorf("Failed to read dependencies for %s@%s: %v", name, version, err)
		return
	}

	depsList := sortedDeps(deps)
	session.storePackage(name, version, depsList)

	for depName, depSpec := range deps {
		ic.installPackage(depName, depSpec, realm, session)
	}
}

func sortedDeps(deps map[string]string) [][]string {
	result := make([][]string, 0, len(deps))
	for name, spec := range deps {
		result = append(result, []string{name, spec})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i][0] < result[j][0]
	})
	return result
}

func (ic *InstallationContext) writeLockfile(session *installSession) error {
	packages := ic.collectLockedPackages(session)
	packages = append(packages, ic.createRootPackage())

	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Name < packages[j].Name
	})

	lockfile := breadTypes.Lockfile{
		Registry: "test",
		Packages: packages,
	}

	return ic.saveLockfile(lockfile)
}

func (ic *InstallationContext) collectLockedPackages(session *installSession) []breadTypes.LockedPackage {
	var packages []breadTypes.LockedPackage
	session.packages.Range(func(_, value interface{}) bool {
		packages = append(packages, *value.(*breadTypes.LockedPackage))
		return true
	})
	return packages
}

func (ic *InstallationContext) createRootPackage() breadTypes.LockedPackage {
	var deps [][]string

	for name, spec := range ic.Manifest.Dependencies {
		deps = append(deps, []string{name, spec})
	}
	for name, spec := range ic.Manifest.ServerDependencies {
		deps = append(deps, []string{name, spec})
	}
	for name, spec := range ic.Manifest.DevDependencies {
		deps = append(deps, []string{name, spec})
	}

	sort.Slice(deps, func(i, j int) bool {
		return deps[i][0] < deps[j][0]
	})

	return breadTypes.LockedPackage{
		Name:         ic.Manifest.Package.Name,
		Version:      ic.Manifest.Package.Version,
		Dependencies: deps,
	}
}

func (ic *InstallationContext) saveLockfile(lockfile breadTypes.Lockfile) error {
	path := filepath.Join(ic.ProjectPath, "bread.lock")

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create lockfile: %w", err)
	}
	defer f.Close()

	header := "# This file is automatically @generated by Bread.\n# It is not intended for manual editing.\n\n"
	if _, err := f.WriteString(header); err != nil {
		return fmt.Errorf("failed to write lockfile header: %w", err)
	}

	if err := toml.NewEncoder(f).Encode(lockfile); err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}

	return nil
}

// InstallSinglePackage installs a single package without recursive dependencies.
func (ic *InstallationContext) InstallSinglePackage(name, versionSpec string, realm Realm) error {
	start := time.Now()

	if err := os.MkdirAll(ic.getIndexDir(realm), 0755); err != nil {
		return err
	}

	pkgName, constraint := parsePackageSpec(name, versionSpec)

	version, err := ResolveVersion(pkgName, constraint)
	if err != nil {
		return err
	}

	if err := ic.downloadPackage(pkgName, version, realm); err != nil {
		return err
	}

	baseDir := ic.getRealmDir(realm)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	if err := ic.writeLinkFile(baseDir, pkgName, version, realm); err != nil {
		return err
	}

	elapsed := time.Since(start)
	log.Infof("%s Installed %s@%s in %.2fs [%dms]", Check, pkgName, version, elapsed.Seconds(), elapsed.Milliseconds())
	return nil
}
