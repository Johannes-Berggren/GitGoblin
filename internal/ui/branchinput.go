package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type branchInputDoneMsg struct {
	name string
}

type branchInputCancelMsg struct{}

type BranchInputView struct {
	textInput textinput.Model
	width     int
	height    int
}

func NewBranchInputView() *BranchInputView {
	ti := textinput.New()
	ti.Placeholder = "feature/my-branch"
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40

	return &BranchInputView{
		textInput: ti,
	}
}

func (b *BranchInputView) Init() tea.Cmd {
	return textinput.Blink
}

func (b *BranchInputView) Update(msg tea.Msg) (*BranchInputView, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			name := b.textInput.Value()
			if name != "" {
				return b, func() tea.Msg { return branchInputDoneMsg{name: name} }
			}
			return b, nil
		case "esc":
			return b, func() tea.Msg { return branchInputCancelMsg{} }
		}

	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
	}

	b.textInput, cmd = b.textInput.Update(msg)
	return b, cmd
}

func (b *BranchInputView) View() string {
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	return "\n" +
		promptStyle.Render("New branch name: ") + b.textInput.View() + "\n\n" +
		helpStyle.Render("enter to create • esc to cancel")
}
