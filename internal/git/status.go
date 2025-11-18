package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

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

// GetLastCommitTime returns the timestamp of the last commit
func GetLastCommitTime() (time.Time, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%ct")
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last commit time: %w", err)
	}

	timestampStr := strings.TrimSpace(string(output))
	if timestampStr == "" {
		return time.Time{}, fmt.Errorf("no commits found")
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return time.Unix(timestamp, 0), nil
}

// GetLineStats returns per-file line statistics for uncommitted changes
// Returns a map of filename -> [added, deleted]
func GetLineStats() (map[string][2]int, error) {
	cmd := exec.Command("git", "diff", "--numstat")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get line stats: %w", err)
	}

	fileStats := make(map[string][2]int)
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		addStr := fields[0]
		delStr := fields[1]
		filename := fields[2]

		added := 0
		deleted := 0

		// Handle binary files (marked with -)
		if addStr != "-" {
			if count, err := strconv.Atoi(addStr); err == nil {
				added = count
			}
		}

		if delStr != "-" {
			if count, err := strconv.Atoi(delStr); err == nil {
				deleted = count
			}
		}

		fileStats[filename] = [2]int{added, deleted}
	}

	return fileStats, nil
}
