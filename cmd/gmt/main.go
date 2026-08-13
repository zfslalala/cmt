package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/git"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "gmt <target-branch>",
		Short: "将当前分支同步到目标分支",
		Long:  "将当前分支合并到目标分支，推送目标分支后回到当前分支。",
		Args:  cobra.ExactArgs(1),
		Run:   run,
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	if !git.IsGitRepo() {
		fmt.Fprintln(os.Stderr, "错误: 当前目录不是 Git 仓库")
		os.Exit(1)
	}

	targetBranch := args[0]
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取当前分支失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("正在将当前分支 %s 合并到 %s 并推送...\n", currentBranch, targetBranch)
	if err := git.SyncBranchFromCurrent(targetBranch); err != nil {
		fmt.Fprintf(os.Stderr, "分支同步失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("同步完成，当前分支仍为 %s\n", currentBranch)
}
