package utils

import (
	"fmt"
	"yoheiyayoi/bread/pkg/config"

	"github.com/blang/semver"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/fatih/color"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

const gitRepo = config.RepoOwner + "/" + config.RepoName

var (
	lineBar   = color.GreenString("┃  ")
	checkIcon = color.GreenString("✓")
	infoIcon  = color.BlueString("ℹ")
)

func CheckForUpdates() (isLatest bool, version string, err error) {
	latest, found, err := selfupdate.DetectLatest(gitRepo)
	if err != nil {
		return false, "", fmt.Errorf("error occurred while detecting version: %w", err)
	}

	ver := semver.MustParse(config.Version)
	if !found || latest.Version.LTE(ver) {
		return true, config.Version, nil
	}

	return false, latest.Version.String(), nil
}

func DoSelfUpdate() error {
	p := tea.NewProgram(initialUpdateModel())

	ver := semver.MustParse(config.Version)
	go func() {
		latest, err := selfupdate.UpdateSelf(ver, gitRepo)
		p.Send(updateResultMsg{release: latest, err: err})
	}()

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	m := finalModel.(updateModel)
	if m.err != nil {
		return fmt.Errorf("Self-update failed: %w", m.err)
	}

	if m.result.Version.EQ(ver) {
		log.Infof("🥖 Bread is already up to date (v%s)", config.Version)
	} else {
		fmt.Printf("\n%s%s Successfully updated to version: v%s\n", lineBar, checkIcon, m.result.Version)
		fmt.Printf("%sRelease note: %s\n", lineBar, m.result.URL)
	}

	return nil
}

type updateResultMsg struct {
	release *selfupdate.Release
	err     error
}

type updateModel struct {
	spinner  spinner.Model
	loading  bool
	result   *selfupdate.Release
	err      error
	quitting bool
}

func initialUpdateModel() updateModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	return updateModel{spinner: s, loading: true}
}

func (m updateModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m updateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			m.quitting = true
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case updateResultMsg:
		m.loading = false
		m.result = msg.release
		m.err = msg.err
		return m, tea.Quit
	}
	return m, nil
}

func (m updateModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n ❌ Error: %v\n\n", m.err)
	}
	if m.loading {
		return fmt.Sprintf("\n %s Checking for updates...\n\n", m.spinner.View())
	}
	return ""
}
