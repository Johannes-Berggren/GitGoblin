package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johannesberggren/gitgoblin/internal/git"
)

type CommitView struct {
	textarea     textarea.Model
	width        int
	height       int
	stagedCount  int
	err          error
}

func NewCommitView(stagedCount int) *CommitView {
	ta := textarea.New()
	ta.Placeholder = "Commit message..."
	ta.Focus()
	ta.CharLimit = 0
	ta.SetWidth(80)
	ta.SetHeight(5)

	return &CommitView{
		textarea:    ta,
		stagedCount: stagedCount,
	}
}

type commitSuccessMsg struct{}

func (c *CommitView) Init() tea.Cmd {
	return textarea.Blink
}

func (c *CommitView) Update(msg tea.Msg) (*CommitView, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			// Cancel commit
			return c, nil

		case "ctrl+d":
			// Submit commit
			message := strings.TrimSpace(c.textarea.Value())
			if message == "" {
				c.err = fmt.Errorf("commit message cannot be empty")
				return c, nil
			}

			return c, c.performCommit(message)
		}

	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height

	case errMsg:
		c.err = msg.err
		return c, nil

	case commitSuccessMsg:
		// Signal that commit was successful
		return c, nil
	}

	// Update textarea
	c.textarea, cmd = c.textarea.Update(msg)
	return c, cmd
}

func (c *CommitView) performCommit(message string) tea.Cmd {
	return func() tea.Msg {
		err := git.Commit(message)
		if err != nil {
			return errMsg{err}
		}
		return commitSuccessMsg{}
	}
}

func (c *CommitView) View() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("cyan")).
		Bold(true).
		MarginBottom(1)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("red"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	var b strings.Builder

	title := fmt.Sprintf("Commit %d staged file(s)", c.stagedCount)
	b.WriteString(titleStyle.Render(title) + "\n")

	if c.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", c.err)) + "\n\n")
	}

	b.WriteString(c.textarea.View() + "\n")

	help := "ctrl+d: commit • esc: cancel"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}
