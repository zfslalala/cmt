package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CreateWorktree 创建新分支，并在当前执行目录的兄弟目录创建对应工作树。
// 目录名取分支名（/ 与空格转成 -）。返回工作树绝对路径与实际使用的起点引用。
// startPoint 为空时基于当前 HEAD；否则优先线上最新（fetch 后取 origin/<name>），
// 远端不存在同名分支时回退本地分支。
func CreateWorktree(branch, startPoint string) (string, string, error) {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "", "", fmt.Errorf("分支名不能为空")
	}

	// 分支已存在时拒绝，避免误操作已有分支
	if _, err := runGit("rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		return "", "", fmt.Errorf("分支 %s 已存在，如需为已有分支创建工作树请手动执行: git worktree add <路径> %s", branch, branch)
	}

	// 分支已在其他 worktree 中检出
	if path, ok, err := WorktreePathForBranch(branch); err != nil {
		return "", "", err
	} else if ok {
		return "", "", fmt.Errorf("分支 %s 已在工作树中检出: %s", branch, path)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("获取当前目录失败: %w", err)
	}

	dirName := SanitizeBranchName(branch)
	if dirName == "" {
		return "", "", fmt.Errorf("分支名 %s 无法转换为目录名", branch)
	}
	target := filepath.Join(filepath.Dir(cwd), dirName)

	// 目标目录不允许是当前仓库根目录（例如当前目录叫 test、分支也叫 test 的情况）
	if root, err := RepoRoot(); err == nil && filepath.Clean(root) == filepath.Clean(target) {
		return "", "", fmt.Errorf("目标目录 %s 与当前仓库根目录相同，请更换分支名", target)
	}
	if _, err := os.Stat(target); err == nil {
		return "", "", fmt.Errorf("目标目录已存在: %s", target)
	}

	args := []string{"worktree", "add", "-b", branch, target}
	resolvedStart := ""
	if startPoint != "" {
		ref, err := resolveStartPoint(startPoint)
		if err != nil {
			return "", "", err
		}
		resolvedStart = ref
		args = append(args, ref)
	}

	if _, err := runGit(args...); err != nil {
		return "", "", err
	}
	return target, resolvedStart, nil
}

// resolveStartPoint 解析新分支的起点：
//  1. fetch 远端最新，用 origin/<name>（保证基于线上代码而非可能落后的本地分支）
//  2. 远端不存在同名分支时，回退本地分支
func resolveStartPoint(name string) (string, error) {
	// 允许直接写 origin/release 的形式
	ref := strings.TrimPrefix(name, "origin/")

	if _, err := runGit("fetch", "origin", ref); err == nil {
		return "origin/" + ref, nil
	}
	if branchExists(ref) {
		return ref, nil
	}
	return "", fmt.Errorf("起点 %s 在远程和本地都不存在", name)
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
