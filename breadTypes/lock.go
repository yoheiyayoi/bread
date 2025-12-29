package breadTypes

type Lockfile struct {
	Registry string          `toml:"registry"`
	Packages []LockedPackage `toml:"package"`
}

type LockedPackage struct {
	Name         string     `toml:"name"`
	Version      string     `toml:"version"`
	Dependencies [][]string `toml:"dependencies"`
}
