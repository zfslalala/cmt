package cmt

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/git"
	"github.com/zfslalala/cmt/internal/idea"
)

var wtOpenIdea bool

// runWt 创建新分支与对应工作树，可用 -i 在 IntelliJ IDEA 中打开
func runWt(cmd *cobra.Command, args []string) error {
	branch := args[0]

	if !git.IsGitRepo() {
		return fmt.Errorf("当前目录不是 Git 仓库")
	}

	path, err := git.CreateWorktree(branch)
	if err != nil {
		return err
	}

	fmt.Printf("工作树已创建: %s\n", path)
	fmt.Printf("分支: %s\n", branch)

	if !wtOpenIdea {
		return nil
	}

	if err := idea.Open(path); err != nil {
		if errors.Is(err, idea.ErrNotFound) {
			fmt.Fprintln(os.Stderr, "未检测到 IntelliJ IDEA，已跳过打开")
			return nil
		}
		return err
	}
	fmt.Println("已使用 IntelliJ IDEA 打开工作树")
	return nil
}
