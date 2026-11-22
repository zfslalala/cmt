package cmt

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/codex"
	"github.com/zfslalala/cmt/internal/dsh"
	"github.com/zfslalala/cmt/internal/git"
	"github.com/zfslalala/cmt/internal/idea"
	"github.com/zfslalala/cmt/internal/kb"
	"github.com/zfslalala/cmt/internal/projectconfig"
	"github.com/zfslalala/cmt/internal/qoder"
	"github.com/zfslalala/cmt/internal/workbuddy"
)

type wtOptions struct {
	openIdea      bool
	openWorkBuddy bool
	openDSH       bool
	openQoder     bool
	openCodex     bool
	from          string
	note          string
}

func newWTCommand() *cobra.Command {
	opts := wtOptions{}
	cmd := &cobra.Command{
		Use:           "wt <branch>",
		Short:         "创建分支和工作树",
		Long:          "创建新分支并在当前工作树根目录的同级目录创建对应工作树。起点用 --from 指定 origin 上已存在的分支（如 release 或 origin/release），远程不存在时报错；也可配置 qg.hjson 的 wt.base，都没配则基于当前 HEAD。可用 -i 在 IntelliJ IDEA 中打开，-w 在 WorkBuddy 中打开，-d 在 DSH Desktop 中打开并创建工作区，-q 在 Qoder 中打开工作区，-c 在 Codex Desktop 中打开工作区。",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWt(args[0], opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.openIdea, "idea", "i", false, "创建后在 IntelliJ IDEA 中打开工作树")
	cmd.Flags().BoolVarP(&opts.openWorkBuddy, "workbuddy", "w", false, "创建后在 WorkBuddy 中打开工作树（新任务草稿）")
	cmd.Flags().BoolVarP(&opts.openDSH, "dsh", "d", false, "创建后在 DSH Desktop 中打开工作树并创建工作区")
	cmd.Flags().BoolVarP(&opts.openQoder, "qoder", "q", false, "创建后在 Qoder 中打开工作树并创建工作区")
	cmd.Flags().BoolVarP(&opts.openCodex, "codex", "c", false, "创建后在 Codex Desktop 中打开工作树")
	cmd.Flags().StringVar(&opts.from, "from", "", "origin 上已存在的起点分支（如 release 或 origin/release；不存在时报错）")
	cmd.Flags().StringVarP(&opts.note, "note", "n", "", "本需求一句话说明，写进工作树的 AGENTS.local.md")
	return cmd
}

// runWt 创建新分支与对应工作树，可用 -i / -w / -d 在对应工具中打开
func runWt(branch string, opts wtOptions) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("当前目录不是 Git 仓库")
	}

	// 起点：--from 优先，其次 qg.hjson 的 wt.base，都为空则基于当前 HEAD
	start := opts.from
	if start == "" {
		base, err := projectconfig.WorktreeBase()
		if err != nil {
			return fmt.Errorf("读取工作树默认起点失败: %w", err)
		}
		start = base
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
	attachKB(path, branch, resolvedStart, opts.note)

	if opts.openIdea {
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

	if opts.openWorkBuddy {
		if err := workbuddy.OpenTask(path, ""); err != nil {
			// 工作树已创建成功，打开失败只警告不报错
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Println("已用 WorkBuddy 打开工作树（新任务草稿）")
		}
	}

	if opts.openDSH {
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

	if opts.openQoder {
		via, err := qoder.OpenWorkspace(path)
		if err != nil {
			// 工作树已创建成功，打开失败只警告不报错
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintf(os.Stderr, "  工作树已就绪，可手动在 Qoder 中打开: %s\n", path)
		} else {
			fmt.Printf("已在 %s 中打开工作树: %s\n", via, path)
			if via == "Qoder Desktop" {
				fmt.Println("（Qoder Desktop 若未自动创建工作区，请在该应用中手动打开上述目录）")
			}
		}
	}

	if opts.openCodex {
		via, err := codex.OpenWorkspace(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintf(os.Stderr, "  工作树已就绪，可手工执行: codex app %s\n", path)
		} else {
			fmt.Printf("已在 %s 中打开工作树: %s\n", via, path)
		}
	}
	return nil
}

// attachKB 给新建的工作树挂载知识库，并写入本工作树专属的说明文件。
// 任何一步失败都只警告，不阻断工作树创建流程。
func attachKB(path, branch, start, note string) {
	layout, err := kb.Attach(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ 知识库挂载失败: %v\n", err)
		fmt.Fprintf(os.Stderr, "  工作树已就绪，可稍后手工执行: qg kb attach %s\n", path)
		return
	}

	if layout.GlobalCreated {
		fmt.Printf("知识库全局层已创建: %s\n", layout.Root)
	}
	if layout.Bootstrapped {
		fmt.Printf("知识库尚未初始化，已自动创建骨架: %s\n", layout.Base)
		fmt.Println("（请补充 AGENTS.md 与 .dsh/skills 下的技能，否则 AI 只知道这是个空项目）")
	}
	fmt.Printf("已挂载知识库: %s（项目 %s）\n", layout.Root, layout.Project)
	fmt.Println("已接入: Codex (.agents) / DSH (.dsh) / Qoder (.qoder)")
	if layout.Branch != "" {
		fmt.Printf("分支知识目录: %s\n", layout.Branch)
	}

	written, err := kb.WriteLocalNote(layout, kb.NoteContent{
		Branch:  branch,
		Start:   start,
		Note:    note,
		Created: time.Now(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ 写入 %s 失败: %v\n", kb.LocalNoteFile, err)
		return
	}
	if written {
		fmt.Printf("已生成 %s（写清本需求目标后 AI 才分得清你在做什么）\n", kb.LocalNoteFile)
		if err := kb.SyncWorktreeInstructions(&layout); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ 同步工作树 AGENTS.md 失败: %v\n", err)
		}
	}

	for _, warning := range layout.Warnings {
		fmt.Fprintf(os.Stderr, "警告: %s\n", warning)
	}
}
