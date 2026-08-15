package cmt

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/git"
)

func runFrom(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("当前目录不是 Git 仓库")
	}

	branch, err := git.CurrentBranch()
	if err != nil {
		return err
	}

	info, err := git.GetBranchInfo(branch)
	if err != nil {
		return fmt.Errorf("获取分支信息失败: %w", err)
	}

	fmt.Printf("分支: %s\n", info.Branch)

	if !info.HasLog {
		fmt.Println("来源: 未知（reflog 无记录）")
		fmt.Println("切出: 未知（reflog 无记录）")
		return nil
	}

	switch {
	case info.Source == "HEAD":
		fmt.Println("来源: 初始分支（HEAD）")
	case info.Source == "":
		fmt.Println("来源: 未知")
	default:
		fmt.Printf("来源: %s\n", info.Source)
	}

	if info.CreatedAt != "" {
		fmt.Printf("切出: %s\n", info.CreatedAt)
	} else {
		fmt.Println("切出: 未知")
	}

	return nil
}
