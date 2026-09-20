package cmt

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/git"
)

// runP 暂存所有本地修改并提交，然后推送到远程
func runP(cmd *cobra.Command, args []string) error {
	message := strings.Join(args, " ")

	if !git.IsGitRepo() {
		return fmt.Errorf("当前目录不是 Git 仓库")
	}

	hasChanges, err := git.HasUncommittedChanges()
	if err != nil {
		return fmt.Errorf("检查变更失败: %w", err)
	}
	if !hasChanges {
		return fmt.Errorf("没有可提交的变更")
	}

	fmt.Println("正在暂存所有修改...")
	if err := git.AddAll(); err != nil {
		return fmt.Errorf("暂存失败: %w", err)
	}

	fmt.Printf("正在提交: %s\n", message)
	if err := git.Commit(message); err != nil {
		return fmt.Errorf("提交失败: %w", err)
	}
	fmt.Println("提交成功!")

	fmt.Println("正在推送到远程...")
	if err := git.Push(); err != nil {
		return fmt.Errorf("推送失败: %w", err)
	}
	fmt.Println("推送成功!")

	return nil
}
