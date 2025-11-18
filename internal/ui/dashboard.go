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
	// Branch as large ASCII art (top)
	branchAscii := d.renderBranchAscii()

	// Remote status (only if diverged, near branch)
	var remoteStatus string
	if d.aheadCount > 0 || d.behindCount > 0 {
		remoteStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("yellow")).
			Bold(true).
			MarginBottom(1).
			MarginLeft(5)

		parts := []string{}
		if d.aheadCount > 0 {
			parts = append(parts, fmt.Sprintf("↑%d ahead", d.aheadCount))
		}
		if d.behindCount > 0 {
			parts = append(parts, fmt.Sprintf("↓%d behind", d.behindCount))
		}
		remoteStatus = remoteStyle.Render("⚠ origin: " + strings.Join(parts, ", "))
	}

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
			Foreground(lipgloss.Color("yellow")).
			Bold(true).
			MarginTop(2).
			MarginBottom(1)

		fileStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("white"))

		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("yellow")).
			Bold(true)

		var fileList strings.Builder
		fileList.WriteString(titleStyle.Render(fmt.Sprintf("%d Uncommitted File(s):", len(d.files))) + "\n\n")

		// Show ALL files (no limit)
		for _, file := range d.files {
			status := statusStyle.Render(file.DisplayStatus())
			path := fileStyle.Render(file.Path)
			fileList.WriteString(fmt.Sprintf(" %s  %s\n", status, path))
		}

		content = fileList.String()
	}

	// Build top section (branch + remote)
	var topSection string
	if remoteStatus != "" {
		topSection = lipgloss.JoinVertical(lipgloss.Left, branchAscii, "", remoteStatus, "")
	} else {
		topSection = lipgloss.JoinVertical(lipgloss.Left, branchAscii, "")
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

	// Add logo to bottom right
	logoLine := strings.Repeat(" ", d.width-15) + logo

	return paddedContent + "\n" + logoLine
}

// renderBranchAscii creates large ASCII art representation of branch name
func (d *DashboardView) renderBranchAscii() string {
	// Convert branch name to uppercase for cleaner ASCII art
	branch := strings.ToUpper(d.branch)

	// Build ASCII art - using block style
	var lines [6]strings.Builder

	for _, char := range branch {
		ascii := getAsciiChar(char)
		for i, line := range ascii {
			lines[i].WriteString(line)
			lines[i].WriteString(" ") // Space between letters
		}
	}

	// Join all lines and style
	var result strings.Builder
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("cyan")).
		Bold(true).
		MarginTop(1).
		MarginBottom(1).
		MarginLeft(5)

	for _, line := range lines {
		result.WriteString(style.Render(line.String()) + "\n")
	}

	return result.String()
}

// getAsciiChar returns ASCII art for a single character
func getAsciiChar(c rune) [6]string {
	switch c {
	case 'A':
		return [6]string{
			" █████╗ ",
			"██╔══██╗",
			"███████║",
			"██╔══██║",
			"██║  ██║",
			"╚═╝  ╚═╝",
		}
	case 'B':
		return [6]string{
			"██████╗ ",
			"██╔══██╗",
			"██████╔╝",
			"██╔══██╗",
			"██████╔╝",
			"╚═════╝ ",
		}
	case 'C':
		return [6]string{
			" ██████╗",
			"██╔════╝",
			"██║     ",
			"██║     ",
			"╚██████╗",
			" ╚═════╝",
		}
	case 'D':
		return [6]string{
			"██████╗ ",
			"██╔══██╗",
			"██║  ██║",
			"██║  ██║",
			"██████╔╝",
			"╚═════╝ ",
		}
	case 'E':
		return [6]string{
			"███████╗",
			"██╔════╝",
			"█████╗  ",
			"██╔══╝  ",
			"███████╗",
			"╚══════╝",
		}
	case 'F':
		return [6]string{
			"███████╗",
			"██╔════╝",
			"█████╗  ",
			"██╔══╝  ",
			"██║     ",
			"╚═╝     ",
		}
	case 'G':
		return [6]string{
			" ██████╗ ",
			"██╔════╝ ",
			"██║  ███╗",
			"██║   ██║",
			"╚██████╔╝",
			" ╚═════╝ ",
		}
	case 'H':
		return [6]string{
			"██╗  ██╗",
			"██║  ██║",
			"███████║",
			"██╔══██║",
			"██║  ██║",
			"╚═╝  ╚═╝",
		}
	case 'I':
		return [6]string{
			"██╗",
			"██║",
			"██║",
			"██║",
			"██║",
			"╚═╝",
		}
	case 'J':
		return [6]string{
			"     ██╗",
			"     ██║",
			"     ██║",
			"██   ██║",
			"╚█████╔╝",
			" ╚════╝ ",
		}
	case 'K':
		return [6]string{
			"██╗  ██╗",
			"██║ ██╔╝",
			"█████╔╝ ",
			"██╔═██╗ ",
			"██║  ██╗",
			"╚═╝  ╚═╝",
		}
	case 'L':
		return [6]string{
			"██╗     ",
			"██║     ",
			"██║     ",
			"██║     ",
			"███████╗",
			"╚══════╝",
		}
	case 'M':
		return [6]string{
			"███╗   ███╗",
			"████╗ ████║",
			"██╔████╔██║",
			"██║╚██╔╝██║",
			"██║ ╚═╝ ██║",
			"╚═╝     ╚═╝",
		}
	case 'N':
		return [6]string{
			"███╗   ██╗",
			"████╗  ██║",
			"██╔██╗ ██║",
			"██║╚██╗██║",
			"██║ ╚████║",
			"╚═╝  ╚═══╝",
		}
	case 'O':
		return [6]string{
			" ██████╗ ",
			"██╔═══██╗",
			"██║   ██║",
			"██║   ██║",
			"╚██████╔╝",
			" ╚═════╝ ",
		}
	case 'P':
		return [6]string{
			"██████╗ ",
			"██╔══██╗",
			"██████╔╝",
			"██╔═══╝ ",
			"██║     ",
			"╚═╝     ",
		}
	case 'Q':
		return [6]string{
			" ██████╗ ",
			"██╔═══██╗",
			"██║   ██║",
			"██║▄▄ ██║",
			"╚██████╔╝",
			" ╚══▀▀═╝ ",
		}
	case 'R':
		return [6]string{
			"██████╗ ",
			"██╔══██╗",
			"██████╔╝",
			"██╔══██╗",
			"██║  ██║",
			"╚═╝  ╚═╝",
		}
	case 'S':
		return [6]string{
			"███████╗",
			"██╔════╝",
			"███████╗",
			"╚════██║",
			"███████║",
			"╚══════╝",
		}
	case 'T':
		return [6]string{
			"████████╗",
			"╚══██╔══╝",
			"   ██║   ",
			"   ██║   ",
			"   ██║   ",
			"   ╚═╝   ",
		}
	case 'U':
		return [6]string{
			"██╗   ██╗",
			"██║   ██║",
			"██║   ██║",
			"██║   ██║",
			"╚██████╔╝",
			" ╚═════╝ ",
		}
	case 'V':
		return [6]string{
			"██╗   ██╗",
			"██║   ██║",
			"██║   ██║",
			"╚██╗ ██╔╝",
			" ╚████╔╝ ",
			"  ╚═══╝  ",
		}
	case 'W':
		return [6]string{
			"██╗    ██╗",
			"██║    ██║",
			"██║ █╗ ██║",
			"██║███╗██║",
			"╚███╔███╔╝",
			" ╚══╝╚══╝ ",
		}
	case 'X':
		return [6]string{
			"██╗  ██╗",
			"╚██╗██╔╝",
			" ╚███╔╝ ",
			" ██╔██╗ ",
			"██╔╝ ██╗",
			"╚═╝  ╚═╝",
		}
	case 'Y':
		return [6]string{
			"██╗   ██╗",
			"╚██╗ ██╔╝",
			" ╚████╔╝ ",
			"  ╚██╔╝  ",
			"   ██║   ",
			"   ╚═╝   ",
		}
	case 'Z':
		return [6]string{
			"███████╗",
			"╚══███╔╝",
			"  ███╔╝ ",
			" ███╔╝  ",
			"███████╗",
			"╚══════╝",
		}
	case '0':
		return [6]string{
			" ██████╗ ",
			"██╔═████╗",
			"██║██╔██║",
			"████╔╝██║",
			"╚██████╔╝",
			" ╚═════╝ ",
		}
	case '1':
		return [6]string{
			" ██╗",
			"███║",
			"╚██║",
			" ██║",
			" ██║",
			" ╚═╝",
		}
	case '2':
		return [6]string{
			"██████╗ ",
			"╚════██╗",
			" █████╔╝",
			"██╔═══╝ ",
			"███████╗",
			"╚══════╝",
		}
	case '3':
		return [6]string{
			"██████╗ ",
			"╚════██╗",
			" █████╔╝",
			" ╚═══██╗",
			"██████╔╝",
			"╚═════╝ ",
		}
	case '4':
		return [6]string{
			"██╗  ██╗",
			"██║  ██║",
			"███████║",
			"╚════██║",
			"     ██║",
			"     ╚═╝",
		}
	case '5':
		return [6]string{
			"███████╗",
			"██╔════╝",
			"███████╗",
			"╚════██║",
			"███████║",
			"╚══════╝",
		}
	case '6':
		return [6]string{
			" ██████╗ ",
			"██╔════╝ ",
			"███████╗ ",
			"██╔═══██╗",
			"╚██████╔╝",
			" ╚═════╝ ",
		}
	case '7':
		return [6]string{
			"███████╗",
			"╚════██║",
			"    ██╔╝",
			"   ██╔╝ ",
			"   ██║  ",
			"   ╚═╝  ",
		}
	case '8':
		return [6]string{
			" █████╗ ",
			"██╔══██╗",
			"╚█████╔╝",
			"██╔══██╗",
			"╚█████╔╝",
			" ╚════╝ ",
		}
	case '9':
		return [6]string{
			" █████╗ ",
			"██╔══██╗",
			"╚██████║",
			" ╚═══██║",
			" █████╔╝",
			" ╚════╝ ",
		}
	case '-':
		return [6]string{
			"       ",
			"       ",
			"█████╗ ",
			"╚════╝ ",
			"       ",
			"       ",
		}
	case '_':
		return [6]string{
			"        ",
			"        ",
			"        ",
			"        ",
			"███████╗",
			"╚══════╝",
		}
	case '/':
		return [6]string{
			"    ██╗",
			"   ██╔╝",
			"  ██╔╝ ",
			" ██╔╝  ",
			"██╔╝   ",
			"╚═╝    ",
		}
	default:
		// For unsupported characters, return a space
		return [6]string{
			"   ",
			"   ",
			"   ",
			"   ",
			"   ",
			"   ",
		}
	}
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
