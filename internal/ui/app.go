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
	viewBranchList
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
	branchView  *BranchView
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
			if m.viewMode == viewDashboard {
				m.branchInput = NewBranchInputView()
				m.viewMode = viewBranchInput
				m.statusMsg = ""
				return m, m.branchInput.Init()
			}
		case "c":
			if m.viewMode == viewDashboard {
				m.commitFlow = NewCommitFlowView()
				m.viewMode = viewCommitFlow
				m.statusMsg = ""
				return m, m.commitFlow.Init()
			}
		case "b":
			if m.viewMode == viewDashboard {
				m.branchView = NewBranchView()
				m.viewMode = viewBranchList
				m.statusMsg = ""
				return m, m.branchView.Init()
			}
		case "r":
			if m.viewMode == viewDashboard {
				currentBranch := m.dashboard.branch
				if currentBranch != "" {
					m.branchInput = NewBranchRenameView(currentBranch)
					m.viewMode = viewBranchInput
					m.statusMsg = ""
					return m, m.branchInput.Init()
				}
			}
		}

	case branchInputDoneMsg:
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

	case branchRenameDoneMsg:
		err := git.RenameBranch(msg.oldName, msg.newName)
		m.viewMode = viewDashboard
		m.branchInput = nil
		if err != nil {
			m.statusMsg = "Error: " + err.Error()
			m.statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		} else {
			m.statusMsg = "Renamed branch: " + msg.oldName + " → " + msg.newName
			m.statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		}
		return m, tea.Batch(
			m.dashboard.loadData(),
			tea.Tick(time.Second*3, func(t time.Time) tea.Msg { return clearStatusMsg{} }),
		)

	case branchSwitchDoneMsg:
		err := git.SwitchBranch(msg.name)
		m.viewMode = viewDashboard
		m.branchView = nil
		if err != nil {
			m.statusMsg = "Error: " + err.Error()
			m.statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		} else {
			m.statusMsg = "Switched to branch: " + msg.name
			m.statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		}
		return m, tea.Batch(
			m.dashboard.loadData(),
			tea.Tick(time.Second*3, func(t time.Time) tea.Msg { return clearStatusMsg{} }),
		)

	case branchSwitchCancelMsg:
		m.viewMode = viewDashboard
		m.branchView = nil
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
		m.dashboard, cmd = m.dashboard.Update(msg)
		return m, cmd

	case branchesLoadedMsg:
		if m.branchView != nil {
			m.branchView, cmd = m.branchView.Update(msg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.dashboard, cmd = m.dashboard.Update(msg)
		if m.branchInput != nil {
			m.branchInput, _ = m.branchInput.Update(msg)
		}
		if m.commitFlow != nil {
			m.commitFlow, _ = m.commitFlow.Update(msg)
		}
		if m.branchView != nil {
			m.branchView, _ = m.branchView.Update(msg)
		}
		return m, cmd

	case tickMsg:
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
	if m.viewMode == viewBranchList && m.branchView != nil {
		m.branchView, cmd = m.branchView.Update(msg)
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
	case viewBranchList:
		if m.branchView != nil {
			return m.branchView.View()
		}
	}

	// Dashboard view with optional status message
	view := m.dashboard.View()
	if m.statusMsg != "" {
		view += "\n" + m.statusStyle.Render(m.statusMsg)
	}
	return view
}
