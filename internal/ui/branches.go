package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Johannes-Berggren/GitGoblin/internal/git"
	"github.com/Johannes-Berggren/GitGoblin/internal/models"
)

const (
	sortDefault = iota
	sortAZ
	sortZA
)

type BranchView struct {
	branches    []models.Branch
	localOnly   []models.Branch
	filtered    []models.Branch
	cursor      int
	width       int
	height      int
	searchInput textinput.Model
	searchQuery string
	sortMode    int
}

func NewBranchView() *BranchView {
	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40

	return &BranchView{
		cursor:      0,
		searchInput: ti,
	}
}

type branchesLoadedMsg struct {
	branches []models.Branch
}

type branchSwitchDoneMsg struct {
	name string
}

type branchSwitchCancelMsg struct{}

func (b *BranchView) Init() tea.Cmd {
	return tea.Batch(b.loadBranches(), textinput.Blink)
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

func (b *BranchView) applyFilterAndSort() {
	b.filtered = nil
	query := strings.ToLower(b.searchQuery)
	for _, branch := range b.localOnly {
		if query == "" || strings.Contains(strings.ToLower(branch.Name), query) {
			b.filtered = append(b.filtered, branch)
		}
	}

	switch b.sortMode {
	case sortAZ:
		sort.Slice(b.filtered, func(i, j int) bool {
			return strings.ToLower(b.filtered[i].Name) < strings.ToLower(b.filtered[j].Name)
		})
	case sortZA:
		sort.Slice(b.filtered, func(i, j int) bool {
			return strings.ToLower(b.filtered[i].Name) > strings.ToLower(b.filtered[j].Name)
		})
	}

	if b.cursor >= len(b.filtered) {
		b.cursor = len(b.filtered) - 1
	}
	if b.cursor < 0 {
		b.cursor = 0
	}
}

func (b *BranchView) Update(msg tea.Msg) (*BranchView, tea.Cmd) {
	switch msg := msg.(type) {
	case branchesLoadedMsg:
		b.branches = msg.branches
		b.localOnly = nil
		for _, branch := range b.branches {
			if !branch.IsRemote {
				b.localOnly = append(b.localOnly, branch)
			}
		}
		b.applyFilterAndSort()

	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if b.cursor > 0 {
				b.cursor--
			}
			return b, nil

		case "down":
			if b.cursor < len(b.filtered)-1 {
				b.cursor++
			}
			return b, nil

		case "enter":
			selected := b.SelectedBranch()
			if selected != nil && !selected.IsCurrent {
				return b, func() tea.Msg {
					return branchSwitchDoneMsg{name: selected.Name}
				}
			}
			return b, nil

		case "esc":
			if b.searchQuery != "" {
				b.searchInput.SetValue("")
				b.searchQuery = ""
				b.applyFilterAndSort()
				return b, nil
			}
			return b, func() tea.Msg { return branchSwitchCancelMsg{} }

		case "tab":
			b.sortMode = (b.sortMode + 1) % 3
			b.applyFilterAndSort()
			return b, nil
		}

		// Forward all other keys to the text input for filtering
		var cmd tea.Cmd
		b.searchInput, cmd = b.searchInput.Update(msg)

		newQuery := b.searchInput.Value()
		if newQuery != b.searchQuery {
			b.searchQuery = newQuery
			b.applyFilterAndSort()
		}

		return b, cmd

	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
	}

	return b, nil
}

func (b *BranchView) View() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("cyan")).
		Bold(true)

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

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	searchLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("cyan")).
		Bold(true)

	var out strings.Builder

	// Search bar (always active)
	out.WriteString("  " + searchLabelStyle.Render("🔍 ") + b.searchInput.View() + "\n\n")

	// Header
	var header string
	if b.searchQuery != "" {
		header = fmt.Sprintf("Branches (%d of %d local)", len(b.filtered), len(b.localOnly))
	} else {
		header = fmt.Sprintf("Branches (%d local)", len(b.localOnly))
	}

	switch b.sortMode {
	case sortAZ:
		header += " • sorted: a-z"
	case sortZA:
		header += " • sorted: z-a"
	}

	out.WriteString("  " + headerStyle.Render(header) + "\n\n")

	// Calculate visible rows for branch list
	// Reserve: search(1) + blank(1) + header(1) + blank(1) + blank(1) + footer(1) = 6 fixed rows
	height := b.height
	if height == 0 {
		height = 24 // sensible default before WindowSizeMsg arrives
	}
	visibleRows := height - 6
	if visibleRows < 1 {
		visibleRows = 1
	}

	// Branch list with scrolling window
	if len(b.filtered) == 0 {
		if b.searchQuery != "" {
			out.WriteString("  " + helpStyle.Render("No matching branches") + "\n")
		} else if len(b.localOnly) == 0 {
			out.WriteString("  " + helpStyle.Render("Loading branches...") + "\n")
		}
	} else {
		// Calculate scroll window to keep cursor visible
		start := 0
		end := len(b.filtered)

		if len(b.filtered) > visibleRows {
			// Keep cursor centered when possible
			half := visibleRows / 2
			start = b.cursor - half
			if start < 0 {
				start = 0
			}
			end = start + visibleRows
			if end > len(b.filtered) {
				end = len(b.filtered)
				start = end - visibleRows
			}
		}

		scrollInfo := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

		if start > 0 {
			out.WriteString("  " + scrollInfo.Render(fmt.Sprintf("↑ %d more", start)) + "\n")
		}

		for i := start; i < end; i++ {
			branch := b.filtered[i]
			var line string

			if branch.IsCurrent {
				line = "* " + currentStyle.Render(branch.Name)
			} else {
				line = "  " + branchStyle.Render(branch.Name)
			}

			hash := branch.Hash
			if len(hash) > 7 {
				hash = hash[:7]
			}
			line += " " + hashStyle.Render(hash)

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

		if end < len(b.filtered) {
			out.WriteString("  " + scrollInfo.Render(fmt.Sprintf("↓ %d more", len(b.filtered)-end)) + "\n")
		}
	}

	// Footer
	out.WriteString("\n" + helpStyle.Render("  ↑↓: navigate • tab: sort • enter: checkout • esc: back"))

	return out.String()
}

func (b *BranchView) SelectedBranch() *models.Branch {
	if b.cursor >= 0 && b.cursor < len(b.filtered) {
		return &b.filtered[b.cursor]
	}
	return nil
}
