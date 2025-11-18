package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johannesberggren/gitgoblin/internal/git"
)

type errMsg struct {
	err error
}

type statusMsg struct {
	branch     string
	hasChanges bool
}

type viewMode int

const (
	viewStaging viewMode = iota
	viewBranches
	viewCommit
)

type Model struct {
	width        int
	height       int
	mode         viewMode
	stagingView  *StagingView
	branchView   *BranchView
	commitView   *CommitView
	err          error
	branch       string
	hasChanges   bool
}

func NewModel() Model {
	return Model{
		stagingView: NewStagingView(),
		branchView:  NewBranchView(),
		mode:        viewStaging, // Will be determined on init
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.checkStatus(),
		m.stagingView.Init(),
		m.branchView.Init(),
	)
}

func (m Model) checkStatus() tea.Cmd {
	return func() tea.Msg {
		branch, err := git.GetCurrentBranch()
		if err != nil {
			branch = "unknown"
		}

		hasChanges, err := git.HasUncommittedChanges()
		if err != nil {
			hasChanges = false
		}

		return statusMsg{branch, hasChanges}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "r":
			// Refresh all views
			return m, tea.Batch(
				m.checkStatus(),
				m.stagingView.loadFiles(),
				m.branchView.loadBranches(),
			)

		case "b":
			// Switch to branch view
			if m.mode != viewCommit {
				m.mode = viewBranches
			}
			return m, nil

		case "s":
			// Switch to staging view
			if m.mode != viewCommit {
				m.mode = viewStaging
			}
			return m, nil

		case "c":
			// Open commit editor (only from staging view)
			if m.mode == viewStaging && m.stagingView.HasStagedFiles() {
				stagedCount := 0
				for _, f := range m.stagingView.files {
					if f.IsStaged {
						stagedCount++
					}
				}
				m.commitView = NewCommitView(stagedCount)
				m.mode = viewCommit
				return m, m.commitView.Init()
			}
		}

		// Handle commit view specially
		if m.mode == viewCommit {
			// Check for cancel
			if msg.String() == "esc" {
				m.mode = viewStaging
				return m, m.stagingView.loadFiles()
			}
		}

	case statusMsg:
		m.branch = msg.branch
		m.hasChanges = msg.hasChanges

		// Smart start screen: show staging if dirty, branches if clean
		if msg.hasChanges {
			m.mode = viewStaging
		} else {
			m.mode = viewBranches
		}

	case commitSuccessMsg:
		// Commit successful, return to staging
		m.mode = viewStaging
		return m, tea.Batch(m.checkStatus(), m.stagingView.loadFiles())

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Update active view
	switch m.mode {
	case viewStaging:
		m.stagingView, cmd = m.stagingView.Update(msg)
		cmds = append(cmds, cmd)

	case viewBranches:
		m.branchView, cmd = m.branchView.Update(msg)
		cmds = append(cmds, cmd)

	case viewCommit:
		if m.commitView != nil {
			m.commitView, cmd = m.commitView.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	// Header
	header := m.renderHeader()

	// Content based on mode
	var content string
	switch m.mode {
	case viewStaging:
		content = m.stagingView.View()
	case viewBranches:
		content = m.branchView.View()
	case viewCommit:
		if m.commitView != nil {
			content = m.commitView.View()
		}
	}

	// Footer
	footer := m.renderFooter()

	// Combine
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
}

func (m Model) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		MarginRight(2)

	branchStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("green")).
		Bold(true)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	dirtyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("yellow"))

	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238"))

	title := titleStyle.Render("🧙 GitGoblin")

	status := "(clean)"
	if m.hasChanges {
		status = dirtyStyle.Render("(uncommitted changes)")
	}

	branchInfo := branchStyle.Render(m.branch) + " " + statusStyle.Render(status)

	headerLine := lipgloss.JoinHorizontal(lipgloss.Top, title, branchInfo)
	divider := dividerStyle.Render(strings.Repeat("─", m.width))

	return lipgloss.JoinVertical(lipgloss.Left, headerLine, divider)
}

func (m Model) renderFooter() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238"))

	var keys []string
	switch m.mode {
	case viewStaging:
		keys = []string{
			"space: stage/unstage",
			"a: stage all",
			"d: toggle diff",
			"c: commit",
			"b: branches",
			"r: refresh",
			"q: quit",
		}
	case viewBranches:
		keys = []string{
			"j/k: navigate",
			"s: staging",
			"r: refresh",
			"q: quit",
		}
	case viewCommit:
		keys = []string{
			"ctrl+d: commit",
			"esc: cancel",
		}
	}

	divider := dividerStyle.Render(strings.Repeat("─", m.width))
	helpText := helpStyle.Render(strings.Join(keys, " • "))

	return lipgloss.JoinVertical(lipgloss.Left, divider, helpText)
}
