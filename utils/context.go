package utils

import (
	"os"
	"path/filepath"
	"yoheiyayoi/bread/breadTypes"

	"github.com/BurntSushi/toml"
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
	SharedPath  *string
	ServerPath  *string
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

	// Load lockfile
	lockPath := filepath.Join(projectPath, "bread.lock")
	var lockfileData breadTypes.Lockfile
	lockfileMap := make(map[string][]breadTypes.LockedPackage)

	if _, err := toml.DecodeFile(lockPath, &lockfileData); err == nil {
		for _, pkg := range lockfileData.Packages {
			lockfileMap[pkg.Name] = append(lockfileMap[pkg.Name], pkg)
		}
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
		SharedPath:  sharedPath,
		ServerPath:  serverPath,
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
