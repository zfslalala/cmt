package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/z/cmt/internal/git"
	"github.com/z/cmt/internal/llm"
	"github.com/z/cmt/internal/prompt"
	"github.com/z/cmt/pkg/config"
)

var (
	verbose bool
	model   string
	push    bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "cmt",
		Short: "Git Commit Message 自动生成工具",
		Long:  "基于大语言模型的 Git Commit Message 自动生成工具，支持所有 OpenAI 协议兼容的大模型服务。",
		Run:   run,
	}

	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "生成详细的 commit message")
	rootCmd.Flags().StringVarP(&model, "model", "m", "", "指定使用的模型")
	rootCmd.Flags().BoolVarP(&push, "push", "p", false, "提交并推送到远程仓库")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	// Debug: 打印当前参数状态
	//fmt.Printf("DEBUG: verbose=%v, model='%s', push=%v, args=%v\n", verbose, model, push, args)

	// 1. 加载配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置加载失败: %v\n", err)
		os.Exit(1)
	}

	// 命令行参数覆盖配置
	if model != "" {
		cfg.Model = model
	}

	// 2. 检查是否在 Git 仓库中
	if !git.IsGitRepo() {
		fmt.Fprintln(os.Stderr, "错误: 当前目录不是 Git 仓库")
		os.Exit(1)
	}

	// 3. 获取暂存区变更，如果没有暂存则自动暂存所有变更
	changes, err := git.GetStagedChanges()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取变更失败: %v\n", err)
		os.Exit(1)
	}

	// 如果暂存区为空，自动执行 git add .
	if len(changes.Files) == 0 {
		fmt.Println("暂存区为空，自动暂存所有变更...")
		if err := git.AddAll(); err != nil {
			fmt.Fprintf(os.Stderr, "自动暂存失败: %v\n", err)
			os.Exit(1)
		}
		// 重新获取变更
		changes, err = git.GetStagedChanges()
		if err != nil || len(changes.Files) == 0 {
			fmt.Fprintln(os.Stderr, "暂存后仍未检测到变更")
			os.Exit(1)
		}
		fmt.Printf("已暂存 %d 个文件\n", len(changes.Files))
	}

	// 4. 构建提示词
	systemPrompt := prompt.GetSystemPrompt()
	userPrompt := prompt.BuildUserPrompt(changes, verbose)

	// 5. 调用 LLM API
	client := llm.NewClient(cfg)
	message, err := client.Chat(systemPrompt, userPrompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成失败: %v\n", err)
		os.Exit(1)
	}

	// 6. 输出结果
	fmt.Println(message)

	// 7. 如果指定了 -p 参数，执行 commit 和 push
	if push {
		fmt.Println("正在提交...")
		if err := git.Commit(message); err != nil {
			fmt.Fprintf(os.Stderr, "提交失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("正在推送...")
		if err := git.Push(); err != nil {
			fmt.Fprintf(os.Stderr, "推送失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("提交并推送成功!")
	}
}
