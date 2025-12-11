package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Johannes-Berggren/GitGoblin/internal/ui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goblin",
	Short: "A terminal-based Git client",
	Long:  `GitGoblin - A lightweight, terminal-based Git client inspired by GitKraken`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if we're in a git repo
		if !isGitRepo() {
			fmt.Println("Error: Not a git repository")
			os.Exit(1)
		}

		// Initialize and run the TUI
		p := tea.NewProgram(ui.NewModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running app: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func isGitRepo() bool {
	_, err := os.Stat(".git")
	return err == nil
}
