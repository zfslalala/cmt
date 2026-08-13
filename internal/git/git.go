package git

import (
	"fmt"
	"os/exec"
	"strings"
)

const defaultRemote = "origin"

type FileChange struct {
	Status  string
	OldPath string
	NewPath string
}

type StagedChanges struct {
	Files       []FileChange
	DiffContent string
}

type Worktree struct {
	Path   string
	Branch string
}

// IsGitRepo 检查当前是否在 Git 仓库中
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
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

// Worktrees 获取当前仓库的 worktree 列表
func Worktrees() ([]Worktree, error) {
	output, err := runGit("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var worktrees []Worktree
	var current Worktree
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = Worktree{}
			}
		case strings.HasPrefix(line, "worktree "):
			current.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch refs/heads/"):
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		}
	}

	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil

}

// WorktreePathForBranch 获取指定分支所在的 worktree 路径
func WorktreePathForBranch(branch string) (string, bool, error) {
	worktrees, err := Worktrees()
	if err != nil {
		return "", false, err
	}

	for _, worktree := range worktrees {
		if worktree.Branch == branch {
			return worktree.Path, true, nil
		}
	}

	return "", false, nil
}

// SyncBranchFromCurrent 将当前分支合并到目标分支，推送后切回当前分支
func SyncBranchFromCurrent(targetBranch string) (err error) {
	targetBranch = strings.TrimSpace(targetBranch)
	if targetBranch == "" {
		return fmt.Errorf("目标分支不能为空")
	}

	currentBranch, err := CurrentBranch()
	if err != nil {
		return err
	}

	if currentBranch == targetBranch {
		return fmt.Errorf("当前已经在 %s 分支，无需同步", targetBranch)
	}

	hasChanges, err := HasUncommittedChanges()
	if err != nil {
		return err
	}
	if hasChanges {
		return fmt.Errorf("当前工作区有未提交变更，请先提交或暂存后再执行")
	}

	if targetWorktreePath, ok, err := WorktreePathForBranch(targetBranch); err != nil {
		return err
	} else if ok {
		return syncBranchInWorktree(targetWorktreePath, currentBranch, targetBranch)
	}

	if err := checkoutBranch(targetBranch); err != nil {
		return err
	}

	defer func() {
		if checkoutErr := checkoutBranch(currentBranch); checkoutErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; 另外切回 %s 分支失败: %v", err, currentBranch, checkoutErr)
				return
			}
			err = fmt.Errorf("同步已执行，但切回 %s 分支失败: %w", currentBranch, checkoutErr)
		}
	}()

	if err := mergeBranch(currentBranch); err != nil {
		_ = abortMerge()
		return fmt.Errorf("合并 %s 到 %s 失败: %w", currentBranch, targetBranch, err)
	}

	if err := pushBranch(defaultRemote, targetBranch); err != nil {
		return err
	}

	return nil
}

func syncBranchInWorktree(worktreePath, sourceBranch, targetBranch string) error {
	hasChanges, err := hasUncommittedChangesInDir(worktreePath)
	if err != nil {
		return err
	}
	if hasChanges {
		return fmt.Errorf("目标分支 %s 所在 worktree 有未提交变更: %s", targetBranch, worktreePath)
	}

	if err := mergeBranchInDir(worktreePath, sourceBranch); err != nil {
		_ = abortMergeInDir(worktreePath)
		return fmt.Errorf("在 worktree %s 合并 %s 到 %s 失败: %w", worktreePath, sourceBranch, targetBranch, err)
	}

	if err := pushBranchInDir(worktreePath, defaultRemote, targetBranch); err != nil {
		return err
	}

	return nil
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

	return &StagedChanges{
		Files:       files,
		DiffContent: diff,
	}, nil
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

		// 处理重命名 R old_path new_path
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

	// 限制 diff 行数，防止过长
	lines := strings.Split(string(output), "\n")
	maxLines := 500
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, "... (diff 内容过长，已截断)")
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

// AddAll 执行 git add . 暂存所有变更
func AddAll() error {
	cmd := exec.Command("git", "add", ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add 失败: %v, output: %s", err, string(output))
	}
	return nil
}

// Commit 执行 git commit
func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit 失败: %v, output: %s", err, string(output))
	}
	return nil
}

// Push 执行 git push
func Push() error {
	cmd := exec.Command("git", "push")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push 失败: %v, output: %s", err, string(output))
	}
	return nil
}

func checkoutBranch(branch string) error {
	_, err := runGit("checkout", branch)
	return err
}

func mergeBranch(branch string) error {
	return mergeBranchInDir("", branch)
}

func abortMerge() error {
	return abortMergeInDir("")
}

func pushBranch(remote, branch string) error {
	return pushBranchInDir("", remote, branch)
}

func hasUncommittedChangesInDir(dir string) (bool, error) {
	output, err := runGitInDir(dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(output) != "", nil
}

func mergeBranchInDir(dir, branch string) error {
	_, err := runGitInDir(dir, "merge", branch)
	return err
}

func abortMergeInDir(dir string) error {
	_, err := runGitInDir(dir, "merge", "--abort")
	return err
}

func pushBranchInDir(dir, remote, branch string) error {
	_, err := runGitInDir(dir, "push", remote, branch)
	return err
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
