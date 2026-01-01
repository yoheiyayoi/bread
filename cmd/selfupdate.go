package cmd

import (
	"fmt"
	"yoheiyayoi/bread/config"

	"github.com/blang/semver"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/fatih/color"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
)

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

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Update Bread package manager to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(initialUpdateModel())

		v := semver.MustParse(config.Version)
		repoText := config.RepoOwner + "/" + config.RepoName

		go func() {
			latest, err := selfupdate.UpdateSelf(v, repoText)
			p.Send(updateResultMsg{release: latest, err: err})
		}()

		finalModel, err := p.Run()
		if err != nil {
			log.Fatalf("UI error: %v", err)
		}

		m := finalModel.(updateModel)
		if m.err != nil {
			log.Errorf("Self-update failed: %s", m.err)
			return
		}

		if m.result.Version.Equals(v) {
			log.Infof("Bread is already up to date (v%s)", config.Version)
		} else {
			checkIcon := color.GreenString("✓")
			linebar := color.GreenString("┃  ")

			fmt.Printf("%s %sSuccessfully updated to version: v%s\n", checkIcon, linebar, m.result.Version)
			fmt.Printf("%s\nRelease notes: \n%s", linebar, m.result.ReleaseNotes)
		}
	},
}

func init() {
	rootCmd.AddCommand(selfUpdateCmd)
}
