package kb

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/zfslalala/cmt/internal/git"
)

// WorktreeStatus 是单个工作树的挂载状态。
type WorktreeStatus struct {
	// Path 是工作树根目录。
	Path string
	// Mounted 表示 AGENTS.md 与 .dsh 都已就位。
	Mounted bool
	// IsMain 表示这是主工作树。
	IsMain bool
	// Detail 是供人阅读的一句话说明。
	Detail string
	// Prunable 表示工作树已失效，可通过 git worktree prune 清理。
	Prunable bool
	// Codex 表示 Codex 的指令与项目技能入口已就绪。
	Codex bool
	// DSH 表示 DSH 的 AGENTS.md 与 .dsh 合成视图已就绪。
	DSH bool
	// Qoder 表示 Qoder 的 AGENTS.md 与 .qoder/skills 已就绪。
	Qoder bool
}

// Status 列出仓库全部工作树的挂载状态，不写任何文件。
func Status(dir string) ([]WorktreeStatus, string, error) {
	layout, err := Resolve(dir)
	if err != nil {
		return nil, "", err
	}

	worktrees, err := git.WorktreesIn(layout.Worktree)
	if err != nil {
		return nil, "", err
	}

	items := make([]WorktreeStatus, 0, len(worktrees))
	for _, worktree := range worktrees {
		item := WorktreeStatus{
			Path:     worktree.Path,
			IsMain:   filepath.Clean(worktree.Path) == layout.MainWorktree,
			Prunable: worktree.Prunable,
		}
		if worktree.Prunable {
			item.Detail = "工作树已失效，可执行 git worktree prune 清理"
			items = append(items, item)
			continue
		}

		instructions := instructionsStatus(worktree.Path, layout.Base)
		dshOK := pathExists(filepath.Join(worktree.Path, DshDir))
		codexSkillsOK := linkPointsTo(filepath.Join(worktree.Path, SharedAgentsDir, "skills"), filepath.Join(worktree.Path, DshDir, "skills"))
		qoderSkillsOK := linkPointsTo(filepath.Join(worktree.Path, QoderDir, "skills"), filepath.Join(worktree.Path, DshDir, "skills"))
		item.Codex = instructions && dshOK && codexSkillsOK
		item.DSH = instructions && dshOK
		item.Qoder = instructions && dshOK && qoderSkillsOK
		item.Mounted = item.Codex && item.DSH && item.Qoder

		switch {
		case item.Mounted && item.IsMain:
			item.Detail = "Codex / DSH / Qoder 已接入共享基座"
		case item.Mounted:
			item.Detail = "Codex / DSH / Qoder 已接入基座与分支说明"
		case !instructions:
			item.Detail = "AGENTS.md 未接管或不可用"
		case !dshOK:
			item.Detail = "缺少 .dsh 合成视图"
		case !codexSkillsOK:
			item.Detail = "缺少 .agents/skills 入口"
		case !qoderSkillsOK:
			item.Detail = "缺少 .qoder/skills 入口"
		default:
			item.Detail = "部分接入"
		}

		items = append(items, item)
	}

	return items, layout.Project, nil
}
func instructionsStatus(worktree, base string) bool {
	path := filepath.Join(worktree, "AGENTS.md")
	if !pathExists(path) {
		return false
	}
	if isSymlink(path) {
		target, err := resolvedLinkTarget(path)
		return err == nil && filepath.Clean(target) == filepath.Clean(filepath.Join(base, "AGENTS.md"))
	}
	data, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(data), instructionsMarker)
}

func linkPointsTo(path, target string) bool {
	if !pathExists(path) || !isSymlink(path) {
		return false
	}
	got, err := resolvedLinkTarget(path)
	if err != nil || filepath.Clean(got) != filepath.Clean(target) {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
