package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Worktree 描述 git worktree list --porcelain 中的一条工作树记录。
type Worktree struct {
	Path           string
	Head           string
	Branch         string
	Detached       bool
	Bare           bool
	Locked         bool
	LockedReason   string
	Prunable       bool
	PrunableReason string
}

// RepoRootIn 获取指定目录所在工作树的根目录。
func RepoRootIn(dir string) (string, error) {
	output, err := runGitInDir(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// CurrentBranchIn 获取指定目录的当前分支名。
func CurrentBranchIn(dir string) (string, error) {
	output, err := runGitInDir(dir, "branch", "--show-current")
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(output)
	if branch == "" {
		return "", fmt.Errorf("当前处于 detached HEAD 状态，无法确定当前分支")
	}

	return branch, nil
}

// WorktreePaths 返回仓库全部工作树的绝对路径，主工作树位于首位。
// git worktree list 本身以主工作树开头，这里保持该顺序。
func WorktreePaths(dir string) ([]string, error) {
	worktrees, err := WorktreesIn(dir)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(worktrees))
	for _, worktree := range worktrees {
		paths = append(paths, worktree.Path)
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("未找到任何工作树")
	}

	return paths, nil
}

// Worktrees 获取当前仓库的工作树列表。
func Worktrees() ([]Worktree, error) {
	return WorktreesIn("")
}

// WorktreesIn 获取 dir 所属仓库的工作树列表。
func WorktreesIn(dir string) ([]Worktree, error) {
	output, err := runGitInDir(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktreesPorcelain(output), nil
}

func parseWorktreesPorcelain(output string) []Worktree {
	var worktrees []Worktree
	var current Worktree
	flush := func() {
		if current.Path != "" {
			worktrees = append(worktrees, current)
			current = Worktree{}
		}
	}

	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			flush()
			continue
		}
		switch {
		case strings.HasPrefix(line, "worktree "):
			current.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "HEAD "):
			current.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch refs/heads/"):
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		case line == "detached":
			current.Detached = true
		case line == "bare":
			current.Bare = true
		case line == "locked":
			current.Locked = true
		case strings.HasPrefix(line, "locked "):
			current.Locked = true
			current.LockedReason = strings.TrimPrefix(line, "locked ")
		case line == "prunable":
			current.Prunable = true
		case strings.HasPrefix(line, "prunable "):
			current.Prunable = true
			current.PrunableReason = strings.TrimPrefix(line, "prunable ")
		}
	}
	flush()
	return worktrees
}

// CommonDir 返回仓库共享的 git 目录，所有工作树共用同一个。
// 用于读写 info/exclude、info/kb-project 这类不随工作树变化的本地文件。
// --git-common-dir 在主工作树下返回相对路径，这里统一转成绝对路径。
func CommonDir(dir string) (string, error) {
	output, err := runGitInDir(dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}

	common := strings.TrimSpace(output)
	if common == "" {
		return "", fmt.Errorf("无法确定 git 共享目录")
	}

	if !filepath.IsAbs(common) {
		base := dir
		if base == "" {
			if base, err = os.Getwd(); err != nil {
				return "", fmt.Errorf("获取当前目录失败: %w", err)
			}
		}
		common = filepath.Join(base, common)
	}

	return filepath.Clean(common), nil
}
