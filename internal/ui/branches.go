package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Johannes-Berggren/GitGoblin/internal/git"
	"github.com/Johannes-Berggren/GitGoblin/internal/models"
)

type BranchView struct {
	branches     []models.Branch
	localOnly    []models.Branch
	cursor       int
	width        int
	height       int
}

func NewBranchView() *BranchView {
	return &BranchView{
		cursor: 0,
	}
}

type branchesLoadedMsg struct {
	branches []models.Branch
}

func (b *BranchView) Init() tea.Cmd {
	return b.loadBranches()
}

func (b *BranchView) loadBranches() tea.Cmd {
	return func() tea.Msg {
		branches, err := git.GetBranches()
		if err != nil {
			return errMsg{err}
		}
		return branchesLoadedMsg{branches}
	}
}

func (b *BranchView) Update(msg tea.Msg) (*BranchView, tea.Cmd) {
	switch msg := msg.(type) {
	case branchesLoadedMsg:
		b.branches = msg.branches
		// Filter to local branches only for display
		b.localOnly = []models.Branch{}
		for _, branch := range b.branches {
			if !branch.IsRemote {
				b.localOnly = append(b.localOnly, branch)
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if b.cursor < len(b.localOnly)-1 {
				b.cursor++
			}

		case "k", "up":
			if b.cursor > 0 {
				b.cursor--
			}

		case "g":
			b.cursor = 0

		case "G":
			b.cursor = len(b.localOnly) - 1

		case "r":
			return b, b.loadBranches()
		}

	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
	}

	return b, nil
}

func (b *BranchView) View() string {
	if len(b.localOnly) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("Loading branches...")
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("cyan")).
		Bold(true).
		MarginBottom(1)

	branchStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("white"))

	currentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("green")).
		Bold(true)

	hashStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("yellow"))

	upstreamStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("238"))

	var out strings.Builder

	header := fmt.Sprintf("Branches (%d local)", len(b.localOnly))
	out.WriteString(headerStyle.Render(header) + "\n")

	for i, branch := range b.localOnly {
		var line string

		if branch.IsCurrent {
			line = "* " + currentStyle.Render(branch.Name)
		} else {
			line = "  " + branchStyle.Render(branch.Name)
		}

		line += " " + hashStyle.Render(branch.Hash[:7])

		if branch.Upstream != "" {
			line += " " + upstreamStyle.Render(fmt.Sprintf("[%s]", branch.Upstream))
		}

		if branch.LastCommit != "" {
			line += " " + upstreamStyle.Render(branch.LastCommit)
		}

		if i == b.cursor {
			line = selectedStyle.Render("▸ " + line)
		} else {
			line = "  " + line
		}

		out.WriteString(line + "\n")
	}

	return out.String()
}

func (b *BranchView) SelectedBranch() *models.Branch {
	if b.cursor >= 0 && b.cursor < len(b.localOnly) {
		return &b.localOnly[b.cursor]
	}
	return nil
}
