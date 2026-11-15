package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RepoRootIn 获取指定目录所在工作树的根目录。
func RepoRootIn(dir string) (string, error) {
	output, err := runGitInDir(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// CurrentBranchIn 获取指定目录的当前分支名。
func CurrentBranchIn(dir string) (string, error) {
	output, err := runGitInDir(dir, "branch", "--show-current")
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(output)
	if branch == "" {
		return "", fmt.Errorf("当前处于 detached HEAD 状态，无法确定当前分支")
	}

	return branch, nil
}

// WorktreePaths 返回仓库全部工作树的绝对路径，主工作树位于首位。
// git worktree list 本身以主工作树开头，这里保持该顺序。
func WorktreePaths(dir string) ([]string, error) {
	output, err := runGitInDir(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	paths := []string{}
	for _, line := range strings.Split(output, "\n") {
		path, ok := strings.CutPrefix(line, "worktree ")
		if !ok {
			continue
		}
		if path = strings.TrimSpace(path); path != "" {
			paths = append(paths, path)
		}
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("未找到任何工作树")
	}

	return paths, nil
}

// CommonDir 返回仓库共享的 git 目录，所有工作树共用同一个。
// 用于读写 info/exclude、info/kb-project 这类不随工作树变化的本地文件。
// --git-common-dir 在主工作树下返回相对路径，这里统一转成绝对路径。
func CommonDir(dir string) (string, error) {
	output, err := runGitInDir(dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}

	common := strings.TrimSpace(output)
	if common == "" {
		return "", fmt.Errorf("无法确定 git 共享目录")
	}

	if !filepath.IsAbs(common) {
		base := dir
		if base == "" {
			if base, err = os.Getwd(); err != nil {
				return "", fmt.Errorf("获取当前目录失败: %w", err)
			}
		}
		common = filepath.Join(base, common)
	}

	return filepath.Clean(common), nil
}
