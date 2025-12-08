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
	branch          string
	files           []models.FileChange
	aheadCount      int
	behindCount     int
	lastCommitTime  time.Time
	linesAdded      int
	linesDeleted    int
	fileStats       map[string][2]int
	defaultBranch   string
	aheadOfDefault  int
	behindOfDefault int
	isDefaultBranch bool
	width           int
	height          int
}

func NewDashboardView() *DashboardView {
	return &DashboardView{}
}

type dashboardDataMsg struct {
	branch          string
	files           []models.FileChange
	aheadCount      int
	behindCount     int
	lastCommitTime  time.Time
	linesAdded      int
	linesDeleted    int
	fileStats       map[string][2]int
	defaultBranch   string
	aheadOfDefault  int
	behindOfDefault int
	isDefaultBranch bool
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

		// Get default branch comparison
		defaultBranch, err := git.GetDefaultBranch()
		aheadOfDefault := 0
		behindOfDefault := 0
		isDefaultBranch := false

		if err == nil {
			isDefaultBranch = (branch == defaultBranch)
			if !isDefaultBranch {
				aheadOfDefault, behindOfDefault, _ = git.GetBranchComparison(branch, defaultBranch)
			}
		}

		return dashboardDataMsg{branch, files, ahead, behind, lastCommitTime, linesAdded, linesDeleted, fileStats, defaultBranch, aheadOfDefault, behindOfDefault, isDefaultBranch}
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
		d.defaultBranch = msg.defaultBranch
		d.aheadOfDefault = msg.aheadOfDefault
		d.behindOfDefault = msg.behindOfDefault
		d.isDefaultBranch = msg.isDefaultBranch

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

	greenStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("34")).
		Bold(true)

	redStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	grayStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	// Build line stats with colored numbers (gray for zeros)
	var addedText, deletedText string
	if d.linesAdded > 0 {
		addedText = greenStyle.Render(fmt.Sprintf("+%d", d.linesAdded))
	} else {
		addedText = grayStyle.Render(fmt.Sprintf("+%d", d.linesAdded))
	}

	if d.linesDeleted > 0 {
		deletedText = redStyle.Render(fmt.Sprintf("-%d", d.linesDeleted))
	} else {
		deletedText = grayStyle.Render(fmt.Sprintf("-%d", d.linesDeleted))
	}

	lineStats := fmt.Sprintf("%s/%s", addedText, deletedText)

	metrics := []string{
		fmt.Sprintf("📁 %s %s", labelStyle.Render("Files Changed:"), valueStyle.Render(fmt.Sprintf("%d", len(d.files)))),
		fmt.Sprintf("⏰ %s %s", labelStyle.Render("Last Commit:"), valueStyle.Render(timeSinceCommit)),
		fmt.Sprintf("⬆️  %s %s", labelStyle.Render("Commits Ahead:"), valueStyle.Render(fmt.Sprintf("%d", d.aheadCount))),
		fmt.Sprintf("📊 %s %s", labelStyle.Render("Lines:"), lineStats),
	}

	// Add default branch comparison if not on default branch
	if !d.isDefaultBranch && d.defaultBranch != "" {
		orangeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

		defaultBranchMetric := fmt.Sprintf("🎯 %s %s: %s %s",
			labelStyle.Render("vs"),
			valueStyle.Render(d.defaultBranch),
			greenStyle.Render(fmt.Sprintf("↑%d", d.aheadOfDefault)),
			orangeStyle.Render(fmt.Sprintf("↓%d", d.behindOfDefault)),
		)
		metrics = append(metrics, defaultBranchMetric)
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
		// Clean state - no message needed (metrics show 0 files)
		content = ""
	} else {
		// Dirty state - full width file list, ALL files
		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("cyan")).
			Bold(true)

		// Define status styles with proper colors
		modifiedStatusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("white")).
			Bold(true)

		deletedStatusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

		addedStatusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("34")).
			Bold(true)

		// Styles for line stats
		addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Bold(true)
		deletedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		grayStatsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

		// Build file list content
		var fileList strings.Builder

		// Add title
		title := titleStyle.Render(fmt.Sprintf("📄 %d Uncommitted File(s)", len(d.files)))
		fileList.WriteString(title + "\n\n")

		// Calculate max path width (terminal width - margin - status - spacing - stats)
		// Format: " MM  path (+999/-999)\n"
		// Margin: 5, Status: 4, Spacing: 2, Stats: ~15, Buffer: 5
		maxPathWidth := d.width - 31
		if maxPathWidth < 20 {
			maxPathWidth = 20 // Minimum readable width
		}

		// Show ALL files (no limit)
		for _, file := range d.files {
			// Determine status color based on file state
			var statusStyle lipgloss.Style
			if file.Status == models.StatusDeleted || file.StagedStatus == models.StatusDeleted {
				statusStyle = deletedStatusStyle
			} else if file.IsUntracked || file.Status == models.StatusAdded || file.StagedStatus == models.StatusAdded {
				statusStyle = addedStatusStyle
			} else {
				// Modified, Renamed, Copied, Updated - use white
				statusStyle = modifiedStatusStyle
			}

			status := statusStyle.Render(file.DisplayStatus())

			// Truncate path from left if too long
			displayPath := file.Path
			if len(displayPath) > maxPathWidth {
				// Keep the end of the path (filename is most important)
				displayPath = "..." + displayPath[len(displayPath)-(maxPathWidth-3):]
			}

			// Apply same color to path as status
			var pathStyle lipgloss.Style
			if file.Status == models.StatusDeleted || file.StagedStatus == models.StatusDeleted {
				pathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			} else if file.IsUntracked || file.Status == models.StatusAdded || file.StagedStatus == models.StatusAdded {
				pathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("34"))
			} else {
				pathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("white"))
			}
			path := pathStyle.Render(displayPath)

			// Get line stats for this file
			var statsText string
			if stats, ok := d.fileStats[file.Path]; ok {
				added := stats[0]
				deleted := stats[1]
				if added > 0 || deleted > 0 {
					// Use gray for zero values, green/red for actual changes
					var addText, delText string
					if added > 0 {
						addText = addedStyle.Render(fmt.Sprintf("+%d", added))
					} else {
						addText = grayStatsStyle.Render(fmt.Sprintf("+%d", added))
					}

					if deleted > 0 {
						delText = deletedStyle.Render(fmt.Sprintf("-%d", deleted))
					} else {
						delText = grayStatsStyle.Render(fmt.Sprintf("-%d", deleted))
					}

					statsText = fmt.Sprintf(" (%s/%s)", addText, delText)
				}
			}

			fileList.WriteString(fmt.Sprintf(" %s  %s%s\n", status, path, statsText))
		}

		// Apply left margin to entire file list
		fileListStyle := lipgloss.NewStyle().
			MarginLeft(5).
			MarginTop(1)

		content = fileListStyle.Render(strings.TrimRight(fileList.String(), "\n"))
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

	// Add hint on left, logo on right
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	hint := hintStyle.Render("n new branch")

	// Calculate spacing between hint and logo
	hintLen := 12 // "n new branch"
	logoLen := 12 // "🧙 GitGoblin"
	spacing := d.width - hintLen - logoLen - 5
	if spacing < 1 {
		spacing = 1
	}

	bottomLine := "     " + hint + strings.Repeat(" ", spacing) + logo

	return paddedContent + "\n" + bottomLine
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
