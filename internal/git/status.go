package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/johannesberggren/gitgoblin/internal/models"
)

// GetWorkingTreeStatus returns all file changes in the working tree
func GetWorkingTreeStatus() ([]models.FileChange, error) {
	cmd := exec.Command("git", "status", "--porcelain=v1")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	return parseStatus(output)
}

// parseStatus parses git status --porcelain output
// Format: XY PATH
// X = staged status, Y = working tree status
func parseStatus(output []byte) ([]models.FileChange, error) {
	var files []models.FileChange
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 4 {
			continue
		}

		stagedChar := string(line[0])
		workingChar := string(line[1])
		path := strings.TrimSpace(line[3:])

		// Handle renames (format: "R  old -> new")
		if stagedChar == "R" {
			parts := strings.Split(path, " -> ")
			if len(parts) == 2 {
				path = parts[1] // Use new name
			}
		}

		file := models.FileChange{
			Path: path,
		}

		// Parse staged status
		switch stagedChar {
		case "M":
			file.StagedStatus = models.StatusModified
			file.IsStaged = true
		case "A":
			file.StagedStatus = models.StatusAdded
			file.IsStaged = true
		case "D":
			file.StagedStatus = models.StatusDeleted
			file.IsStaged = true
		case "R":
			file.StagedStatus = models.StatusRenamed
			file.IsStaged = true
		case "C":
			file.StagedStatus = models.StatusCopied
			file.IsStaged = true
		}

		// Parse working tree status
		switch workingChar {
		case "M":
			file.Status = models.StatusModified
		case "D":
			file.Status = models.StatusDeleted
		case "?":
			file.Status = models.StatusUntracked
			file.IsUntracked = true
		}

		files = append(files, file)
	}

	return files, nil
}

// StageFile stages a specific file
func StageFile(path string) error {
	cmd := exec.Command("git", "add", path)
	return cmd.Run()
}

// UnstageFile unstages a specific file
func UnstageFile(path string) error {
	cmd := exec.Command("git", "restore", "--staged", path)
	return cmd.Run()
}

// StageAll stages all changes
func StageAll() error {
	cmd := exec.Command("git", "add", "-A")
	return cmd.Run()
}

// GetDiff returns the diff for a file
func GetDiff(path string, staged bool) (string, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--staged")
	}
	args = append(args, "--", path)

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	return string(output), nil
}

// Commit creates a commit with the given message
func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("commit failed: %s", string(output))
	}
	return nil
}

// HasUncommittedChanges checks if there are any uncommitted changes
func HasUncommittedChanges() (bool, error) {
	files, err := GetWorkingTreeStatus()
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}
