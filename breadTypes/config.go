package breadTypes

type Config struct {
	Package            Package           `toml:"package"`
	BreadConfig        BreadConfig       `toml:"bread"`
	Dependencies       map[string]string `toml:"dependencies"`
	ServerDependencies map[string]string `toml:"server-dependencies"`
	DevDependencies    map[string]string `toml:"dev-dependencies"`
}

type Package struct {
	Name        string   `toml:"name"`
	Description string   `toml:"description"`
	Version     string   `toml:"version"`
	License     string   `toml:"license"`
	Authors     []string `toml:"authors"`
	Realm       string   `toml:"realm"`
	Registry    string   `toml:"registry"`
	Homepage    string   `toml:"homepage"`
	Repository  string   `toml:"repository"`
	Exclude     []string `toml:"exclude"`
	Private     bool     `toml:"private"`
}

type BreadConfig struct {
	PackagesDir string `toml:"shared_dir"`
	ServerDir   string `toml:"server_dir"`
	DevDir      string `toml:"dev_dir"`
}
