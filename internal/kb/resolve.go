package kb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zfslalala/cmt/internal/git"
)

// Layout 描述一次挂载在磁盘上的全部位置。
type Layout struct {
	// Root 是知识库根目录。
	Root string
	// Project 是项目键，对应 projects/<Project>。
	Project string
	// Base 是基座目录。
	Base string
	// Branch 是分支知识目录，主工作树为空。
	Branch string
	// BranchName 是当前分支名。
	BranchName string
	// Worktree 是被挂载的工作树根目录。
	Worktree string
	// MainWorktree 是仓库的主工作树根目录。
	MainWorktree string
	// IsMain 表示被挂载的是主工作树。
	IsMain bool
	// Bootstrapped 表示这次挂载顺带创建了知识库骨架。
	Bootstrapped bool
	// GlobalCreated 表示这次挂载顺带创建了知识库全局层。
	GlobalCreated bool
	// Warnings 收集不致命、但需要人知道的情况，例如已存在的真实文件未被覆盖。
	Warnings []string
}

// DshPath 返回工作树里 .dsh 的路径。
func (l Layout) DshPath() string {
	return filepath.Join(l.Worktree, DshDir)
}

// InstructionsPath 返回工作树根 AGENTS.md 的路径。
func (l Layout) InstructionsPath() string {
	return filepath.Join(l.Worktree, "AGENTS.md")
}

// CodexSkillsPath 返回 Codex 会发现的共享项目技能入口。
func (l Layout) CodexSkillsPath() string {
	return filepath.Join(l.Worktree, SharedAgentsDir, "skills")
}

// QoderSkillsPath 返回 Qoder 会发现的共享项目技能入口。
func (l Layout) QoderSkillsPath() string {
	return filepath.Join(l.Worktree, QoderDir, "skills")
}

// ProjectPath 返回该项目在知识库里的根目录。
func (l Layout) ProjectPath() string {
	return filepath.Join(l.Root, "projects", l.Project)
}

// Resolve 解析目录应挂载到知识库的哪个位置，不写任何文件。
// 传入工作树内任意子目录都可以，会先归一到该工作树的根。
func Resolve(dir string) (Layout, error) {
	var layout Layout

	worktree, err := git.RepoRootIn(dir)
	if err != nil {
		return layout, fmt.Errorf("当前目录不是 Git 仓库: %w", err)
	}

	main, err := mainWorktreeIn(worktree)
	if err != nil {
		return layout, err
	}

	project, err := projectKey(main)
	if err != nil {
		return layout, err
	}

	root, err := rootDir(main)
	if err != nil {
		return layout, err
	}

	branch, err := git.CurrentBranchIn(worktree)
	if err != nil {
		return layout, err
	}

	layout = Layout{
		Root:         root,
		Project:      project,
		Base:         filepath.Join(root, "projects", project, baseDirName),
		BranchName:   branch,
		Worktree:     worktree,
		MainWorktree: main,
		IsMain:       filepath.Clean(worktree) == filepath.Clean(main),
	}

	if !layout.IsMain {
		layout.Branch = filepath.Join(layout.ProjectPath(), branchesDirName, git.SanitizeBranchName(branch))
	}

	return layout, nil
}

// BaseDir 返回目录所属项目在知识库里的基座目录（<根>/projects/<项目>/base）。
// 只定位项目级位置，不要求当前分支可解析，供只需要项目级配置的调用方使用。
// 目录不在 Git 仓库中时返回错误；基座目录本身不保证已存在。
func BaseDir(dir string) (string, error) {
	worktree, err := git.RepoRootIn(dir)
	if err != nil {
		return "", fmt.Errorf("当前目录不是 Git 仓库: %w", err)
	}

	main, err := mainWorktreeIn(worktree)
	if err != nil {
		return "", err
	}

	project, err := projectKey(main)
	if err != nil {
		return "", err
	}

	root, err := rootDir(main)
	if err != nil {
		return "", err
	}

	return filepath.Join(root, "projects", project, baseDirName), nil
}

// mainWorktreeIn 返回主工作树的绝对路径；git worktree list 的首项即主工作树。
func mainWorktreeIn(worktree string) (string, error) {
	paths, err := git.WorktreePaths(worktree)
	if err != nil {
		return "", err
	}

	main, err := filepath.Abs(paths[0])
	if err != nil {
		return "", fmt.Errorf("解析主工作树路径失败: %w", err)
	}

	return main, nil
}

// projectKey 决定项目键：优先读仓库本地的 info/kb-project，
// 否则退化为主工作树的目录名。
func projectKey(mainWorktree string) (string, error) {
	common, err := git.CommonDir(mainWorktree)
	if err != nil {
		return "", err
	}
	path := filepath.Join(common, "info", projectKeyFile)
	data, readErr := os.ReadFile(path)
	if readErr == nil {
		if key := strings.TrimSpace(string(data)); key != "" {
			if !validProjectKey(key) {
				return "", fmt.Errorf("%s 包含非法项目名 %q", path, key)
			}
			return key, nil
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return "", fmt.Errorf("读取 %s 失败: %w", path, readErr)
	}

	key := filepath.Base(filepath.Clean(mainWorktree))
	if key == "" || key == "." || key == string(filepath.Separator) {
		return "", fmt.Errorf("无法从主工作树路径推导项目名: %s", mainWorktree)
	}

	return key, nil
}

func validProjectKey(key string) bool {
	return key != "." &&
		key != ".." &&
		filepath.Base(key) == key &&
		!strings.ContainsAny(key, `/\`)
}

// rootDir 决定知识库根目录：环境变量优先，其次仓库同级的 _ai-kb，
// 最后回退到用户目录下的 .ai-kb。
func rootDir(mainWorktree string) (string, error) {
	if value := strings.TrimSpace(os.Getenv("AI_KB_ROOT")); value != "" {
		return filepath.Abs(value)
	}

	sibling := filepath.Join(filepath.Dir(filepath.Clean(mainWorktree)), defaultRootName)
	if isDir(sibling) {
		return sibling, nil
	}

	if home, err := os.UserHomeDir(); err == nil {
		fallback := filepath.Join(home, ".ai-kb")
		if isDir(fallback) {
			return fallback, nil
		}
	}

	// 都不存在时返回默认候选，由 Attach 给出缺少基座的明确错误。
	return sibling, nil
}

// Attach 把知识库挂载到目录所在的工作树。
// 普通文件不会被覆盖；无法识别的真实 .dsh 目录会直接报错保留。
