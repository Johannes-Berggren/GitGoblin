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
	branch string
	status string
}

type Model struct {
	width      int
	height     int
	graphView  *GraphView
	err        error
	branch     string
	statusText string
}

func NewModel() Model {
	return Model{
		graphView: NewGraphView(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.graphView.Init(),
		m.loadStatus(),
	)
}

func (m Model) loadStatus() tea.Cmd {
	return func() tea.Msg {
		branch, err := git.GetCurrentBranch()
		if err != nil {
			branch = "unknown"
		}
		status, err := git.GetStatus()
		if err != nil {
			status = "unknown"
		}
		return statusMsg{branch, status}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			// Refresh
			return m, tea.Batch(
				m.graphView.loadCommits(),
				m.loadStatus(),
			)
		}

	case statusMsg:
		m.branch = msg.branch
		m.statusText = msg.status

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Update the graph view
	m.graphView, cmd = m.graphView.Update(msg)

	return m, cmd
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	// Header
	header := m.renderHeader()

	// Graph view
	graphContent := m.graphView.View()

	// Footer
	footer := m.renderFooter()

	// Combine with proper spacing
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		graphContent,
		footer,
	)

	return content
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

	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238"))

	title := titleStyle.Render("🧙 GitGoblin")
	branchInfo := branchStyle.Render(m.branch) + " " + statusStyle.Render(fmt.Sprintf("(%s)", m.statusText))

	headerLine := lipgloss.JoinHorizontal(lipgloss.Top, title, branchInfo)
	divider := dividerStyle.Render(strings.Repeat("─", m.width))

	return lipgloss.JoinVertical(lipgloss.Left, headerLine, divider)
}

func (m Model) renderFooter() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238"))

	keys := []string{
		"j/k: navigate",
		"g/G: top/bottom",
		"r: refresh",
		"q: quit",
	}

	divider := dividerStyle.Render(strings.Repeat("─", m.width))
	helpText := helpStyle.Render(strings.Join(keys, " • "))

	return lipgloss.JoinVertical(lipgloss.Left, divider, helpText)
}
