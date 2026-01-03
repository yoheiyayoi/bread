package utils

import (
	"net/http"
	"os"
	"path/filepath"
	"sync"
	breadTypes "yoheiyayoi/bread/pkg/bread_type"

	"github.com/charmbracelet/log"
	"github.com/fatih/color"
)

// Types
type InstallationContext struct {
	Manifest    breadTypes.Config
	Lockfile    map[string][]breadTypes.LockedPackage // Map name -> versions
	ProjectPath string
	SharedDir   string
	ServerDir   string
	DevDir      string
	Client      *http.Client
	Packages    sync.Map
	Visited     sync.Map
}

type RealmDeps struct {
	realm string
	deps  map[string]string
}

const (
	RealmShared  string = "shared"
	RealmServer  string = "server"
	RealmDev     string = "dev"
	IndexDirName string = "_Index"
)

var (
	Check = color.GreenString("✓")
	Info  = color.BlueString("ℹ")
)

func init() {
	log.SetReportTimestamp(false)
}

func NewInstaller(projectPath string) *InstallationContext {
	var config breadTypes.Config
	if err := DecodeManifest(&config); err != nil {
		log.Error("bread.toml not found!, Run bread init first")
		return nil
	}

	// Load lockfile
	var lockfileData breadTypes.Lockfile
	LoadLockfile(&lockfileData)

	lockfileMap := make(map[string][]breadTypes.LockedPackage)

	for _, pkg := range lockfileData.Packages {
		lockfileMap[pkg.Name] = append(lockfileMap[pkg.Name], pkg)
	}

	getDir := func(configDir, defaultName string) string {
		if configDir != "" {
			return filepath.Join(projectPath, configDir)
		}
		return filepath.Join(projectPath, defaultName)
	}

	return &InstallationContext{
		Manifest:    config,
		Lockfile:    lockfileMap,
		ProjectPath: projectPath,
		SharedDir:   getDir(config.BreadConfig.PackagesDir, "Packages"),
		ServerDir:   getDir(config.BreadConfig.ServerDir, "ServerPackages"),
		DevDir:      getDir(config.BreadConfig.DevDir, "DevPackages"),
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

func (ic *InstallationContext) getRealmDir(realm string) string {
	switch realm {
	case RealmServer:
		return ic.ServerDir
	case RealmDev:
		return ic.DevDir
	default:
		return ic.SharedDir
	}
}

func (ic *InstallationContext) getIndexDir(realm string) string {
	return filepath.Join(ic.getRealmDir(realm), IndexDirName)
}
