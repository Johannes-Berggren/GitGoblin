package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johannesberggren/gitgoblin/internal/git"
	"github.com/johannesberggren/gitgoblin/internal/models"
)

type DashboardView struct {
	branch         string
	files          []models.FileChange
	aheadCount     int
	behindCount    int
	lastCommitTime time.Time
	linesAdded     int
	linesDeleted   int
	fileStats      map[string][2]int
	width          int
	height         int
}

func NewDashboardView() *DashboardView {
	return &DashboardView{}
}

type dashboardDataMsg struct {
	branch         string
	files          []models.FileChange
	aheadCount     int
	behindCount    int
	lastCommitTime time.Time
	linesAdded     int
	linesDeleted   int
	fileStats      map[string][2]int
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

		// Get last commit time
		lastCommitTime, err := git.GetLastCommitTime()
		if err != nil {
			lastCommitTime = time.Time{}
		}

		// Get line stats (per-file)
		fileStats, err := git.GetLineStats()
		if err != nil {
			fileStats = make(map[string][2]int)
		}

		// Calculate totals for status box
		linesAdded := 0
		linesDeleted := 0
		for _, stats := range fileStats {
			linesAdded += stats[0]
			linesDeleted += stats[1]
		}

		return dashboardDataMsg{branch, files, ahead, behind, lastCommitTime, linesAdded, linesDeleted, fileStats}
	}
}

func (d *DashboardView) Update(msg tea.Msg) (*DashboardView, tea.Cmd) {
	switch msg := msg.(type) {
	case dashboardDataMsg:
		d.branch = msg.branch
		d.files = msg.files
		d.aheadCount = msg.aheadCount
		d.behindCount = msg.behindCount
		d.lastCommitTime = msg.lastCommitTime
		d.linesAdded = msg.linesAdded
		d.linesDeleted = msg.linesDeleted
		d.fileStats = msg.fileStats

	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
	}

	return d, nil
}

// renderStatusBox creates a bordered box with development metrics
func (d *DashboardView) renderStatusBox() string {
	// Calculate time since last commit
	var timeSinceCommit string
	if !d.lastCommitTime.IsZero() {
		duration := time.Since(d.lastCommitTime)
		if duration.Hours() < 1 {
			timeSinceCommit = fmt.Sprintf("%.0fm ago", duration.Minutes())
		} else if duration.Hours() < 24 {
			timeSinceCommit = fmt.Sprintf("%.1fh ago", duration.Hours())
		} else {
			days := duration.Hours() / 24
			timeSinceCommit = fmt.Sprintf("%.1fd ago", days)
		}
	} else {
		timeSinceCommit = "n/a"
	}

	// Create metrics display with emoji icons
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("cyan")).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("white"))

	metrics := []string{
		fmt.Sprintf("📁 %s %s", labelStyle.Render("Files Changed:"), valueStyle.Render(fmt.Sprintf("%d", len(d.files)))),
		fmt.Sprintf("⏰ %s %s", labelStyle.Render("Last Commit:"), valueStyle.Render(timeSinceCommit)),
		fmt.Sprintf("⬆️  %s %s", labelStyle.Render("Commits Ahead:"), valueStyle.Render(fmt.Sprintf("%d", d.aheadCount))),
		fmt.Sprintf("📊 %s %s", labelStyle.Render("Lines:"), valueStyle.Render(fmt.Sprintf("+%d/-%d", d.linesAdded, d.linesDeleted))),
	}

	content := strings.Join(metrics, "\n")

	// Create bordered box with subtle colors
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")). // Subtle gray instead of bright magenta
		Padding(1, 2).
		MarginLeft(5).
		MarginBottom(1)

	return boxStyle.Render(content)
}

func (d *DashboardView) View() string {
	// Handle case where terminal size isn't set yet
	if d.width == 0 || d.height == 0 {
		return "Loading..."
	}

	// Branch as large ASCII art (top)
	branchAscii := d.renderBranchAscii()

	// Remote status (only if behind origin) - styled alert box
	var remoteStatus string
	if d.behindCount > 0 {
		warningTextStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("yellow")).
			Bold(true)

		warningBoxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("214")). // Orange border
			Background(lipgloss.Color("58")).        // Subtle dark orange background
			Padding(0, 2).
			MarginBottom(1).
			MarginLeft(5)

		warningText := warningTextStyle.Render(fmt.Sprintf("⚠  Behind origin: ↓%d", d.behindCount))
		remoteStatus = warningBoxStyle.Render(warningText)
	}

	// Status box with metrics
	statusBox := d.renderStatusBox()

	// Main content area
	var content string
	if len(d.files) == 0 {
		// Clean state - centered message
		checkmarks := lipgloss.NewStyle().
			Foreground(lipgloss.Color("green")).
			Align(lipgloss.Center).
			Render("✓ ✓ ✓ ✓ ✓")

		message := lipgloss.NewStyle().
			Foreground(lipgloss.Color("green")).
			Bold(true).
			Align(lipgloss.Center).
			Render("NO UNCOMMITTED CHANGES")

		content = lipgloss.NewStyle().
			Width(d.width).
			Height(d.height - 15). // Leave space for branch and logo
			Align(lipgloss.Center, lipgloss.Center).
			Render(lipgloss.JoinVertical(
				lipgloss.Center,
				checkmarks,
				"",
				message,
				"",
				checkmarks,
			))
	} else {
		// Dirty state - full width file list, ALL files
		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("cyan")).
			Bold(true)

		fileStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("white"))

		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("yellow")).
			Bold(true)

		// Styles for line stats
		addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("green")).Bold(true)
		deletedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("red")).Bold(true)

		// Build file list content
		var fileList strings.Builder

		// Show ALL files (no limit)
		for _, file := range d.files {
			status := statusStyle.Render(file.DisplayStatus())
			path := fileStyle.Render(file.Path)

			// Get line stats for this file
			var statsText string
			if stats, ok := d.fileStats[file.Path]; ok {
				added := stats[0]
				deleted := stats[1]
				if added > 0 || deleted > 0 {
					addText := addedStyle.Render(fmt.Sprintf("+%d", added))
					delText := deletedStyle.Render(fmt.Sprintf("-%d", deleted))
					statsText = fmt.Sprintf(" (%s/%s)", addText, delText)
				}
			}

			fileList.WriteString(fmt.Sprintf(" %s  %s%s\n", status, path, statsText))
		}

		// Wrap file list in bordered box
		fileBoxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2).
			MarginLeft(5).
			MarginTop(1)

		// Add title above the box
		title := titleStyle.Render(fmt.Sprintf("📄 %d Uncommitted File(s)", len(d.files)))
		boxContent := fileBoxStyle.Render(strings.TrimRight(fileList.String(), "\n"))

		content = lipgloss.JoinVertical(lipgloss.Left, "", title, boxContent)
	}

	// Create subtle divider
	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MarginLeft(5)
	divider := dividerStyle.Render("─────────────────────────────────────────")

	// Build top section (branch + remote + status box)
	var topSection string
	if remoteStatus != "" {
		topSection = lipgloss.JoinVertical(lipgloss.Left, branchAscii, "", remoteStatus, "", statusBox, "", divider)
	} else {
		topSection = lipgloss.JoinVertical(lipgloss.Left, branchAscii, "", statusBox, "", divider)
	}

	// Logo in bottom right
	logo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Render("🧙 GitGoblin")

	// Combine everything
	mainContent := lipgloss.JoinVertical(lipgloss.Left, topSection, content)

	// Position logo at bottom right
	mainHeight := strings.Count(mainContent, "\n") + 1
	bottomPadding := d.height - mainHeight - 2
	if bottomPadding < 0 {
		bottomPadding = 0
	}

	// Add padding to push logo down
	paddedContent := mainContent + strings.Repeat("\n", bottomPadding)

	// Add logo to bottom right (ensure we have enough width)
	logoWidth := d.width - 15
	if logoWidth < 0 {
		logoWidth = 0
	}
	logoLine := strings.Repeat(" ", logoWidth) + logo

	return paddedContent + "\n" + logoLine
}

// renderBranchAscii creates a simple bordered box with the branch name
func (d *DashboardView) renderBranchAscii() string {
	branchStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("cyan"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("cyan")).
		Padding(0, 2).
		MarginTop(1).
		MarginBottom(1).
		MarginLeft(5)

	// Add branch icon
	branchText := fmt.Sprintf("🌿 %s", d.branch)
	return boxStyle.Render(branchStyle.Render(branchText))
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
