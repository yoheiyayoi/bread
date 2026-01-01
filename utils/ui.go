package utils

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// UI Model
type model struct {
	packages    []string
	index       int
	width       int
	height      int
	spinner     spinner.Model
	progress    progress.Model
	done        bool
	installChan chan tea.Msg
	quitting    bool
	current     int
	total       int
}

type pkgInstalledMsg struct {
	name    string
	current int
	total   int
}
type installFinishedMsg struct{ err error }

var (
	pkgNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("211"))
	doneStyle    = lipgloss.NewStyle().Margin(1, 2)
	checkMark    = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
)

func initialModel(installChan chan tea.Msg) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(25),
		progress.WithoutPercentage(),
	)
	return model{
		spinner:     s,
		progress:    p,
		installChan: installChan,
		packages:    make([]string, 0),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		waitForActivity(m.installChan),
	)
}

func waitForActivity(sub chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.progress.Width = min(msg.Width-4, 50)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.quitting = true
			return m, tea.Quit
		}
	case pkgInstalledMsg:
		m.packages = append(m.packages, msg.name)
		m.current = msg.current // Store current
		m.total = msg.total     // Store total
		// Calculate progress
		if msg.total > 0 {
			prog := float64(msg.current) / float64(msg.total)
			return m, tea.Batch(
				m.progress.SetPercent(prog),
				waitForActivity(m.installChan),
			)
		}
		return m, waitForActivity(m.installChan)
	case installFinishedMsg:
		m.done = true
		if msg.err != nil {
			return m, tea.Quit
		}
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case progress.FrameMsg:
		newModel, cmd := m.progress.Update(msg)
		if newModel, ok := newModel.(progress.Model); ok {
			m.progress = newModel
		}
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	s := ""

	if m.done {
		s += "\n"
		for _, pkg := range m.packages {
			s += fmt.Sprintf(" %s %s\n", checkMark, pkg)
		}
		s += doneStyle.Render(fmt.Sprintf("Done! Installed %d packages.\n", len(m.packages)))
		return s
	}

	if m.quitting {
		return "Installation cancelled.\n"
	}

	// Removed the top spinner line since it's moved below
	s += "\n"

	for _, pkg := range m.packages {
		s += fmt.Sprintf(" %s %s\n", checkMark, pkg)
	}

	s += fmt.Sprintf("\n %s Installing %s [%d/%d]\n", m.spinner.View(), m.progress.View(), m.current, m.total)

	return s
}
