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

// GetDefaultBranch detects the repository's default branch
func GetDefaultBranch() (string, error) {
	// Method 1: Try symbolic-ref (fastest, most reliable if set)
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD", "--short")
	output, err := cmd.Output()
	if err == nil {
		branchName := strings.TrimSpace(string(output))
		// Output is like "origin/main", strip "origin/" prefix
		if strings.HasPrefix(branchName, "origin/") {
			return strings.TrimPrefix(branchName, "origin/"), nil
		}
		return branchName, nil
	}

	// Method 2: Try git remote show origin
	cmd = exec.Command("git", "remote", "show", "origin")
	output, err = cmd.Output()
	if err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(output))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.Contains(line, "HEAD branch:") {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1]), nil
				}
			}
		}
	}

	// Method 3: Fallback to common default branch names
	commonDefaults := []string{"main", "master", "dev", "develop"}
	for _, branchName := range commonDefaults {
		cmd = exec.Command("git", "rev-parse", "--verify", "origin/"+branchName)
		if err := cmd.Run(); err == nil {
			return branchName, nil
		}
	}

	return "", fmt.Errorf("could not detect default branch")
}

// CreateBranchFromDefault creates a new branch from the latest default branch
func CreateBranchFromDefault(branchName string) error {
	// 1. Get default branch name
	defaultBranch, err := GetDefaultBranch()
	if err != nil {
		return fmt.Errorf("failed to detect default branch: %w", err)
	}

	// 2. Fetch latest from origin
	cmd := exec.Command("git", "fetch", "origin", defaultBranch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fetch: %s", string(output))
	}

	// 3. Create and checkout new branch from origin/<default>
	cmd = exec.Command("git", "checkout", "-b", branchName, "origin/"+defaultBranch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create branch: %s", string(output))
	}

	return nil
}

// GetBranchComparison returns ahead/behind counts compared to the default branch
func GetBranchComparison(currentBranch, defaultBranch string) (ahead, behind int, err error) {
	// Use git rev-list --left-right --count to get both values efficiently
	// Format: origin/<default>...HEAD
	target := fmt.Sprintf("origin/%s...HEAD", defaultBranch)
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", target)
	output, cmdErr := cmd.Output()
	if cmdErr != nil {
		return 0, 0, fmt.Errorf("failed to compare branches: %w", cmdErr)
	}

	// Output format: "5\t3" (5 behind, 3 ahead)
	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected git rev-list output format")
	}

	// First number is commits in default branch not in current (behind)
	// Second number is commits in current branch not in default (ahead)
	fmt.Sscanf(parts[0], "%d", &behind)
	fmt.Sscanf(parts[1], "%d", &ahead)

	return ahead, behind, nil
}
