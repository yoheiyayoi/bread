package utils

import (
	"fmt"
	"strings"

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
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	pkgNameStyle  = lipgloss.NewStyle().Foreground(highlight).Bold(true)
	versionStyle  = lipgloss.NewStyle().Foreground(subtle)
	checkMark     = lipgloss.NewStyle().Foreground(special).SetString("✓")
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(highlight).MarginBottom(1)
	statusStyle   = lipgloss.NewStyle().Foreground(subtle).Italic(true)
	progressStyle = lipgloss.NewStyle().Foreground(highlight)
	doneStyle     = lipgloss.NewStyle().Foreground(special).Bold(true).MarginTop(1)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

func initialModel(installChan chan tea.Msg) model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(highlight)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
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
		m.progress.Width = min(msg.Width-6, 60)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.quitting = true
			return m, tea.Quit
		}
	case pkgInstalledMsg:
		m.packages = append(m.packages, msg.name)
		m.current = msg.current
		m.total = msg.total
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
	if m.quitting {
		return "\n" + errorStyle.Render("✗ Installation cancelled") + "\n\n"
	}

	var b strings.Builder
	b.WriteString("\n")

	if m.done {
		// Show completion summary
		b.WriteString(doneStyle.Render("✓ Installation complete!"))
		b.WriteString("\n\n")

		if len(m.packages) > 0 {
			b.WriteString(fmt.Sprintf("Installed %d package%s:", len(m.packages), pluralize(len(m.packages))))
			b.WriteString("\n\n")

			// Show recently installed packages (last 5)
			start := max(0, len(m.packages)-5)
			for i := start; i < len(m.packages); i++ {
				fmt.Fprintf(&b, "  %s %s\n", checkMark, m.packages[i])
			}

			if len(m.packages) > 5 {
				b.WriteString(fmt.Sprintf("\n  ... and %d more\n", len(m.packages)-5))
			}
		}

		b.WriteString("\n")
		return b.String()
	}

	// Active installation view
	b.WriteString(headerStyle.Render("Installing packages"))
	b.WriteString("\n\n")

	// Show last 3 installed packages
	if len(m.packages) > 0 {
		start := max(0, len(m.packages)-3)
		for i := start; i < len(m.packages); i++ {
			fmt.Fprintf(&b, "  %s %s\n", checkMark, m.packages[i])
		}
		b.WriteString("\n")
	}

	// Progress bar with spinner
	fmt.Fprintf(&b, "  %s %s", m.spinner.View(), progressStyle.Render(m.progress.View()))
	b.WriteString("\n")

	// Status line
	var status string
	if m.total > 0 {
		status = fmt.Sprintf("%d of %d package%s", m.current, m.total, pluralize(m.total))
	} else {
		status = "Resolving dependencies..."
	}
	b.WriteString(statusStyle.Render(fmt.Sprintf("  %s", status)))
	b.WriteString("\n\n")

	return b.String()
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
