package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Johannes-Berggren/GitGoblin/internal/git"
)

type viewMode int

const (
	viewDashboard viewMode = iota
	viewBranchInput
	viewCommitFlow
)

type errMsg struct {
	err error
}

type tickMsg time.Time

type clearStatusMsg struct{}

type Model struct {
	dashboard   *DashboardView
	branchInput *BranchInputView
	commitFlow  *CommitFlowView
	viewMode    viewMode
	statusMsg   string
	statusStyle lipgloss.Style
	err         error
}

func NewModel() Model {
	return Model{
		dashboard: NewDashboardView(),
		viewMode:  viewDashboard,
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
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "n":
			// Only handle 'n' in dashboard mode
			if m.viewMode == viewDashboard {
				m.branchInput = NewBranchInputView()
				m.viewMode = viewBranchInput
				m.statusMsg = ""
				return m, m.branchInput.Init()
			}
		case "c":
			// Only handle 'c' in dashboard mode
			if m.viewMode == viewDashboard {
				m.commitFlow = NewCommitFlowView()
				m.viewMode = viewCommitFlow
				m.statusMsg = ""
				return m, m.commitFlow.Init()
			}
		}

	case branchInputDoneMsg:
		// Create the branch
		err := git.CreateBranchFromDefault(msg.name)
		m.viewMode = viewDashboard
		m.branchInput = nil
		if err != nil {
			m.statusMsg = "Error: " + err.Error()
			m.statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		} else {
			m.statusMsg = "Created branch: " + msg.name
			m.statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		}
		return m, tea.Batch(
			m.dashboard.loadData(),
			tea.Tick(time.Second*3, func(t time.Time) tea.Msg { return clearStatusMsg{} }),
		)

	case branchInputCancelMsg:
		m.viewMode = viewDashboard
		m.branchInput = nil
		return m, nil

	case commitFlowDoneMsg:
		m.viewMode = viewDashboard
		m.commitFlow = nil
		m.statusMsg = "Committed: " + msg.message
		m.statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		return m, tea.Batch(
			m.dashboard.loadData(),
			tea.Tick(time.Second*3, func(t time.Time) tea.Msg { return clearStatusMsg{} }),
		)

	case commitFlowCancelMsg:
		m.viewMode = viewDashboard
		m.commitFlow = nil
		return m, m.dashboard.loadData()

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case dashboardDataMsg:
		// Forward to dashboard
		m.dashboard, cmd = m.dashboard.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		// Forward window size to dashboard and active views
		m.dashboard, cmd = m.dashboard.Update(msg)
		if m.branchInput != nil {
			m.branchInput, _ = m.branchInput.Update(msg)
		}
		if m.commitFlow != nil {
			m.commitFlow, _ = m.commitFlow.Update(msg)
		}
		return m, cmd

	case tickMsg:
		// Auto-refresh on tick (only in dashboard mode)
		if m.viewMode == viewDashboard {
			return m, tea.Batch(
				m.dashboard.loadData(),
				tickCmd(),
			)
		}
		return m, tickCmd()

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	// Forward messages to active view
	if m.viewMode == viewBranchInput && m.branchInput != nil {
		m.branchInput, cmd = m.branchInput.Update(msg)
		return m, cmd
	}
	if m.viewMode == viewCommitFlow && m.commitFlow != nil {
		m.commitFlow, cmd = m.commitFlow.Update(msg)
		return m, cmd
	}

	return m, cmd
}

func (m Model) View() string {
	switch m.viewMode {
	case viewBranchInput:
		if m.branchInput != nil {
			return m.branchInput.View()
		}
	case viewCommitFlow:
		if m.commitFlow != nil {
			return m.commitFlow.View()
		}
	}

	// Dashboard view with optional status message
	view := m.dashboard.View()
	if m.statusMsg != "" {
		view += "\n" + m.statusStyle.Render(m.statusMsg)
	}
	return view
}
