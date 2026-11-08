package cmt

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/git"
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
	return nil
}
