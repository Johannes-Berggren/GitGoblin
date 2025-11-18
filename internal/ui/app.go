package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type errMsg struct {
	err error
}

type tickMsg time.Time

type Model struct {
	dashboard *DashboardView
	err       error
}

func NewModel() Model {
	return Model{
		dashboard: NewDashboardView(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.dashboard.Init(),
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Only handle Ctrl+C for terminal compatibility
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

	case dashboardDataMsg:
		// Forward to dashboard
		m.dashboard, cmd = m.dashboard.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		// Forward window size to dashboard
		m.dashboard, cmd = m.dashboard.Update(msg)
		return m, cmd

	case tickMsg:
		// Auto-refresh on tick
		return m, tea.Batch(
			m.dashboard.loadData(),
			tickCmd(), // Schedule next tick
		)

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	return m, cmd
}

func (m Model) View() string {
	return m.dashboard.View()
}
