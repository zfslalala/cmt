package cmt

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/kb"
)

// runKbAttach 把知识库挂载到指定工作树，缺省为当前目录
func runKbAttach(cmd *cobra.Command, args []string) error {
	target := "."
	if len(args) > 0 {
		target = args[0]
	}

	layout, err := kb.Attach(target)
	if err != nil {
		return err
	}

	if layout.GlobalCreated {
		fmt.Printf("知识库全局层已创建: %s\n", layout.Root)
	}
	if layout.Bootstrapped {
		fmt.Printf("知识库尚未初始化，已自动创建骨架: %s\n", layout.Base)
		fmt.Println("（请补充 AGENTS.md 与 .dsh/skills 下的技能，否则 AI 只知道这是个空项目）")
	}

	fmt.Printf("知识库: %s\n", layout.Root)
	fmt.Printf("项目:   %s\n", layout.Project)
	fmt.Printf("基座:   %s\n", layout.Base)
	fmt.Printf("工作树: %s\n", layout.Worktree)

	if layout.IsMain {
		fmt.Println("模式:   主工作树，.dsh 直接指向基座")
	} else {
		fmt.Printf("分支:   %s\n", layout.BranchName)
		fmt.Printf("分支知识: %s\n", layout.Branch)
		fmt.Println("模式:   特性分支，.dsh 为合成视图（技能分支覆盖基座，知识写入分支目录）")
	}

	for _, warning := range layout.Warnings {
		fmt.Fprintf(os.Stderr, "警告: %s\n", warning)
	}

	return nil
}

// runKbStatus 列出当前仓库全部工作树的挂载状态
func runKbStatus(cmd *cobra.Command, args []string) error {
	items, project, err := kb.Status(".")
	if err != nil {
		return err
	}

	fmt.Printf("项目: %s\n", project)
	for _, item := range items {
		mark := "✗"
		if item.Mounted {
			mark = "✓"
		}
		fmt.Printf("  %s %s    %s\n", mark, item.Path, item.Detail)
	}

	return nil
}
