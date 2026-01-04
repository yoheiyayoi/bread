package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	breadTypes "yoheiyayoi/bread/pkg/bread_type"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// Types
type model struct {
	packages    [][]string
	completed   int
	width       int
	height      int
	spinner     spinner.Model
	progress    progress.Model
	done        bool
	lastPkg     string
	ic          *InstallationContext
	resultsChan chan string
}

type pkgFinishedMsg string

// Local
var (
	currentPkgNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("211"))
	doneStyle           = lipgloss.NewStyle().Margin(1, 2)
	checkMark           = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
)

// Functions

// Progrss bar session (code from bubbletea examples with some edit skibidi)
func newModel(pkgs [][]string, ic *InstallationContext) model {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	return model{
		packages:    pkgs,
		spinner:     s,
		progress:    p,
		ic:          ic,
		resultsChan: make(chan string),
	}
}

func waitForActivity(res chan string) tea.Cmd {
	return func() tea.Msg {
		return pkgFinishedMsg(<-res)
	}
}

func (m model) Init() tea.Cmd {
	const numWorkers = 10 // I saw on wally they use `.worker_threads(50)` but 10 should be enough
	jobs := make(chan []string, len(m.packages))

	for _, pkg := range m.packages {
		jobs <- pkg
	}
	close(jobs)

	for range numWorkers {
		go func() {
			for pkg := range jobs {
				fullSpec := pkg[0]
				realm := pkg[1]

				parts := strings.Split(fullSpec, "@")
				name, version := parts[0], parts[1]

				if _, err := m.ic.installPackage(name, version, realm); err != nil {
					log.Errorf("Failed to install %s: %v", fullSpec, err)
				}

				m.resultsChan <- fullSpec
				// m.resultsChan <- name
			}
		}()
	}

	return tea.Batch(m.spinner.Tick, waitForActivity(m.resultsChan))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		}

	case pkgFinishedMsg:
		m.completed++
		m.lastPkg = string(msg)

		progressCmd := m.progress.SetPercent(float64(m.completed) / float64(len(m.packages)))
		logLine := tea.Printf("%s %s", checkMark, m.lastPkg)

		if m.completed >= len(m.packages) {
			m.done = true
			return m, tea.Sequence(logLine, tea.Quit)
		}

		return m, tea.Batch(
			progressCmd,
			logLine,
			waitForActivity(m.resultsChan),
		)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		newProgModel, cmd := m.progress.Update(msg)
		if newProgModel, ok := newProgModel.(progress.Model); ok {
			m.progress = newProgModel
		}
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	n := len(m.packages)
	w := lipgloss.Width(fmt.Sprintf("%d", n))

	if m.done {
		return doneStyle.Render(fmt.Sprintf("Done! Installed %d packages.\n", n))
	}

	pkgCount := fmt.Sprintf(" %*d/%*d", w, m.completed, w, n)

	spin := m.spinner.View() + " "
	prog := m.progress.View()
	cellsAvail := max(0, m.width-lipgloss.Width(spin+prog+pkgCount))

	infoText := "Installing packages..."
	if m.lastPkg != "" {
		infoText = "Installed " + currentPkgNameStyle.Render(m.lastPkg)
	}
	info := lipgloss.NewStyle().MaxWidth(cellsAvail).Render(infoText)

	return spin + info + " " + prog + pkgCount
}

// End progress bar section

func (ic *InstallationContext) InstallAll() error {
	log.Info("Checking dependencies...")
	realms := []RealmDeps{
		{RealmShared, ic.Manifest.Dependencies},
		{RealmServer, ic.Manifest.ServerDependencies},
		{RealmDev, ic.Manifest.DevDependencies},
	}

	var flatPackages [][]string
	for _, realm := range realms {
		for name, spec := range realm.deps {
			flatPackages = append(flatPackages, []string{fmt.Sprintf("%s@%s", name, spec), realm.realm})
		}
	}

	total := len(flatPackages)
	if total == 0 {
		log.Warn("No packages to install")
		return nil
	}

	return RunPackageInstallation(ic, flatPackages)
}

func RunPackageInstallation(ic *InstallationContext, packagesArray [][]string) error {
	if len(packagesArray) == 0 {
		return nil
	}

	total := len(packagesArray)
	start := time.Now()
	log.Info("Installing packages...")

	p := tea.NewProgram(newModel(packagesArray, ic))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running installation UI: %v", err)
	}

	realms := []RealmDeps{
		{RealmShared, ic.Manifest.Dependencies},
		{RealmServer, ic.Manifest.ServerDependencies},
		{RealmDev, ic.Manifest.DevDependencies},
	}

	if err := ic.linkAll(realms); err != nil {
		return fmt.Errorf("failed to write link files: %w", err)
	}

	if err := ic.writeLockfile(); err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}

	elapsed := time.Since(start)
	log.Infof("Done! Installed %d package(s) in %.2fs", total, elapsed.Seconds())
	return nil
}

func (ic *InstallationContext) installPackage(name, spec string, realm string) (string, error) {
	pkgName, constraint := ParsePackageSpec(name, spec)

	version, err := ic.resolveVersion(pkgName, constraint)
	if err != nil {
		return "", fmt.Errorf("failed to resolve %s@%s: %w", name, constraint, err)
	}

	pkgID := fmt.Sprintf("%s:%s@%s", realm, pkgName, version)
	if _, exists := ic.Visited.LoadOrStore(pkgID, true); exists {
		return version, nil
	}

	if err := ic.downloadPackage(pkgName, version, realm); err != nil {
		return "", fmt.Errorf("failed to install %s@%s: %w", pkgName, version, err)
	}

	var config breadTypes.Config
	packageDirName := packageIDFileName(pkgName, version)
	targetDir := filepath.Join(ic.getIndexDir(realm), packageDirName)
	finalDir := filepath.Join(targetDir, getPackageName(pkgName))

	manifestPath := filepath.Join(finalDir, "wally.toml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		manifestPath = filepath.Join(finalDir, "bread.toml")
	}

	if _, err := os.Stat(manifestPath); err == nil {
		if _, err := toml.DecodeFile(manifestPath, &config); err != nil {
			return "", fmt.Errorf("failed to decode manifest for %s: %w", pkgName, err)
		}
	}

	deps := config.Dependencies
	depsList := sortedDeps(deps)
	ic.storePackage(name, version, depsList)

	resolvedDeps := make(map[string]string)
	for depName, depSpec := range deps {
		ver, err := ic.installPackage(depName, depSpec, realm)
		if err != nil {
			return "", fmt.Errorf("Failed to install %s: %v", depName, err)
		}
		resolvedDeps[depName] = ver
	}

	if err := ic.writeNestedPackageLinks(targetDir, resolvedDeps, realm); err != nil {
		return "", fmt.Errorf("failed to link nested deps for %s: %w", pkgName, err)
	}

	return version, nil
}

func (ic *InstallationContext) resolveVersion(name, constraint string) (string, error) {
	// Check lockfile first
	if locked, ok := ic.Lockfile[name]; ok {
		for _, pkg := range locked {
			if pass, err := MatchConstraint(pkg.Version, constraint); err == nil && pass {
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

func (ic *InstallationContext) linkAll(realms []RealmDeps) error {
	for _, r := range realms {
		if len(r.deps) > 0 {
			if err := ic.writeRootPackageLinks(r.realm, r.deps); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ic *InstallationContext) storePackage(name, version string, deps [][]string) {
	key := fmt.Sprintf("%s@%s", name, version)
	ic.Packages.Store(key, &breadTypes.LockedPackage{
		Name:         name,
		Version:      version,
		Dependencies: deps,
	})
}

func (ic *InstallationContext) writeLockfile() error {
	packages := ic.collectLockedPackages()
	packages = append(packages, ic.createRootPackage())

	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Name < packages[j].Name
	})

	lockfile := breadTypes.Lockfile{
		Registry: "test",
		Packages: packages,
	}

	return SaveLockfile(lockfile)
}

func (ic *InstallationContext) collectLockedPackages() []breadTypes.LockedPackage {
	var packages []breadTypes.LockedPackage
	ic.Packages.Range(func(_, value any) bool {
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
