package git

import (
	"fmt"
	"strings"
)

// DeletedBranch 描述一次分支删除的结果
type DeletedBranch struct {
	Branch        string // 被删除的分支
	WorktreePath  string // 一并删除的工作树路径，无则空
	SwitchedTo    string // 删除前切换到的分支（分支检出于主仓库时），无则空
	RemoteDeleted bool   // 是否成功删除远程分支
	RemoteMessage string // 远程分支删除失败的警告信息
}

// DeleteBranch 删除本地分支；若分支检出于链接工作树则一并删除工作树。
// remote 为 true 时同时删除远程同名分支（失败仅警告，不影响本地删除结果）。
// force 为 true 时跳过脏工作树检查，并用 -D 忽略未合并检查。
// 所有 git 操作在主工作树根目录执行——即使当前 shell 位于被删的工作树内也不受影响。
func DeleteBranch(branch string, remote, force bool) (*DeletedBranch, error) {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return nil, fmt.Errorf("分支名不能为空")
	}

	wts, err := Worktrees()
	if err != nil {
		return nil, err
	}
	if len(wts) == 0 {
		return nil, fmt.Errorf("获取工作树列表失败")
	}
	// porcelain 输出的第一条是主工作树
	mainWt := wts[0]
	res := &DeletedBranch{Branch: branch}

	// 待删分支检出于主仓库：先切到本地默认分支（main/master），再执行删除
	if mainWt.Branch == branch {
		target, err := defaultLocalBranch(mainWt.Path, branch)
		if err != nil {
			return nil, fmt.Errorf("分支 %s 正被主仓库检出，%w", branch, err)
		}
		if err := requireCleanWorktree(mainWt.Path); err != nil {
			return nil, fmt.Errorf("切换分支前需主仓库无未提交变更: %w", err)
		}
		if _, err := runGitInDir(mainWt.Path, "checkout", target); err != nil {
			return nil, fmt.Errorf("切换到 %s 失败: %w", target, err)
		}
		res.SwitchedTo = target
	}

	// 分支所在的链接工作树（如有）
	var wtPath string
	for _, wt := range wts[1:] {
		if wt.Branch == branch {
			wtPath = wt.Path
			break
		}
	}

	// 删除链接工作树
	if wtPath != "" {
		if hasChanges, err := hasUncommittedChangesInDir(wtPath); err == nil && hasChanges && !force {
			return nil, fmt.Errorf("工作树 %s 有未提交变更，请先处理或使用 --force 强制删除", wtPath)
		}
		args := []string{"worktree", "remove"}
		if force {
			args = append(args, "--force")
		}
		args = append(args, wtPath)
		if _, err := runGitInDir(mainWt.Path, args...); err != nil {
			return nil, fmt.Errorf("删除工作树 %s 失败: %w", wtPath, err)
		}
		res.WorktreePath = wtPath
	}

	// 删除本地分支（-d 拒绝未合并分支，防止丢代码；--force 升级为 -D）
	delFlag := "-d"
	if force {
		delFlag = "-D"
	}
	if _, err := runGitInDir(mainWt.Path, "branch", delFlag, branch); err != nil {
		if !force && strings.Contains(err.Error(), "not fully merged") {
			return nil, fmt.Errorf("分支 %s 有未合并的提交，确认放弃请使用 --force", branch)
		}
		return nil, fmt.Errorf("删除分支失败: %w", err)
	}

	// 删除远程同名分支
	if remote {
		if _, err := runGitInDir(mainWt.Path, "push", defaultRemote, "--delete", branch); err != nil {
			res.RemoteMessage = fmt.Sprintf("远程分支删除失败（本地分支已删除）: %v", err)
		} else {
			res.RemoteDeleted = true
		}
	}

	return res, nil
}

// defaultLocalBranch 返回本地的默认分支名：优先 main，其次 master。
// exclude 用于排除待删除分支自身（例如仓库只有 master 而要删的就是 master）。
func defaultLocalBranch(dir, exclude string) (string, error) {
	for _, name := range []string{"main", "master"} {
		if name == exclude {
			continue
		}
		if _, err := runGitInDir(dir, "rev-parse", "--verify", "--quiet", "refs/heads/"+name); err == nil {
			return name, nil
		}
	}
	return "", fmt.Errorf("本地不存在可切换的默认分支（main/master）")
}
