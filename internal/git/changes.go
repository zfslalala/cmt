package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

const (
	maxStagedDiffBytes = 64 << 10
	maxStagedDiffLines = 500
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
		return "", fmt.Errorf("git %s 失败: %w, stderr: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "git", "diff", "--cached")
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("读取暂存区 diff 失败: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("启动 git diff 失败: %w", err)
	}

	data, readErr := io.ReadAll(io.LimitReader(stdout, maxStagedDiffBytes+1))
	byteTruncated := len(data) > maxStagedDiffBytes
	if byteTruncated {
		data = data[:maxStagedDiffBytes]
		cancel()
	}

	waitErr := cmd.Wait()
	if readErr != nil {
		return "", fmt.Errorf("读取 git diff 失败: %w", readErr)
	}
	if waitErr != nil && !byteTruncated {
		return "", fmt.Errorf("git diff --cached 失败: %v, stderr: %s", waitErr, strings.TrimSpace(stderr.String()))
	}

	text := strings.TrimRight(string(data), "\n")
	lines := strings.Split(text, "\n")
	lineTruncated := len(lines) > maxStagedDiffLines
	if lineTruncated {
		lines = lines[:maxStagedDiffLines]
		text = strings.Join(lines, "\n")
	}

	switch {
	case byteTruncated:
		text += "\n... (diff 超过 64 KiB，已截断；可用 git diff --cached 查看完整内容)"
	case lineTruncated:
		text += "\n... (diff 超过 500 行，已截断；可用 git diff --cached 查看完整内容)"
	}

	return text, nil
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
