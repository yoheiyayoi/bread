package breadTypes

type Config struct {
	Package            Package           `toml:"package"`
	BreadConfig        BreadConfig       `toml:"bread"`
	Dependencies       map[string]string `toml:"dependencies"`
	ServerDependencies map[string]string `toml:"server-dependencies"`
	DevDependencies    map[string]string `toml:"dev-dependencies"`
}

type Package struct {
	Name     string `toml:"name"`
	Version  string `toml:"version"`
	Registry string `toml:"registry"`
	Realm    string `toml:"realm"`
}

type BreadConfig struct {
	PackagesDir string `toml:"shared_dir"`
	ServerDir   string `toml:"server_dir"`
	DevDir      string `toml:"dev_dir"`
}
