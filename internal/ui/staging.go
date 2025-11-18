package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johannesberggren/gitgoblin/internal/git"
	"github.com/johannesberggren/gitgoblin/internal/models"
)

type StagingView struct {
	files    []models.FileChange
	cursor   int
	width    int
	height   int
	showDiff bool
	diff     string
}

func NewStagingView() *StagingView {
	return &StagingView{
		cursor:   0,
		showDiff: false,
	}
}

type filesLoadedMsg struct {
	files []models.FileChange
}

type diffLoadedMsg struct {
	diff string
}

func (s *StagingView) Init() tea.Cmd {
	return s.loadFiles()
}

func (s *StagingView) loadFiles() tea.Cmd {
	return func() tea.Msg {
		files, err := git.GetWorkingTreeStatus()
		if err != nil {
			return errMsg{err}
		}
		return filesLoadedMsg{files}
	}
}

func (s *StagingView) loadDiff() tea.Cmd {
	if s.cursor < 0 || s.cursor >= len(s.files) {
		return nil
	}

	file := s.files[s.cursor]
	return func() tea.Msg {
		diff, err := git.GetDiff(file.Path, file.IsStaged)
		if err != nil {
			return errMsg{err}
		}
		return diffLoadedMsg{diff}
	}
}

func (s *StagingView) Update(msg tea.Msg) (*StagingView, tea.Cmd) {
	switch msg := msg.(type) {
	case filesLoadedMsg:
		s.files = msg.files
		if len(s.files) > 0 && s.showDiff {
			return s, s.loadDiff()
		}

	case diffLoadedMsg:
		s.diff = msg.diff

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if s.cursor < len(s.files)-1 {
				s.cursor++
				if s.showDiff {
					return s, s.loadDiff()
				}
			}

		case "k", "up":
			if s.cursor > 0 {
				s.cursor--
				if s.showDiff {
					return s, s.loadDiff()
				}
			}

		case "d":
			// Toggle diff preview
			s.showDiff = !s.showDiff
			if s.showDiff {
				return s, s.loadDiff()
			}

		case " ":
			// Stage/unstage file
			return s, s.toggleStage()

		case "a":
			// Stage all
			return s, s.stageAll()

		case "r":
			// Refresh
			return s, s.loadFiles()
		}

	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
	}

	return s, nil
}

func (s *StagingView) toggleStage() tea.Cmd {
	if s.cursor < 0 || s.cursor >= len(s.files) {
		return nil
	}

	file := s.files[s.cursor]

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

		// Reload files after staging
		files, err := git.GetWorkingTreeStatus()
		if err != nil {
			return errMsg{err}
		}
		return filesLoadedMsg{files}
	}
}

func (s *StagingView) stageAll() tea.Cmd {
	return func() tea.Msg {
		err := git.StageAll()
		if err != nil {
			return errMsg{err}
		}

		// Reload files
		files, err := git.GetWorkingTreeStatus()
		if err != nil {
			return errMsg{err}
		}
		return filesLoadedMsg{files}
	}
}

func (s *StagingView) View() string {
	if len(s.files) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("No changes to display\n\nPress 'b' to view branches")
	}

	var b strings.Builder

	// File list
	b.WriteString(s.renderFileList())

	// Diff preview (if enabled)
	if s.showDiff {
		b.WriteString("\n\n")
		b.WriteString(s.renderDiff())
	}

	return b.String()
}

func (s *StagingView) renderFileList() string {
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("yellow")).
		Width(3)

	pathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("white"))

	stagedPathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("green"))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("238"))

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("cyan")).
		Bold(true).
		MarginBottom(1)

	var b strings.Builder

	// Header
	stagedCount := 0
	for _, f := range s.files {
		if f.IsStaged {
			stagedCount++
		}
	}
	header := fmt.Sprintf("Changes (%d total, %d staged)", len(s.files), stagedCount)
	b.WriteString(headerStyle.Render(header) + "\n")

	// Files
	visibleHeight := s.height - 10 // Leave room for header/footer/diff
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	start := s.cursor - visibleHeight/2
	if start < 0 {
		start = 0
	}
	end := start + visibleHeight
	if end > len(s.files) {
		end = len(s.files)
		start = end - visibleHeight
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		file := s.files[i]

		status := statusStyle.Render(file.DisplayStatus())

		var path string
		if file.IsStaged {
			path = stagedPathStyle.Render(file.Path)
		} else {
			path = pathStyle.Render(file.Path)
		}

		line := fmt.Sprintf("%s %s", status, path)

		if i == s.cursor {
			line = selectedStyle.Render("▸ " + line)
		} else {
			line = "  " + line
		}

		b.WriteString(line + "\n")
	}

	return b.String()
}

func (s *StagingView) renderDiff() string {
	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238"))

	diffStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("white")).
		MaxHeight(s.height / 2)

	divider := dividerStyle.Render(strings.Repeat("─", s.width))

	if s.diff == "" {
		return divider + "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("No diff available")
	}

	// Limit diff lines
	lines := strings.Split(s.diff, "\n")
	maxLines := s.height/2 - 3
	if maxLines < 5 {
		maxLines = 5
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, "... (truncated)")
	}

	return divider + "\n" + diffStyle.Render(strings.Join(lines, "\n"))
}

func (s *StagingView) HasStagedFiles() bool {
	for _, f := range s.files {
		if f.IsStaged {
			return true
		}
	}
	return false
}
