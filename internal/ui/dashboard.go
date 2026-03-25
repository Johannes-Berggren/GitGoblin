package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Johannes-Berggren/GitGoblin/internal/git"
	"github.com/Johannes-Berggren/GitGoblin/internal/models"
)

// Display mode constants based on terminal height
const (
	displayModeNormal      = iota // >= 20 rows
	displayModeCompact            // 12-19 rows
	displayModeUltraCompact       // <= 11 rows
)

type DashboardView struct {
	repoName        string
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
	repoName        string
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
		repoName, err := git.GetRepoName()
		if err != nil {
			repoName = ""
		}

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

		return dashboardDataMsg{repoName, branch, files, ahead, behind, lastCommitTime, linesAdded, linesDeleted, fileStats, defaultBranch, aheadOfDefault, behindOfDefault, isDefaultBranch}
	}
}

func (d *DashboardView) Update(msg tea.Msg) (*DashboardView, tea.Cmd) {
	switch msg := msg.(type) {
	case dashboardDataMsg:
		d.repoName = msg.repoName
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

// getDisplayMode determines which display mode to use based on terminal height
func (d *DashboardView) getDisplayMode() int {
	if d.height >= 25 {
		return displayModeNormal
	} else if d.height >= 12 {
		return displayModeCompact
	}
	return displayModeUltraCompact
}

// formatTimeSinceCommit returns a human-readable time since last commit
func (d *DashboardView) formatTimeSinceCommit() string {
	if d.lastCommitTime.IsZero() {
		return "n/a"
	}
	duration := time.Since(d.lastCommitTime)
	if duration.Hours() < 1 {
		return fmt.Sprintf("%.0fm ago", duration.Minutes())
	} else if duration.Hours() < 24 {
		return fmt.Sprintf("%.1fh ago", duration.Hours())
	}
	days := duration.Hours() / 24
	return fmt.Sprintf("%.1fd ago", days)
}

// renderMetricsPanel creates a bordered box with development metrics (no margins, tight padding)
func (d *DashboardView) renderMetricsPanel() string {
	timeSinceCommit := d.formatTimeSinceCommit()

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

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 2)

	return boxStyle.Render(content)
}

// renderCompactBranchLine renders branch name and warning on a single line
func (d *DashboardView) renderCompactBranchLine() string {
	branchStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("cyan"))

	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("yellow")).
		Bold(true)

	line := fmt.Sprintf("  🌿 %s", branchStyle.Render(d.branch))

	if d.behindCount > 0 {
		// Add spacing and warning
		warning := warningStyle.Render(fmt.Sprintf("⚠ ↓%d behind origin", d.behindCount))
		// Calculate spacing to spread across width
		lineLen := 5 + len(d.branch) // "  🌿 " + branch
		warningLen := 15 + len(fmt.Sprintf("%d", d.behindCount))
		spacing := d.width - lineLen - warningLen - 2
		if spacing < 2 {
			spacing = 2
		}
		line += strings.Repeat(" ", spacing) + warning
	}

	return line
}

// renderCompactMetricsLine renders all metrics horizontally on one line
func (d *DashboardView) renderCompactMetricsLine() string {
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Bold(true)
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Build line stats
	var addedText, deletedText string
	if d.linesAdded > 0 {
		addedText = greenStyle.Render(fmt.Sprintf("+%d", d.linesAdded))
	} else {
		addedText = grayStyle.Render("+0")
	}
	if d.linesDeleted > 0 {
		deletedText = redStyle.Render(fmt.Sprintf("-%d", d.linesDeleted))
	} else {
		deletedText = grayStyle.Render("-0")
	}

	parts := []string{
		fmt.Sprintf("📁 %d files", len(d.files)),
		fmt.Sprintf("📊 %s/%s", addedText, deletedText),
	}

	if d.aheadCount > 0 {
		parts = append(parts, fmt.Sprintf("⬆ %d ahead", d.aheadCount))
	}

	parts = append(parts, fmt.Sprintf("⏰ %s", d.formatTimeSinceCommit()))

	return "  " + strings.Join(parts, "  ")
}

// renderCompactFileList renders a limited number of files for compact mode
func (d *DashboardView) renderCompactFileList(maxFiles int) string {
	if len(d.files) == 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("cyan")).
		Bold(true)

	modifiedStatusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("white")).
		Bold(true)
	deletedStatusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
	addedStatusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("34")).
		Bold(true)

	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Bold(true)
	deletedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	grayStatsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var fileList strings.Builder

	title := titleStyle.Render(fmt.Sprintf("📄 %d Uncommitted File(s)", len(d.files)))
	fileList.WriteString("  " + title + "\n")

	// Calculate max path width
	maxPathWidth := d.width - 31
	if maxPathWidth < 20 {
		maxPathWidth = 20
	}

	// Show limited files
	filesToShow := len(d.files)
	if filesToShow > maxFiles {
		filesToShow = maxFiles
	}

	for i := 0; i < filesToShow; i++ {
		file := d.files[i]

		var statusStyle lipgloss.Style
		if file.Status == models.StatusDeleted || file.StagedStatus == models.StatusDeleted {
			statusStyle = deletedStatusStyle
		} else if file.IsUntracked || file.Status == models.StatusAdded || file.StagedStatus == models.StatusAdded {
			statusStyle = addedStatusStyle
		} else {
			statusStyle = modifiedStatusStyle
		}

		status := statusStyle.Render(file.DisplayStatus())

		displayPath := file.Path
		if len(displayPath) > maxPathWidth {
			displayPath = "..." + displayPath[len(displayPath)-(maxPathWidth-3):]
		}

		var pathStyle lipgloss.Style
		if file.Status == models.StatusDeleted || file.StagedStatus == models.StatusDeleted {
			pathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		} else if file.IsUntracked || file.Status == models.StatusAdded || file.StagedStatus == models.StatusAdded {
			pathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("34"))
		} else {
			pathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("white"))
		}
		path := pathStyle.Render(displayPath)

		var statsText string
		if stats, ok := d.fileStats[file.Path]; ok {
			added := stats[0]
			deleted := stats[1]
			if added > 0 || deleted > 0 {
				var addText, delText string
				if added > 0 {
					addText = addedStyle.Render(fmt.Sprintf("+%d", added))
				} else {
					addText = grayStatsStyle.Render("+0")
				}
				if deleted > 0 {
					delText = deletedStyle.Render(fmt.Sprintf("-%d", deleted))
				} else {
					delText = grayStatsStyle.Render("-0")
				}
				statsText = fmt.Sprintf(" (%s/%s)", addText, delText)
			}
		}

		fileList.WriteString(fmt.Sprintf("   %s  %s%s\n", status, path, statsText))
	}

	if len(d.files) > maxFiles {
		remaining := len(d.files) - maxFiles
		moreStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		fileList.WriteString(moreStyle.Render(fmt.Sprintf("   ... and %d more file(s)\n", remaining)))
	}

	return strings.TrimRight(fileList.String(), "\n")
}

// renderCompactView renders the compact layout for 12-19 row terminals
func (d *DashboardView) renderCompactView() string {
	dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	divider := dividerStyle.Render("  " + strings.Repeat("─", d.width-4))

	// Build main content
	var parts []string

	// Branch line with warning
	parts = append(parts, d.renderCompactBranchLine())
	parts = append(parts, divider)

	// Metrics line
	parts = append(parts, d.renderCompactMetricsLine())
	parts = append(parts, divider)

	// File list (limited based on available height)
	availableRows := d.height - 6 // branch, 2 dividers, metrics, footer
	maxFiles := availableRows - 1 // account for title
	if maxFiles < 1 {
		maxFiles = 1
	}
	if maxFiles > 5 {
		maxFiles = 5
	}

	if len(d.files) > 0 {
		parts = append(parts, d.renderCompactFileList(maxFiles))
	}

	mainContent := strings.Join(parts, "\n")

	// Calculate padding to push footer to bottom
	mainHeight := strings.Count(mainContent, "\n") + 1
	bottomPadding := d.height - mainHeight - 2
	if bottomPadding < 0 {
		bottomPadding = 0
	}

	paddedContent := mainContent + strings.Repeat("\n", bottomPadding)

	// Footer with hints and logo
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	logoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("170"))

	hint := hintStyle.Render("b: branches • n: new branch • r: rename • c: commit")
	logo := logoStyle.Render("🧙 GitGoblin")

	hintLen := 52
	logoLen := 12
	spacing := d.width - hintLen - logoLen - 5
	if spacing < 1 {
		spacing = 1
	}

	bottomLine := "     " + hint + strings.Repeat(" ", spacing) + logo

	return paddedContent + "\n" + bottomLine
}

// renderUltraCompactView renders the densest format for <=11 row terminals
func (d *DashboardView) renderUltraCompactView() string {
	branchStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("cyan"))
	repoStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("white"))
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("yellow")).Bold(true)
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Bold(true)
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	logoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("170"))

	var lines []string

	// Line 1: Repo name + branch + warning
	headerParts := []string{}
	if d.repoName != "" {
		headerParts = append(headerParts, repoStyle.Render(d.repoName))
	}
	headerParts = append(headerParts, fmt.Sprintf("🌿 %s", branchStyle.Render(d.branch)))
	if d.behindCount > 0 {
		headerParts = append(headerParts, warningStyle.Render(fmt.Sprintf("⚠ ↓%d behind", d.behindCount)))
	}
	lines = append(lines, "  "+strings.Join(headerParts, " | "))

	// Line 2: Metrics
	var addedText, deletedText string
	if d.linesAdded > 0 {
		addedText = greenStyle.Render(fmt.Sprintf("+%d", d.linesAdded))
	} else {
		addedText = grayStyle.Render("+0")
	}
	if d.linesDeleted > 0 {
		deletedText = redStyle.Render(fmt.Sprintf("-%d", d.linesDeleted))
	} else {
		deletedText = grayStyle.Render("-0")
	}

	metricsParts := []string{
		fmt.Sprintf("📁 %d files", len(d.files)),
		fmt.Sprintf("📊 %s/%s", addedText, deletedText),
	}
	if d.aheadCount > 0 {
		metricsParts = append(metricsParts, fmt.Sprintf("⬆ %d ahead", d.aheadCount))
	}
	metricsParts = append(metricsParts, fmt.Sprintf("⏰ %s", d.formatTimeSinceCommit()))
	lines = append(lines, "  "+strings.Join(metricsParts, "  "))

	// Lines 3-4: Up to 2 files
	if len(d.files) > 0 {
		maxFiles := 2
		if len(d.files) < maxFiles {
			maxFiles = len(d.files)
		}

		modifiedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("white")).Bold(true)
		deletedStatusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		addedStatusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Bold(true)

		for i := 0; i < maxFiles; i++ {
			file := d.files[i]

			var statusStyle lipgloss.Style
			if file.Status == models.StatusDeleted || file.StagedStatus == models.StatusDeleted {
				statusStyle = deletedStatusStyle
			} else if file.IsUntracked || file.Status == models.StatusAdded || file.StagedStatus == models.StatusAdded {
				statusStyle = addedStatusStyle
			} else {
				statusStyle = modifiedStyle
			}

			status := statusStyle.Render(file.DisplayStatus())

			// Truncate path if needed
			displayPath := file.Path
			maxPathWidth := d.width - 15
			if maxPathWidth < 20 {
				maxPathWidth = 20
			}
			if len(displayPath) > maxPathWidth {
				displayPath = "..." + displayPath[len(displayPath)-(maxPathWidth-3):]
			}

			var pathStyle lipgloss.Style
			if file.Status == models.StatusDeleted || file.StagedStatus == models.StatusDeleted {
				pathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			} else if file.IsUntracked || file.Status == models.StatusAdded || file.StagedStatus == models.StatusAdded {
				pathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("34"))
			} else {
				pathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("white"))
			}

			lines = append(lines, fmt.Sprintf("   %s  %s", status, pathStyle.Render(displayPath)))
		}

		if len(d.files) > 2 {
			remaining := len(d.files) - 2
			lines = append(lines, grayStyle.Render(fmt.Sprintf("   ... +%d more", remaining)))
		}
	}

	mainContent := strings.Join(lines, "\n")

	// Calculate padding to push footer to bottom
	mainHeight := strings.Count(mainContent, "\n") + 1
	bottomPadding := d.height - mainHeight - 2
	if bottomPadding < 0 {
		bottomPadding = 0
	}

	paddedContent := mainContent + strings.Repeat("\n", bottomPadding)

	// Footer with hints and logo
	hint := hintStyle.Render("b: list • n: new • r: ren • c: commit")
	logo := logoStyle.Render("🧙 GitGoblin")

	hintLen := 39
	logoLen := 12
	spacing := d.width - hintLen - logoLen - 5
	if spacing < 1 {
		spacing = 1
	}

	bottomLine := "     " + hint + strings.Repeat(" ", spacing) + logo

	return paddedContent + "\n" + bottomLine
}

// renderNormalView renders the full layout for terminals >= 20 rows
// renderFileList renders the file list with an optional max file limit
func (d *DashboardView) renderFileList(maxFiles int) string {
	if len(d.files) == 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("cyan")).
		Bold(true)

	modifiedStatusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("white")).
		Bold(true)

	deletedStatusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	addedStatusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("34")).
		Bold(true)

	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Bold(true)
	deletedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	grayStatsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var fileList strings.Builder

	title := titleStyle.Render(fmt.Sprintf("📄 %d Uncommitted File(s)", len(d.files)))
	fileList.WriteString(title + "\n\n")

	maxPathWidth := d.width - 31
	if maxPathWidth < 20 {
		maxPathWidth = 20
	}

	filesToShow := len(d.files)
	if maxFiles > 0 && filesToShow > maxFiles {
		filesToShow = maxFiles
	}

	for i := 0; i < filesToShow; i++ {
		file := d.files[i]

		var statusStyle lipgloss.Style
		if file.Status == models.StatusDeleted || file.StagedStatus == models.StatusDeleted {
			statusStyle = deletedStatusStyle
		} else if file.IsUntracked || file.Status == models.StatusAdded || file.StagedStatus == models.StatusAdded {
			statusStyle = addedStatusStyle
		} else {
			statusStyle = modifiedStatusStyle
		}

		status := statusStyle.Render(file.DisplayStatus())

		displayPath := file.Path
		if len(displayPath) > maxPathWidth {
			displayPath = "..." + displayPath[len(displayPath)-(maxPathWidth-3):]
		}

		var pathStyle lipgloss.Style
		if file.Status == models.StatusDeleted || file.StagedStatus == models.StatusDeleted {
			pathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		} else if file.IsUntracked || file.Status == models.StatusAdded || file.StagedStatus == models.StatusAdded {
			pathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("34"))
		} else {
			pathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("white"))
		}
		path := pathStyle.Render(displayPath)

		var statsText string
		if stats, ok := d.fileStats[file.Path]; ok {
			added := stats[0]
			deleted := stats[1]
			if added > 0 || deleted > 0 {
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

	if maxFiles > 0 && len(d.files) > maxFiles {
		remaining := len(d.files) - maxFiles
		moreStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		fileList.WriteString(moreStyle.Render(fmt.Sprintf(" ... and %d more file(s)\n", remaining)))
	}

	fileListStyle := lipgloss.NewStyle().
		MarginLeft(5).
		MarginTop(1)

	return fileListStyle.Render(strings.TrimRight(fileList.String(), "\n"))
}

// renderNormalView renders the full layout for terminals >= 25 rows
func (d *DashboardView) renderNormalView() string {
	branchPanel := d.renderBranchPanel()
	metricsPanel := d.renderMetricsPanel()

	// Side-by-side layout if terminal is wide enough, otherwise vertical
	var headerBlock string
	if d.width >= 60 {
		gap := "  "
		headerBlock = lipgloss.JoinHorizontal(lipgloss.Top, branchPanel, gap, metricsPanel)
	} else {
		headerBlock = lipgloss.JoinVertical(lipgloss.Left, branchPanel, metricsPanel)
	}

	headerStyle := lipgloss.NewStyle().
		MarginLeft(5).
		MarginTop(1)
	headerBlock = headerStyle.Render(headerBlock)

	// Divider
	dividerWidth := d.width - 10
	if dividerWidth < 20 {
		dividerWidth = 20
	}
	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MarginLeft(5)
	divider := dividerStyle.Render(strings.Repeat("─", dividerWidth))

	// Calculate available rows for files
	headerHeight := strings.Count(headerBlock, "\n") + 1
	footerRows := 2  // padding + footer line
	dividerRows := 1
	titleRows := 3   // title + blank line + margin-top
	availableFileRows := d.height - headerHeight - footerRows - dividerRows - titleRows
	if availableFileRows < 1 {
		availableFileRows = 1
	}

	// File list (0 = no limit, positive = max files)
	content := d.renderFileList(availableFileRows)

	// Combine
	mainContent := lipgloss.JoinVertical(lipgloss.Left, headerBlock, divider, content)

	// Logo in bottom right
	logo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Render("🧙 GitGoblin")

	mainHeight := strings.Count(mainContent, "\n") + 1
	bottomPadding := d.height - mainHeight - 2
	if bottomPadding < 0 {
		bottomPadding = 0
	}

	paddedContent := mainContent + strings.Repeat("\n", bottomPadding)

	// Footer
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	hint := hintStyle.Render("b: branches • n: new branch • r: rename • c: commit")

	hintLen := 52
	logoLen := 12
	spacing := d.width - hintLen - logoLen - 5
	if spacing < 1 {
		spacing = 1
	}

	bottomLine := "     " + hint + strings.Repeat(" ", spacing) + logo

	return paddedContent + "\n" + bottomLine
}

func (d *DashboardView) View() string {
	// Handle case where terminal size isn't set yet
	if d.width == 0 || d.height == 0 {
		return "Loading..."
	}

	// Switch based on display mode
	switch d.getDisplayMode() {
	case displayModeUltraCompact:
		return d.renderUltraCompactView()
	case displayModeCompact:
		return d.renderCompactView()
	default:
		return d.renderNormalView()
	}
}

// renderBranchPanel creates a bordered box with branch name and optional warning (no margins)
func (d *DashboardView) renderBranchPanel() string {
	branchStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("cyan"))

	// Truncate branch name if needed to leave room for metrics panel
	displayBranch := d.branch
	maxBranchLen := (d.width - 5 - 2) / 2 // (termWidth - margin - gap) / 2, minus box chrome
	maxBranchLen -= 12                      // padding(4) + border(2) + icon(6)
	if maxBranchLen < 15 {
		maxBranchLen = 15
	}
	if len(displayBranch) > maxBranchLen {
		displayBranch = "..." + displayBranch[len(displayBranch)-(maxBranchLen-3):]
	}

	branchText := fmt.Sprintf("🌿 %s", branchStyle.Render(displayBranch))

	var lines []string
	lines = append(lines, branchText)

	if d.behindCount > 0 {
		warningStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("yellow")).
			Bold(true)
		lines = append(lines, "")
		lines = append(lines, warningStyle.Render(fmt.Sprintf("⚠  Behind origin: ↓%d", d.behindCount)))
	}

	content := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("cyan")).
		Padding(0, 2)

	return boxStyle.Render(content)
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
