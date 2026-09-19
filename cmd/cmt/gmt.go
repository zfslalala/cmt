package cmt

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/build"
	"github.com/zfslalala/cmt/internal/git"
)

// buildFlag 由 -b flag 控制,推送成功后是否触发构建
var buildFlag bool

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

	if buildFlag {
		triggerBuilds(targetBranch)
	}
	return nil
}

// triggerBuilds 依次执行 qg.hjson 中目标分支的构建命令。
// 单条失败仅警告并继续执行后续命令,不影响已完成的合并推送结果。
func triggerBuilds(branch string) {
	cfg, err := build.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ 读取 qg.hjson 失败: %v\n", err)
		return
	}

	entry, ok := cfg.EntryFor(branch)
	if !ok {
		fmt.Printf("qg.hjson 中未配置 %s 的构建命令，跳过触发\n", branch)
		return
	}

	total := len(entry.Curl)
	fmt.Printf("正在触发 %s 构建（%d 个命令）...\n", branch, total)

	success := 0
	for i, rawCmd := range entry.Curl {
		cmdStr := build.EnsureFailFlag(rawCmd)
		if err := build.RunCommand(cmdStr); err != nil {
			fmt.Fprintf(os.Stderr, "✗ [%d/%d] 执行失败: %v\n    命令: %s\n", i+1, total, err, cmdStr)
			continue
		}
		success++
		fmt.Printf("✓ [%d/%d] 完成\n", i+1, total)
	}

	if success < total {
		fmt.Fprintf(os.Stderr, "⚠ 构建触发完成：%d/%d 成功（合并推送已成功，不受影响）\n", success, total)
	} else {
		fmt.Printf("构建触发完成：%d/%d 成功\n", success, total)
	}
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
