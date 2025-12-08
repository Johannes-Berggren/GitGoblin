package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johannesberggren/gitgoblin/internal/git"
	"github.com/johannesberggren/gitgoblin/internal/models"
)

type commitFlowPanel int

const (
	panelStaging commitFlowPanel = iota
	panelCommit
)

type commitFlowDoneMsg struct {
	message string
}

type commitFlowCancelMsg struct{}

type commitFlowFilesMsg struct {
	files []models.FileChange
}

type CommitFlowView struct {
	files      []models.FileChange
	cursor     int
	panel      commitFlowPanel
	textarea   textarea.Model
	width      int
	height     int
	err        error
}

func NewCommitFlowView() *CommitFlowView {
	ta := textarea.New()
	ta.Placeholder = "Commit message..."
	ta.CharLimit = 0
	ta.SetWidth(60)
	ta.SetHeight(3)

	return &CommitFlowView{
		cursor:   0,
		panel:    panelStaging,
		textarea: ta,
	}
}

func (c *CommitFlowView) Init() tea.Cmd {
	return c.loadFiles()
}

func (c *CommitFlowView) loadFiles() tea.Cmd {
	return func() tea.Msg {
		files, err := git.GetWorkingTreeStatus()
		if err != nil {
			return errMsg{err}
		}
		return commitFlowFilesMsg{files}
	}
}

func (c *CommitFlowView) Update(msg tea.Msg) (*CommitFlowView, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case commitFlowFilesMsg:
		c.files = msg.files
		// Keep cursor in bounds
		if c.cursor >= len(c.files) {
			c.cursor = len(c.files) - 1
		}
		if c.cursor < 0 {
			c.cursor = 0
		}
		return c, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return c, func() tea.Msg { return commitFlowCancelMsg{} }

		case "tab":
			// Toggle between panels
			if c.panel == panelStaging {
				c.panel = panelCommit
				c.textarea.Focus()
			} else {
				c.panel = panelStaging
				c.textarea.Blur()
			}
			return c, nil

		case "enter":
			// Only submit from commit panel
			if c.panel == panelCommit {
				message := strings.TrimSpace(c.textarea.Value())
				if message != "" && c.hasStagedFiles() {
					return c, c.performCommit(message)
				}
				if message == "" {
					c.err = fmt.Errorf("commit message cannot be empty")
				} else if !c.hasStagedFiles() {
					c.err = fmt.Errorf("no files staged for commit")
				}
			}
			return c, nil
		}

		// Panel-specific key handling
		if c.panel == panelStaging {
			switch msg.String() {
			case "j", "down":
				if c.cursor < len(c.files)-1 {
					c.cursor++
				}
				return c, nil

			case "k", "up":
				if c.cursor > 0 {
					c.cursor--
				}
				return c, nil

			case " ":
				// Toggle staging
				return c, c.toggleStage()

			case "a":
				// Stage all
				return c, c.stageAll()
			}
		}

	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height
		c.textarea.SetWidth(c.width - 10)

	case errMsg:
		c.err = msg.err
		return c, nil
	}

	// Forward to textarea if in commit panel
	if c.panel == panelCommit {
		c.textarea, cmd = c.textarea.Update(msg)
		return c, cmd
	}

	return c, nil
}

func (c *CommitFlowView) toggleStage() tea.Cmd {
	if c.cursor < 0 || c.cursor >= len(c.files) {
		return nil
	}

	file := c.files[c.cursor]

	return func() tea.Msg {
		var err error
		if file.IsStaged {
			err = git.UnstageFile(file.Path)
		} else {
			err = git.StageFile(file.Path)
		}

		if err != nil {
			return errMsg{err}
		}

		// Reload files
		files, err := git.GetWorkingTreeStatus()
		if err != nil {
			return errMsg{err}
		}
		return commitFlowFilesMsg{files}
	}
}

func (c *CommitFlowView) stageAll() tea.Cmd {
	return func() tea.Msg {
		err := git.StageAll()
		if err != nil {
			return errMsg{err}
		}

		files, err := git.GetWorkingTreeStatus()
		if err != nil {
			return errMsg{err}
		}
		return commitFlowFilesMsg{files}
	}
}

func (c *CommitFlowView) performCommit(message string) tea.Cmd {
	return func() tea.Msg {
		err := git.Commit(message)
		if err != nil {
			return errMsg{err}
		}
		return commitFlowDoneMsg{message: message}
	}
}

func (c *CommitFlowView) hasStagedFiles() bool {
	for _, f := range c.files {
		if f.IsStaged {
			return true
		}
	}
	return false
}

func (c *CommitFlowView) View() string {
	if len(c.files) == 0 {
		grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		return "\n" + grayStyle.Render("  No changes to commit. Press esc to go back.")
	}

	var b strings.Builder

	// Staging panel
	b.WriteString(c.renderStagingPanel())
	b.WriteString("\n\n")

	// Commit panel
	b.WriteString(c.renderCommitPanel())
	b.WriteString("\n\n")

	// Error message
	if c.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", c.err)) + "\n\n")
	}

	// Help text
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	help := "space: toggle • a: stage all • tab: switch • enter: commit • esc: cancel"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (c *CommitFlowView) renderStagingPanel() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("cyan")).
		Bold(true)

	activeTitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("cyan")).
		Bold(true).
		Background(lipgloss.Color("236"))

	// Count staged files
	stagedCount := 0
	for _, f := range c.files {
		if f.IsStaged {
			stagedCount++
		}
	}

	title := fmt.Sprintf(" Stage Files (%d/%d staged) ", stagedCount, len(c.files))
	if c.panel == panelStaging {
		title = activeTitleStyle.Render(title)
	} else {
		title = titleStyle.Render(title)
	}

	var content strings.Builder
	content.WriteString(title + "\n\n")

	// File list with checkboxes
	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("236"))
	stagedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	unstagedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("white"))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("yellow")).Bold(true)

	maxVisible := 8
	start := 0
	if c.cursor >= maxVisible {
		start = c.cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(c.files) {
		end = len(c.files)
	}

	for i := start; i < end; i++ {
		file := c.files[i]

		// Checkbox
		checkbox := "[ ]"
		if file.IsStaged {
			checkbox = "[x]"
		}

		// Status
		status := statusStyle.Render(file.DisplayStatus())

		// Path
		var path string
		if file.IsStaged {
			path = stagedStyle.Render(file.Path)
		} else {
			path = unstagedStyle.Render(file.Path)
		}

		// Cursor indicator
		cursor := "  "
		if i == c.cursor && c.panel == panelStaging {
			cursor = "> "
		}

		line := fmt.Sprintf("%s%s %s %s", cursor, checkbox, status, path)

		if i == c.cursor && c.panel == panelStaging {
			line = selectedStyle.Render(line)
		}

		content.WriteString(line + "\n")
	}

	// Show scroll indicator if needed
	if len(c.files) > maxVisible {
		scrollInfo := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		content.WriteString(scrollInfo.Render(fmt.Sprintf("  ... %d more files", len(c.files)-maxVisible)))
	}

	return content.String()
}

func (c *CommitFlowView) renderCommitPanel() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("cyan")).
		Bold(true)

	activeTitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("cyan")).
		Bold(true).
		Background(lipgloss.Color("236"))

	title := " Commit Message "
	if c.panel == panelCommit {
		title = activeTitleStyle.Render(title)
	} else {
		title = titleStyle.Render(title)
	}

	var content strings.Builder
	content.WriteString(title + "\n\n")
	content.WriteString(c.textarea.View())

	return content.String()
}
