package git

import (
	"bytes"
	"fmt"
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

// runGitOutput 执行 git 命令并返回 stdout，stderr 单独捕获用于错误信息
func runGitOutput(args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s 失败: %v, stderr: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// getStagedFiles 获取暂存区文件列表。
// 用 -z NUL 分隔格式解析，兼容路径含空格的情况；
// 重命名/复制的状态带相似度分数（如 R100），归一为单个字母。
func getStagedFiles() ([]FileChange, error) {
	output, err := runGitOutput("diff", "--cached", "--name-status", "-z", "--diff-filter=ACDMRT")
	if err != nil {
		return nil, err
	}

	tokens := strings.Split(output, "\x00")
	var files []FileChange
	for i := 0; i < len(tokens); {
		tok := tokens[i]
		if tok == "" {
			i++
			continue
		}
		status := tok
		if len(status) > 1 && status[1] >= '0' && status[1] <= '9' {
			status = status[:1]
		}
		i++

		switch status {
		case "R", "C":
			if i+1 < len(tokens) {
				files = append(files, FileChange{Status: status, OldPath: tokens[i], NewPath: tokens[i+1]})
				i += 2
			} else {
				i = len(tokens)
			}
		default:
			if i < len(tokens) {
				files = append(files, FileChange{Status: status, NewPath: tokens[i]})
				i++
			}
		}
	}

	return files, nil
}

func getStagedDiff() (string, error) {
	output, err := runGitOutput("diff", "--cached")
	if err != nil {
		return "", err
	}

	lines := strings.Split(output, "\n")
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
