package cmt

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/git"
)

// Execute 构建命令树并执行
func Execute() {
	rootCmd := newRootCommand()

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		var conflictErr *git.MergeConflictError
		if errors.As(err, &conflictErr) {
			printMergeConflictInstructions(conflictErr)
			os.Exit(1)
		}
		var exitErr *exitCodeError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.code)
		}
		returnCommandError(err)
	}
}

func newRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "qg",
		Short:         "Quick Git:快速 Git 操作工具",
		Long:          "快速 Git 操作工具，支持 AI 自动生成 commit message，兼容所有 OpenAI 协议的大模型服务。",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	fromCmd := &cobra.Command{
		Use:           "from",
		Short:         "查看当前分支来源信息",
		Long:          "显示当前分支的来源分支与切出时间（基于 reflog 推断）。",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE:          runFrom,
	}

	runCmd := &cobra.Command{
		Use:                "run <脚本> [参数...]",
		Short:              "执行脚本并输出到控制台",
		Long:               "编译并执行 Go 脚本，stdout/stderr 实时打印到控制台，退出码透传。脚本可直接给路径，或放 ~/.qg/scripts/（QG_SCRIPTS_DIR 可覆盖）后用名字调用。",
		SilenceErrors:      true,
		SilenceUsage:       true,
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		RunE:               runRun,
	}

	pCmd := &cobra.Command{
		Use:           "p <提交备注>",
		Short:         "提交所有修改并推送",
		Long:          "将本地所有修改暂存并提交到本地仓库，然后推送到远程，提交信息为指定备注。",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.MinimumNArgs(1),
		RunE:          runP,
	}

	kbCmd := &cobra.Command{
		Use:           "kb",
		Short:         "知识库挂载与状态",
		Long:          "知识库放在仓库之外，通过符号链接挂载到工作树，让每个工作树的 AI 复用同一份项目知识。",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	kbAttachCmd := &cobra.Command{
		Use:           "attach [工作树路径]",
		Short:         "把知识库挂载到工作树",
		Long:          "在当前（或指定）工作树生成 Codex、DSH、Qoder 共用的 AGENTS.md 与技能入口，并把挂载物写入 git info/exclude。主工作树直接指向基座，特性分支合并基座与分支说明。",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.MaximumNArgs(1),
		RunE:          runKbAttach,
	}

	kbStatusCmd := &cobra.Command{
		Use:           "status",
		Short:         "查看各工作树的挂载状态",
		Long:          "列出当前仓库全部工作树，并分别显示 Codex、DSH、Qoder 的知识库接入状态，不修改任何文件。",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE:          runKbStatus,
	}

	kbCmd.AddCommand(kbAttachCmd, kbStatusCmd)

	rootCmd.AddCommand(newCMTCommand(), newGMTCommand(), fromCmd, pCmd, newWTCommand(), runCmd, newRMCommand(), kbCmd)
	return rootCmd
}

func returnCommandError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type exitCodeError struct {
	code int
}

func (e *exitCodeError) Error() string {
	return fmt.Sprintf("exit status %d", e.code)
}
