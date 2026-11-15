package cmt

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/build"
	"github.com/zfslalala/cmt/internal/dsh"
	"github.com/zfslalala/cmt/internal/git"
	"github.com/zfslalala/cmt/internal/idea"
	"github.com/zfslalala/cmt/internal/kb"
	"github.com/zfslalala/cmt/internal/workbuddy"
)

var (
	wtOpenIdea      bool
	wtOpenWorkBuddy bool
	wtOpenDSH       bool
	wtFrom          string
	wtNote          string
)

// runWt 创建新分支与对应工作树，可用 -i / -w / -d 在对应工具中打开
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

	// 挂载知识库并写本工作树的说明文件。工作树已经建好，这里失败只警告，
	// 不把整个命令判为失败；用户仍可后续手工跑 qg kb attach。
	attachKB(path, branch, resolvedStart)

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

	if wtOpenDSH {
		res, err := dsh.OpenWorkspace(path)
		if err != nil {
			// 工作树已创建成功，打开失败只警告不报错
			fmt.Fprintln(os.Stderr, err)
		} else if res.Created {
			fmt.Printf("已在 DeepSeek Harness 中创建工作区并打开: %s\n", res.WorkspaceID)
		} else {
			fmt.Printf("已在 DeepSeek Harness 中打开工作区: %s\n", res.WorkspaceID)
		}
	}
	return nil
}

// attachKB 给新建的工作树挂载知识库，并写入本工作树专属的说明文件。
// 任何一步失败都只警告，不阻断工作树创建流程。
func attachKB(path, branch, start string) {
	layout, err := kb.Attach(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ 知识库挂载失败: %v\n", err)
		fmt.Fprintf(os.Stderr, "  工作树已就绪，可稍后手工执行: qg kb attach %s\n", path)
		return
	}

	fmt.Printf("已挂载知识库: %s（项目 %s）\n", layout.Root, layout.Project)
	if layout.Branch != "" {
		fmt.Printf("分支知识目录: %s\n", layout.Branch)
	}

	written, err := kb.WriteLocalNote(layout, kb.NoteContent{
		Branch:  branch,
		Start:   start,
		Note:    wtNote,
		Created: time.Now(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ 写入 %s 失败: %v\n", kb.LocalNoteFile, err)
		return
	}
	if written {
		fmt.Printf("已生成 %s（写清本需求目标后 AI 才分得清你在做什么）\n", kb.LocalNoteFile)
	}

	for _, warning := range layout.Warnings {
		fmt.Fprintf(os.Stderr, "警告: %s\n", warning)
	}
}
