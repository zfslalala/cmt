package cmt

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/build"
	"github.com/zfslalala/cmt/internal/git"
	"github.com/zfslalala/cmt/internal/idea"
	"github.com/zfslalala/cmt/internal/workbuddy"
)

var (
	wtOpenIdea      bool
	wtOpenWorkBuddy bool
	wtFrom          string
)

// runWt 创建新分支与对应工作树，可用 -i 在 IntelliJ IDEA 中打开
func runWt(cmd *cobra.Command, args []string) error {
	branch := args[0]

	if !git.IsGitRepo() {
		return fmt.Errorf("当前目录不是 Git 仓库")
	}

	// 起点：--from 优先，其次 qg.hjson 的 wt.base，都为空则基于当前 HEAD
	start := wtFrom
	if start == "" {
		start = build.WorktreeBase()
	}

	path, resolvedStart, err := git.CreateWorktree(branch, start)
	if err != nil {
		return err
	}

	fmt.Printf("工作树已创建: %s\n", path)
	fmt.Printf("分支: %s\n", branch)
	if resolvedStart != "" {
		fmt.Printf("基于: %s\n", resolvedStart)
	}

	if wtOpenIdea {
		if err := idea.Open(path); err != nil {
			if errors.Is(err, idea.ErrNotFound) {
				fmt.Fprintln(os.Stderr, "未检测到 IntelliJ IDEA，已跳过打开")
			} else {
				return err
			}
		} else {
			fmt.Println("已使用 IntelliJ IDEA 打开工作树")
		}
	}

	if wtOpenWorkBuddy {
		if err := workbuddy.OpenTask(path, ""); err != nil {
			// 工作树已创建成功，打开失败只警告不报错
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Println("已用 WorkBuddy 打开工作树（新任务草稿）")
		}
	}
	return nil
}
