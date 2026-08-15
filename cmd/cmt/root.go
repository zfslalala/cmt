package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/git"
)

// executeRoot 构建命令树并执行
func executeRoot() {
	rootCmd := &cobra.Command{
		Use:           "qg",
		Short:         "Git 提交助手:自动生成 commit message 与分支同步",
		Long:          "基于大语言模型的 Git Commit Message 自动生成工具，支持所有 OpenAI 协议兼容的大模型服务。",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmtCmd := &cobra.Command{
		Use:           "cmt",
		Short:         "生成 commit message",
		Long:          "根据暂存区变更自动生成符合 Conventional Commits 规范的 commit message。",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE:          runCMT,
	}
	cmtCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "生成详细的 commit message")
	cmtCmd.Flags().BoolVarP(&push, "push", "p", false, "提交并推送")
	cmtCmd.Flags().BoolVarP(&edit, "edit", "e", false, "编辑后提交")

	gmtCmd := &cobra.Command{
		Use:           "gmt <target-branch>",
		Short:         "将当前分支同步到目标分支",
		Long:          "将当前分支合并到目标分支，推送目标分支后回到当前分支。",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ExactArgs(1),
		RunE:          runGMT,
	}

	rootCmd.AddCommand(cmtCmd, gmtCmd)

	if err := rootCmd.Execute(); err != nil {
		var conflictErr *git.MergeConflictError
		if errors.As(err, &conflictErr) {
			printMergeConflictInstructions(conflictErr)
			os.Exit(1)
		}
		returnCommandError(err)
	}
}

func returnCommandError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
