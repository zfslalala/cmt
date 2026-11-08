package cmt

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/git"
)

// Execute 构建命令树并执行
func Execute() {
	rootCmd := &cobra.Command{
		Use:           "qg",
		Short:         "Quick Git:快速 Git 操作工具",
		Long:          "快速 Git 操作工具，支持 AI 自动生成 commit message，兼容所有 OpenAI 协议的大模型服务。",
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
		Long:          "将目标分支同步到远程最新，再把当前分支合并进去，推送目标分支后回到当前分支。",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ExactArgs(1),
		RunE:          runGMT,
	}
	gmtCmd.Flags().BoolVarP(&buildFlag, "build", "b", false, "推送成功后触发构建（读取仓库根 qg.hjson）")

	fromCmd := &cobra.Command{
		Use:           "from",
		Short:         "查看当前分支来源信息",
		Long:          "显示当前分支的来源分支与切出时间（基于 reflog 推断）。",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE:          runFrom,
	}

	wtCmd := &cobra.Command{
		Use:           "wt <branch>",
		Short:         "创建分支和工作树",
		Long:          "创建新分支并在当前目录的同级目录创建对应工作树。起点用 --from 指定（先 fetch 远端最新，如 origin/release），或配置 qg.hjson 的 wt.base；都没配则基于当前 HEAD。可用 -i 在 IntelliJ IDEA 中打开，-w 在 WorkBuddy 中打开。",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ExactArgs(1),
		RunE:          runWt,
	}
	wtCmd.Flags().BoolVarP(&wtOpenIdea, "idea", "i", false, "创建后在 IntelliJ IDEA 中打开工作树")
	wtCmd.Flags().BoolVarP(&wtOpenWorkBuddy, "workbuddy", "w", false, "创建后在 WorkBuddy 中打开工作树（新任务草稿）")
	wtCmd.Flags().StringVar(&wtFrom, "from", "", "新分支的起点分支（fetch 远端最新后切出，如 release）")

	runCmd := &cobra.Command{
		Use:                "run <脚本> [参数...]",
		Short:              "执行脚本并输出到控制台",
		Long:               "执行 Go 脚本（go run），stdout/stderr 实时打印到控制台，退出码透传。脚本可直接给路径，或放 ~/.qg/scripts/（QG_SCRIPTS_DIR 可覆盖）后用名字调用。",
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

	rmCmd := &cobra.Command{
		Use:           "rm",
		Short:         "删除当前分支及对应工作树",
		Long:          "删除当前所在分支，若分支检出于工作树则一并删除工作树，主仓库自动切回默认分支。-o 同时删除远程同名分支，--force 强制跳过脏工作树与未合并检查。",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE:          runRm,
	}
	rmCmd.Flags().BoolVarP(&rmRemote, "origin", "o", false, "同时删除远程 origin 上的同名分支")
	rmCmd.Flags().BoolVar(&rmForce, "force", false, "强制删除：脏工作树、未合并分支也删")

	rootCmd.AddCommand(cmtCmd, gmtCmd, fromCmd, pCmd, wtCmd, runCmd, rmCmd)

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
