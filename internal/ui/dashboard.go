package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johannesberggren/gitgoblin/internal/git"
	"github.com/johannesberggren/gitgoblin/internal/models"
)

type DashboardView struct {
	branch      string
	files       []models.FileChange
	aheadCount  int
	behindCount int
	width       int
	height      int
}

func NewDashboardView() *DashboardView {
	return &DashboardView{}
}

type dashboardDataMsg struct {
	branch      string
	files       []models.FileChange
	aheadCount  int
	behindCount int
}

func (d *DashboardView) Init() tea.Cmd {
	return d.loadData()
}

func (d *DashboardView) loadData() tea.Cmd {
	return func() tea.Msg {
		branch, err := git.GetCurrentBranch()
		if err != nil {
			branch = "unknown"
		}

		files, err := git.GetWorkingTreeStatus()
		if err != nil {
			files = []models.FileChange{}
		}

		// Get upstream status
		branches, err := git.GetBranches()
		ahead, behind := 0, 0
		if err == nil {
			for _, b := range branches {
				if b.IsCurrent && b.Upstream != "" {
					ahead, behind = parseUpstream(b.Upstream)
					break
				}
			}
		}

		return dashboardDataMsg{branch, files, ahead, behind}
	}
}

func (d *DashboardView) Update(msg tea.Msg) (*DashboardView, tea.Cmd) {
	switch msg := msg.(type) {
	case dashboardDataMsg:
		d.branch = msg.branch
		d.files = msg.files
		d.aheadCount = msg.aheadCount
		d.behindCount = msg.behindCount

	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
	}

	return d, nil
}

func (d *DashboardView) View() string {
	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		Align(lipgloss.Center).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("170")).
		Width(40).
		MarginTop(1).
		MarginBottom(2)

	header := headerStyle.Render("🧙 GitGoblin")

	// Branch box
	branchStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("cyan")).
		Align(lipgloss.Center).
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("cyan")).
		Width(30).
		Padding(0, 1).
		MarginBottom(2)

	branchBox := branchStyle.Render(fmt.Sprintf("Branch: %s", d.branch))

	// Remote status (only if diverged)
	var remoteStatus string
	if d.aheadCount > 0 || d.behindCount > 0 {
		remoteStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("yellow")).
			Bold(true).
			Align(lipgloss.Center).
			MarginBottom(2)

		parts := []string{}
		if d.aheadCount > 0 {
			parts = append(parts, fmt.Sprintf("↑%d ahead", d.aheadCount))
		}
		if d.behindCount > 0 {
			parts = append(parts, fmt.Sprintf("↓%d behind", d.behindCount))
		}
		remoteStatus = remoteStyle.Render("⚠ origin: " + strings.Join(parts, ", "))
	}

	// Files or clean message
	var content string
	if len(d.files) == 0 {
		// Clean state - big prominent message
		checkmarks := lipgloss.NewStyle().
			Foreground(lipgloss.Color("green")).
			Align(lipgloss.Center).
			Render("✓ ✓ ✓ ✓ ✓")

		message := lipgloss.NewStyle().
			Foreground(lipgloss.Color("green")).
			Bold(true).
			Align(lipgloss.Center).
			Render("NO UNCOMMITTED CHANGES")

		content = lipgloss.JoinVertical(
			lipgloss.Center,
			checkmarks,
			"",
			message,
			"",
			checkmarks,
		)
	} else {
		// Dirty state - show file list
		fileCount := len(d.files)
		displayCount := fileCount
		if displayCount > 10 {
			displayCount = 10
		}

		fileBoxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("yellow")).
			Padding(0, 1).
			Width(40)

		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("yellow")).
			Bold(true)

		fileStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("white"))

		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("yellow")).
			Bold(true)

		var fileList strings.Builder
		fileList.WriteString(titleStyle.Render(fmt.Sprintf("%d Uncommitted File(s)", fileCount)) + "\n")

		for i := 0; i < displayCount; i++ {
			file := d.files[i]
			status := statusStyle.Render(file.DisplayStatus())
			path := fileStyle.Render(file.Path)
			fileList.WriteString(fmt.Sprintf(" %s  %s\n", status, path))
		}

		if fileCount > 10 {
			moreStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Italic(true)
			fileList.WriteString(moreStyle.Render(fmt.Sprintf(" ...and %d more", fileCount-10)))
		}

		content = fileBoxStyle.Render(fileList.String())
	}

	// Combine all elements
	var parts []string
	parts = append(parts, header)
	parts = append(parts, branchBox)
	if remoteStatus != "" {
		parts = append(parts, remoteStatus)
	}
	parts = append(parts, content)

	dashboard := lipgloss.JoinVertical(lipgloss.Center, parts...)

	// Center everything in the terminal
	fullStyle := lipgloss.NewStyle().
		Width(d.width).
		Height(d.height).
		Align(lipgloss.Center, lipgloss.Center)

	return fullStyle.Render(dashboard)
}

// parseUpstream extracts ahead/behind counts from upstream string
// e.g., "origin/main: ahead 2" or "origin/main: ahead 2, behind 1"
func parseUpstream(upstream string) (ahead, behind int) {
	ahead, behind = 0, 0

	// Format: "origin/main: ahead 2, behind 1"
	if !strings.Contains(upstream, ":") {
		return
	}

	parts := strings.Split(upstream, ":")
	if len(parts) < 2 {
		return
	}

	status := parts[1]

	// Parse "ahead X"
	if strings.Contains(status, "ahead") {
		fmt.Sscanf(status, "%*s%d", &ahead)
	}

	// Parse "behind X"
	if strings.Contains(status, "behind") {
		var dummy int
		if strings.Contains(status, "ahead") {
			// Format: "ahead 2, behind 1"
			fmt.Sscanf(status, "%*s%d%*s%*s%d", &dummy, &behind)
		} else {
			// Format: "behind 1"
			fmt.Sscanf(status, "%*s%d", &behind)
		}
	}

	return
}
