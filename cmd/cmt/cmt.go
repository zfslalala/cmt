package cmt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/git"
	"github.com/zfslalala/cmt/internal/llm"
	"github.com/zfslalala/cmt/internal/prompt"
	"github.com/zfslalala/cmt/pkg/config"
)

var (
	verbose bool
	push    bool
	edit    bool
)

func runCMT(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("配置加载失败: %w", err)
	}

	if !git.IsGitRepo() {
		return fmt.Errorf("当前目录不是 Git 仓库")
	}

	changes, err := git.GetStagedChanges()
	if err != nil {
		return fmt.Errorf("获取变更失败: %w", err)
	}

	if len(changes.Files) == 0 {
		fmt.Println("暂存区为空，自动暂存所有变更...")
		if err := git.AddAll(); err != nil {
			return fmt.Errorf("自动暂存失败: %w", err)
		}

		changes, err = git.GetStagedChanges()
		if err != nil {
			return fmt.Errorf("获取暂存后的变更失败: %w", err)
		}
		if len(changes.Files) == 0 {
			return fmt.Errorf("暂存后仍未检测到变更")
		}
		fmt.Printf("已暂存 %d 个文件:\n", len(changes.Files))
		for _, file := range changes.Files {
			fmt.Printf("  %s %s\n", git.GetStatusSymbol(file.Status), file.NewPath)
		}
	}

	client := llm.NewClient(cfg)
	message, err := client.Chat(prompt.GetSystemPrompt(), prompt.BuildUserPrompt(changes, verbose))
	if err != nil {
		return fmt.Errorf("生成失败: %w", err)
	}

	fmt.Println(message)
	if edit {
		fmt.Println("\n请确认或修改 commit message（直接回车使用上述内容；输入新内容后以空行结束）:")
		editedMessage, err := readMultiline(os.Stdin)
		if err != nil {
			return fmt.Errorf("读取编辑后的 commit message 失败: %w", err)
		}
		if editedMessage != "" {
			message = editedMessage
		}
	}

	if edit || push {
		fmt.Println("正在提交本地仓库...")
		if err := git.Commit(message); err != nil {
			return fmt.Errorf("提交失败: %w", err)
		}
		fmt.Println("提交成功!")
	}

	if push {
		fmt.Println("正在推送远程仓库...")
		if err := git.Push(); err != nil {
			return fmt.Errorf("推送失败: %w", err)
		}
		fmt.Println("推送成功!")
	}

	return nil
}

// readMultiline 从 r 读取多行输入，空行或 EOF 结束。
// 返回去掉首尾空白的拼接内容；未输入任何内容时返回空串。
func readMultiline(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" && len(lines) > 0 {
			break
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}
