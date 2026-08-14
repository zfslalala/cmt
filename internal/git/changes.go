package git

import (
	"os/exec"
	"strings"
)

type FileChange struct {
	Status  string
	OldPath string
	NewPath string
}

type StagedChanges struct {
	Files       []FileChange
	DiffContent string
}

// GetStagedChanges 获取暂存区变更
func GetStagedChanges() (*StagedChanges, error) {
	files, err := getStagedFiles()
	if err != nil {
		return nil, err
	}

	diff, err := getStagedDiff()
	if err != nil {
		return nil, err
	}

	return &StagedChanges{Files: files, DiffContent: diff}, nil
}

func getStagedFiles() ([]FileChange, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-status", "--diff-filter=ACDMRT")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []FileChange
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		change := FileChange{Status: status}
		if status == "R" || status == "C" {
			if len(parts) >= 3 {
				change.OldPath = parts[1]
				change.NewPath = parts[2]
			}
		} else {
			change.NewPath = parts[1]
		}

		files = append(files, change)
	}

	return files, nil
}

func getStagedDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 500 {
		lines = append(lines[:500], "... (diff 内容过长，已截断)")
	}

	return strings.Join(lines, "\n"), nil
}

// GetStatusSymbol 获取状态符号
func GetStatusSymbol(status string) string {
	symbols := map[string]string{
		"A": "新增",
		"M": "修改",
		"D": "删除",
		"R": "重命名",
		"C": "复制",
		"T": "类型变更",
	}
	if symbol, ok := symbols[status]; ok {
		return symbol
	}
	return status
}
