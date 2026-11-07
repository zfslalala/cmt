package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CreateWorktree 创建新分支，并在当前执行目录的兄弟目录创建对应工作树。
// 目录名取分支名（/ 与空格转成 -）。返回工作树绝对路径。
func CreateWorktree(branch string) (string, error) {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "", fmt.Errorf("分支名不能为空")
	}

	// 分支已存在时拒绝，避免误操作已有分支
	if _, err := runGit("rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		return "", fmt.Errorf("分支 %s 已存在，如需为已有分支创建工作树请手动执行: git worktree add <路径> %s", branch, branch)
	}

	// 分支已在其他 worktree 中检出
	if path, ok, err := WorktreePathForBranch(branch); err != nil {
		return "", err
	} else if ok {
		return "", fmt.Errorf("分支 %s 已在工作树中检出: %s", branch, path)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("获取当前目录失败: %w", err)
	}

	dirName := SanitizeBranchName(branch)
	if dirName == "" {
		return "", fmt.Errorf("分支名 %s 无法转换为目录名", branch)
	}
	target := filepath.Join(filepath.Dir(cwd), dirName)

	// 目标目录不允许是当前仓库根目录（例如当前目录叫 test、分支也叫 test 的情况）
	if root, err := RepoRoot(); err == nil && filepath.Clean(root) == filepath.Clean(target) {
		return "", fmt.Errorf("目标目录 %s 与当前仓库根目录相同，请更换分支名", target)
	}
	if _, err := os.Stat(target); err == nil {
		return "", fmt.Errorf("目标目录已存在: %s", target)
	}

	if _, err := runGit("worktree", "add", "-b", branch, target); err != nil {
		return "", err
	}
	return target, nil
}

// SanitizeBranchName 将分支名转换为安全的目录名。
// 除了 / \ 空格，还替换 Windows 文件名的非法字符 : < > " | ? *，
// 保证同一个分支在 macOS 与 Windows 上生成相同且合法的目录名。
func SanitizeBranchName(branch string) string {
	replacer := strings.NewReplacer(
		"/", "-", "\\", "-", " ", "-",
		":", "-", "<", "-", ">", "-",
		"\"", "-", "|", "-", "?", "-", "*", "-",
	)
	return replacer.Replace(branch)
}
