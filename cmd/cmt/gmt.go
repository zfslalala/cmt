package cmt

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/git"
)

func runGMT(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("当前目录不是 Git 仓库")
	}

	targetBranch := args[0]
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return fmt.Errorf("获取当前分支失败: %w", err)
	}

	fmt.Printf("正在将当前分支 %s 合并到 %s 并推送...\n", currentBranch, targetBranch)
	if err := git.SyncBranchFromCurrent(targetBranch); err != nil {
		return err
	}

	fmt.Printf("同步完成，当前分支仍为 %s\n", currentBranch)
	return nil
}

func printMergeConflictInstructions(conflictErr *git.MergeConflictError) {
	fmt.Fprintf(os.Stderr, "合并冲突：自动合并 %s 到 %s 失败，已回滚合并现场。\n", conflictErr.SourceBranch, conflictErr.TargetBranch)
	if conflictErr.WorktreePath == "" {
		fmt.Fprintln(os.Stderr, "当前已切换到目标分支，请手动执行合并并处理冲突。")
	} else {
		fmt.Fprintf(os.Stderr, "目标分支位于 worktree：%s\n", conflictErr.WorktreePath)
		fmt.Fprintf(os.Stderr, "请进入该目录手动执行合并并处理冲突：cd %s\n", conflictErr.WorktreePath)
	}
	fmt.Fprintln(os.Stderr, "处理步骤：")
	fmt.Fprintln(os.Stderr, "  git status")
	fmt.Fprintf(os.Stderr, "  git merge %s\n", conflictErr.SourceBranch)
	fmt.Fprintln(os.Stderr, "  解决冲突后执行 git add <文件>")
	fmt.Fprintln(os.Stderr, "  git commit")
	fmt.Fprintf(os.Stderr, "  git push origin %s\n", conflictErr.TargetBranch)
}
