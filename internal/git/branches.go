package git

import (
	"fmt"
	"strings"
)

const defaultRemote = "origin"

type Worktree struct {
	Path   string
	Branch string
}

// MergeConflictError 表示合并产生冲突，失败的自动合并已被回滚。
type MergeConflictError struct {
	SourceBranch string
	TargetBranch string
	WorktreePath string
	Cause        error
}

func (e *MergeConflictError) Error() string {
	if e.WorktreePath != "" {
		return fmt.Sprintf("合并到 %s 时发生冲突，已回滚自动合并，请在 worktree %s 手动合并", e.TargetBranch, e.WorktreePath)
	}
	return fmt.Sprintf("合并到 %s 时发生冲突，已回滚自动合并，请在目标分支手动合并", e.TargetBranch)
}

func (e *MergeConflictError) Unwrap() error {
	return e.Cause
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

// SyncBranchFromCurrent 将当前分支合并到目标分支，推送后回到当前分支。
// 合并冲突时自动回滚失败的 merge，并提示用户手动合并。
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

	if err := requireCleanWorktree(""); err != nil {
		return err
	}

	targetPath, targetIsWorktree, err := WorktreePathForBranch(targetBranch)
	if err != nil {
		return err
	}
	if targetIsWorktree {
		return syncInWorktree(targetPath, currentBranch, targetBranch)
	}

	if err := checkoutBranch(targetBranch); err != nil {
		return err
	}

	restoreSource := true
	defer func() {
		if !restoreSource {
			return
		}
		if checkoutErr := checkoutBranch(currentBranch); checkoutErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; 另外切回 %s 分支失败: %v", err, currentBranch, checkoutErr)
				return
			}
			err = fmt.Errorf("同步已执行，但切回 %s 分支失败: %w", currentBranch, checkoutErr)
		}
	}()

	if err := mergeBranch(currentBranch); err != nil {
		if hasMergeConflicts("") {
			_ = abortMerge()
			restoreSource = false
			return &MergeConflictError{SourceBranch: currentBranch, TargetBranch: targetBranch, Cause: err}
		}
		return fmt.Errorf("合并 %s 到 %s 失败: %w", currentBranch, targetBranch, err)
	}

	return pushBranch(defaultRemote, targetBranch)
}

func syncInWorktree(worktreePath, sourceBranch, targetBranch string) error {
	if err := requireCleanWorktree(worktreePath); err != nil {
		return fmt.Errorf("目标分支 %s 所在 worktree 不干净: %w，路径: %s", targetBranch, err, worktreePath)
	}

	if err := mergeBranchInDir(worktreePath, sourceBranch); err != nil {
		if hasMergeConflicts(worktreePath) {
			_ = abortMergeInDir(worktreePath)
			return &MergeConflictError{
				SourceBranch: sourceBranch,
				TargetBranch: targetBranch,
				WorktreePath: worktreePath,
				Cause:        err,
			}
		}
		return fmt.Errorf("在 worktree %s 合并 %s 到 %s 失败: %w", worktreePath, sourceBranch, targetBranch, err)
	}

	return pushBranchInDir(worktreePath, defaultRemote, targetBranch)
}

func requireCleanWorktree(dir string) error {
	hasChanges, err := hasUncommittedChangesInDir(dir)
	if err != nil {
		return err
	}
	if hasChanges {
		if dir == "" {
			return fmt.Errorf("当前工作区有未提交变更，请先提交或暂存后再执行")
		}
		return fmt.Errorf("工作区有未提交变更")
	}
	return nil
}

func hasMergeConflicts(dir string) bool {
	output, err := runGitInDir(dir, "diff", "--name-only", "--diff-filter=U")
	return err == nil && strings.TrimSpace(output) != ""
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

func mergeBranchInDir(dir, branch string) error {
	_, err := runGitInDir(dir, "merge", branch)
	return err
}

func abortMergeInDir(dir string) error {
	_, err := runGitInDir(dir, "merge", "--abort")
	return err
}

func pushBranch(remote, branch string) error {
	return pushBranchInDir("", remote, branch)
}

func pushBranchInDir(dir, remote, branch string) error {
	_, err := runGitInDir(dir, "push", remote, branch)
	return err
}
