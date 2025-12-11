package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Johannes-Berggren/GitGoblin/internal/models"
)

// GetCommits retrieves the commit history with graph information
func GetCommits(limit int) ([]models.Commit, []string, error) {
	// Format: hash|short|author|email|date|refs|parents|message
	format := "%H|%h|%an|%ae|%at|%D|%P|%s"

	args := []string{
		"log",
		fmt.Sprintf("--pretty=format:%s", format),
		"--all",
		"--date-order",
	}

	if limit > 0 {
		args = append(args, fmt.Sprintf("-%d", limit))
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to run git log: %w", err)
	}

	// Get graph lines separately
	graphArgs := []string{
		"log",
		"--graph",
		"--oneline",
		"--all",
		"--date-order",
	}

	if limit > 0 {
		graphArgs = append(graphArgs, fmt.Sprintf("-%d", limit))
	}

	graphCmd := exec.Command("git", graphArgs...)
	graphOutput, err := graphCmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to run git log --graph: %w", err)
	}

	commits := parseCommits(output)
	graphLines := parseGraphLines(graphOutput)

	return commits, graphLines, nil
}

func parseCommits(output []byte) []models.Commit {
	var commits []models.Commit
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "|")
		if len(parts) < 8 {
			continue
		}

		unixTime, _ := strconv.ParseInt(parts[4], 10, 64)
		timestamp := time.Unix(unixTime, 0)

		refs := []string{}
		if parts[5] != "" {
			refParts := strings.Split(parts[5], ", ")
			for _, ref := range refParts {
				ref = strings.TrimSpace(ref)
				if ref != "" {
					refs = append(refs, ref)
				}
			}
		}

		parents := []string{}
		if parts[6] != "" {
			parents = strings.Fields(parts[6])
		}

		commit := models.Commit{
			Hash:      parts[0],
			ShortHash: parts[1],
			Author:    parts[2],
			Email:     parts[3],
			Date:      timestamp,
			Refs:      refs,
			Parents:   parents,
			Message:   parts[7],
		}

		commits = append(commits, commit)
	}

	return commits
}

func parseGraphLines(output []byte) []string {
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

// GetCurrentBranch returns the name of the current branch
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetStatus returns a simple status of the repo
func GetStatus() (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return "clean", nil
	}

	return fmt.Sprintf("%d changes", len(lines)), nil
}
