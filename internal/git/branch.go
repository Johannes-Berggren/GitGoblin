package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/johannesberggren/gitgoblin/internal/models"
)

// GetBranches returns all branches with their info
func GetBranches() ([]models.Branch, error) {
	// Get branches with their last commit
	cmd := exec.Command("git", "branch", "-vv", "--all")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}

	return parseBranches(output)
}

func parseBranches(output []byte) ([]models.Branch, error) {
	var branches []models.Branch
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		branch := models.Branch{}

		// Check if current branch (marked with *)
		if strings.HasPrefix(line, "*") {
			branch.IsCurrent = true
			line = strings.TrimPrefix(line, "*")
		} else {
			line = strings.TrimPrefix(line, " ")
		}

		line = strings.TrimSpace(line)

		// Parse: name hash [upstream] message
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		branch.Name = parts[0]
		branch.Hash = parts[1]

		// Check if it's a remote branch
		if strings.HasPrefix(branch.Name, "remotes/") {
			branch.IsRemote = true
			branch.Name = strings.TrimPrefix(branch.Name, "remotes/")
		}

		// Parse upstream info [origin/main: ahead 2, behind 1]
		if len(parts) > 2 && strings.HasPrefix(parts[2], "[") {
			upstreamInfo := extractUpstreamInfo(line)
			if upstreamInfo != "" {
				branch.Upstream = upstreamInfo
			}
		}

		// Get last commit message (everything after hash and upstream)
		messageStart := strings.Index(line, branch.Hash) + len(branch.Hash)
		if messageStart < len(line) {
			message := line[messageStart:]
			// Skip upstream info if present
			if idx := strings.Index(message, "]"); idx > 0 && strings.Contains(message[:idx], "[") {
				message = message[idx+1:]
			}
			branch.LastCommit = strings.TrimSpace(message)
		}

		branches = append(branches, branch)
	}

	return branches, nil
}

func extractUpstreamInfo(line string) string {
	start := strings.Index(line, "[")
	end := strings.Index(line, "]")
	if start >= 0 && end > start {
		return line[start+1 : end]
	}
	return ""
}

// SwitchBranch checks out a different branch
func SwitchBranch(name string) error {
	cmd := exec.Command("git", "checkout", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to switch branch: %s", string(output))
	}
	return nil
}

// CreateBranch creates a new branch
func CreateBranch(name string) error {
	cmd := exec.Command("git", "branch", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create branch: %s", string(output))
	}
	return nil
}

// DeleteBranch deletes a branch
func DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	cmd := exec.Command("git", "branch", flag, name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete branch: %s", string(output))
	}
	return nil
}
