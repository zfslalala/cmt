package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// IsGitRepo 检查当前是否在 Git 仓库中
func IsGitRepo() bool {
	output, err := runGit("rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(output) == "true"
}

// CurrentBranch 获取当前分支名
func CurrentBranch() (string, error) {
	output, err := runGit("branch", "--show-current")
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(output)
	if branch == "" {
		return "", fmt.Errorf("当前处于 detached HEAD 状态，无法确定当前分支")
	}

	return branch, nil
}

// HasUncommittedChanges 检查工作区或暂存区是否有未提交变更
func HasUncommittedChanges() (bool, error) {
	return hasUncommittedChangesInDir("")
}

// AddAll 执行 git add . 暂存所有变更
func AddAll() error {
	_, err := runGit("add", ".")
	return err
}

// Commit 执行 git commit
func Commit(message string) error {
	_, err := runGit("commit", "-m", message)
	return err
}

// Push 执行 git push
func Push() error {
	_, err := runGit("push")
	return err
}

func hasUncommittedChangesInDir(dir string) (bool, error) {
	output, err := runGitInDir(dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(output) != "", nil
}

func runGit(args ...string) (string, error) {
	return runGitInDir("", args...)
}

func runGitInDir(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s 失败: %v, output: %s", strings.Join(args, " "), err, string(output))
	}

	return string(output), nil
}
