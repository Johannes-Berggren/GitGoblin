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

type GraphView struct {
	commits    []models.Commit
	graphLines []string
	cursor     int
	offset     int
	height     int
	width      int
}

func NewGraphView() *GraphView {
	return &GraphView{
		cursor: 0,
		offset: 0,
	}
}

type commitsLoadedMsg struct {
	commits    []models.Commit
	graphLines []string
}

func (g *GraphView) Init() tea.Cmd {
	return g.loadCommits()
}

func (g *GraphView) loadCommits() tea.Cmd {
	return func() tea.Msg {
		commits, graphLines, err := git.GetCommits(100)
		if err != nil {
			return errMsg{err}
		}
		return commitsLoadedMsg{commits, graphLines}
	}
}

func (g *GraphView) Update(msg tea.Msg) (*GraphView, tea.Cmd) {
	switch msg := msg.(type) {
	case commitsLoadedMsg:
		g.commits = msg.commits
		g.graphLines = msg.graphLines

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if g.cursor < len(g.commits)-1 {
				g.cursor++
				// Auto-scroll down
				if g.cursor >= g.offset+g.height-5 {
					g.offset++
				}
			}

		case "k", "up":
			if g.cursor > 0 {
				g.cursor--
				// Auto-scroll up
				if g.cursor < g.offset {
					g.offset--
				}
			}

		case "g":
			// Go to top
			g.cursor = 0
			g.offset = 0

		case "G":
			// Go to bottom
			g.cursor = len(g.commits) - 1
			if g.cursor > g.height-5 {
				g.offset = g.cursor - g.height + 5
			}
		}

	case tea.WindowSizeMsg:
		g.width = msg.Width
		g.height = msg.Height
	}

	return g, nil
}

func (g *GraphView) View() string {
	if len(g.commits) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("Loading commits...")
	}

	var b strings.Builder

	// Determine how many commits we can show
	visibleCount := g.height - 4 // Leave room for header and footer
	if visibleCount < 1 {
		visibleCount = 1
	}

	start := g.offset
	end := start + visibleCount
	if end > len(g.commits) {
		end = len(g.commits)
	}

	for i := start; i < end; i++ {
		commit := g.commits[i]
		graph := ""
		if i < len(g.graphLines) {
			// Extract just the graph part from the line
			graphLine := g.graphLines[i]
			// Find where the hash starts (after the graph)
			hashIdx := strings.Index(graphLine, commit.ShortHash)
			if hashIdx > 0 {
				graph = graphLine[:hashIdx]
			} else {
				graph = "  "
			}
		}

		line := g.formatCommitLine(commit, graph, i == g.cursor)
		b.WriteString(line + "\n")
	}

	return b.String()
}

func (g *GraphView) formatCommitLine(commit models.Commit, graph string, selected bool) string {
	// Styles
	hashStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("yellow"))
	authorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("cyan"))
	dateStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("white"))
	refStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("green")).
		Bold(true)
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("238")).
		Foreground(lipgloss.Color("white"))

	// Format relative time
	relTime := formatRelativeTime(commit.Date)

	// Build the line
	parts := []string{
		graph,
		hashStyle.Render(commit.ShortHash),
	}

	// Add refs if any
	if len(commit.Refs) > 0 {
		refs := "(" + strings.Join(commit.Refs, ", ") + ")"
		parts = append(parts, refStyle.Render(refs))
	}

	parts = append(parts,
		messageStyle.Render(commit.Message),
		dateStyle.Render(fmt.Sprintf("- %s", relTime)),
		authorStyle.Render(fmt.Sprintf("<%s>", commit.Author)),
	)

	line := strings.Join(parts, " ")

	// Truncate if too long
	maxWidth := g.width - 2
	if maxWidth > 0 && len(line) > maxWidth {
		// This is approximate since we're not counting ANSI codes properly
		line = line[:maxWidth-3] + "..."
	}

	if selected {
		line = selectedStyle.Render("▸ " + line)
	} else {
		line = "  " + line
	}

	return line
}

func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return fmt.Sprintf("%d min ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		return fmt.Sprintf("%d weeks ago", weeks)
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(diff.Hours() / 24 / 365)
		return fmt.Sprintf("%d years ago", years)
	}
}

func (g *GraphView) SelectedCommit() *models.Commit {
	if g.cursor >= 0 && g.cursor < len(g.commits) {
		return &g.commits[g.cursor]
	}
	return nil
}
