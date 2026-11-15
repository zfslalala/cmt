package cmt

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/dsh"
	"github.com/zfslalala/cmt/internal/git"
	"github.com/zfslalala/cmt/internal/kb"
)

var (
	rmRemote bool
	rmForce  bool
)

// runRm 删除当前所在分支，检出于工作树则一并删除
func runRm(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("当前目录不是 Git 仓库")
	}

	branch, err := git.CurrentBranch()
	if err != nil {
		return err
	}

	// 远程删除不可逆，显式确认
	if rmRemote {
		fmt.Printf("即将删除本地及远程分支: %s 和 origin/%s\n", branch, branch)
		fmt.Print("确认删除远程分支? (y/N): ")
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			return fmt.Errorf("已取消")
		}
	}

	// 先记下分支知识目录：工作树删掉后就无法再从目录反查它属于哪个分支。
	var kbLayout kb.Layout
	hasKB := false
	if layout, layoutErr := kb.Resolve("."); layoutErr == nil && !layout.IsMain {
		kbLayout, hasKB = layout, true
	}

	res, err := git.DeleteBranch(branch, rmRemote, rmForce)
	if err != nil {
		return err
	}

	if res.SwitchedTo != "" {
		fmt.Printf("已切换到分支: %s\n", res.SwitchedTo)
	}
	if res.WorktreePath != "" {
		fmt.Printf("已删除工作树: %s\n", res.WorktreePath)
		fmt.Println("（若当前终端位于被删目录内，请先 cd 出来）")
	}
	fmt.Printf("已删除分支: %s\n", res.Branch)
	if res.RemoteDeleted {
		fmt.Printf("已删除远程分支: origin/%s\n", res.Branch)
	}
	if res.RemoteMessage != "" {
		fmt.Fprintln(os.Stderr, res.RemoteMessage)
	}

	// 分支知识目录：空的直接清掉，有内容就保留并告知位置，避免误删经验。
	if res.WorktreePath != "" && hasKB {
		removed, kbErr := kb.CleanupBranchKnowledge(kbLayout)
		switch {
		case kbErr != nil:
			fmt.Fprintf(os.Stderr, "⚠ 清理分支知识失败: %v\n", kbErr)
		case removed:
			fmt.Printf("已清理空的分支知识目录: %s\n", kbLayout.Branch)
		default:
			fmt.Printf("分支知识已保留: %s\n", kbLayout.Branch)
			fmt.Println("（有实际内容，确认无需保留后自行删除）")
		}
	}

	// 工作树删除后，同步移除 DSH 中指向它的工作区。DSH 未运行或未安装
	// 插件时静默跳过，不影响已经完成的本地删除。
	if res.WorktreePath != "" {
		removed, dshErr := dsh.RemoveWorkspace(res.WorktreePath)
		if dshErr != nil {
			fmt.Fprintf(os.Stderr, "⚠ 删除 DSH 工作区失败（本地分支与工作树已删除）: %v\n", dshErr)
		} else if removed.Removed {
			fmt.Printf("已删除 DSH 工作区: %s\n", removed.WorkspaceID)
		}
	}
	return nil
}
