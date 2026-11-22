package kb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zfslalala/cmt/internal/git"
)

// Attach 把知识库挂载到目录所在的工作树。
// 普通文件不会被覆盖；无法识别的真实 .dsh 目录会直接报错保留。
func Attach(dir string) (Layout, error) {
	layout, err := Resolve(dir)
	if err != nil {
		return layout, err
	}

	globalCreated, err := ensureGlobal(&layout)
	if err != nil {
		return layout, err
	}
	layout.GlobalCreated = globalCreated

	if !isDir(layout.Base) {
		created, err := ensureProject(&layout)
		if err != nil {
			return layout, err
		}
		layout.Bootstrapped = created
	}

	if !layout.IsMain {
		if err := ensureBranch(&layout); err != nil {
			return layout, err
		}
		if err := checkSynthesisTarget(layout.DshPath(), layout.Base); err != nil {
			return layout, err
		}
	}

	if layout.IsMain {
		// 主工作树直接使用基座：基座就是在主工作树上编辑的。
		if err := linkPath(layout.Base, layout.DshPath(), &layout); err != nil {
			return layout, err
		}
	} else if err := synthesize(&layout); err != nil {
		return layout, err
	}

	if err := ensureAgentSkillLinks(&layout); err != nil {
		return layout, err
	}
	if err := SyncWorktreeInstructions(&layout); err != nil {
		return layout, err
	}
	if err := ensureExclude(layout); err != nil {
		return layout, err
	}

	return layout, nil
}

// ensureProject 在知识库里还没有这个项目时创建骨架，已有则原样返回。
// 只创建缺失的部分，绝不覆盖已存在的文件。
func ensureProject(layout *Layout) (bool, error) {
	dirs := []string{
		layout.Base,
		filepath.Join(layout.Base, "skills"),
		filepath.Join(layout.Base, "knowledge"),
		filepath.Join(layout.Base, "knowledge", "decisions"),
		layout.ProjectPath(),
		filepath.Join(layout.ProjectPath(), branchesDirName),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return false, fmt.Errorf("创建知识库骨架失败: %w", err)
		}
	}

	files := map[string]string{
		filepath.Join(layout.Base, "AGENTS.md"):                          projectAgentsTemplate(layout.Project),
		filepath.Join(layout.Base, "knowledge", "journal.md"):            "# 需求日志\n\n按时间倒序，最新在上。\n",
		filepath.Join(layout.Base, "knowledge", "glossary.md"):           "# 术语表\n\n| 术语 | 含义 | 别称 / 易混 |\n|---|---|---|\n",
		filepath.Join(layout.Base, "knowledge", "decisions", ".gitkeep"): "",
	}
	for path, content := range files {
		if pathExists(path) {
			continue
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return false, fmt.Errorf("写入知识库骨架文件失败: %w", err)
		}
	}

	return true, nil
}

// projectAgentsTemplate 是新项目基座里 AGENTS.md 的初始内容。
// 刻意保持很短：没有真实内容时，让模型去问人而不是猜。
func projectAgentsTemplate(project string) string {
	return fmt.Sprintf(`# %s

> 这个知识库由 qg 自动创建，还没有实际内容。
> 补充之前，遇到项目相关的问题（构建方式、架构、约定）直接问我，不要凭猜测回答。
> 随着需求推进，把确认过的架构、约定、坑逐步补进 .dsh/skills/ 和 .dsh/knowledge/。

## 构建 / 测试

<!-- 例如：make build / make test -->

## 必须遵守

<!-- 项目独有的硬约束，违反会出问题的那种 -->

## 知识索引

<!-- 往 .dsh/skills/ 里补技能后，在这里列出来，让模型知道什么时候该调哪个 -->
`, project)
}

// ensureBranch 创建分支知识目录，缺失时写入 journal 模板。
func ensureBranch(layout *Layout) error {
	dirs := []string{
		layout.Branch,
		filepath.Join(layout.Branch, "skills"),
		filepath.Join(layout.Branch, "knowledge"),
		filepath.Join(layout.Branch, "knowledge", "decisions"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("创建分支知识目录失败: %w", err)
		}
	}

	journal := filepath.Join(layout.Branch, "knowledge", "journal.md")
	if _, err := os.Stat(journal); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(journal, []byte(journalTemplate(layout.BranchName)), 0o644); err != nil {
			return fmt.Errorf("写入分支日志模板失败: %w", err)
		}
	}

	return nil
}

// ensureAgentSkillLinks 把 .dsh/skills 暴露给 Codex 与 Qoder 的项目技能发现。
func ensureAgentSkillLinks(layout *Layout) error {
	src := filepath.Join(layout.DshPath(), "skills")
	for _, dst := range []string{layout.CodexSkillsPath(), layout.QoderSkillsPath()} {
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("创建 agent 技能目录 %s 失败: %w", filepath.Dir(dst), err)
		}
		if err := linkIfAbsentOrDangling(src, dst, layout); err != nil {
			return fmt.Errorf("创建 agent 技能入口 %s 失败: %w", dst, err)
		}
	}
	return nil
}

// synthesize 重建工作树里的 .dsh 合成视图。
// 整体重建而不是增量更新，避免残留的旧链接指向已删除的基座技能。
func synthesize(layout *Layout) error {
	dsh := layout.DshPath()
	if err := os.RemoveAll(dsh); err != nil {
		return fmt.Errorf("清理旧的合成视图失败: %w", err)
	}
	if err := os.MkdirAll(dsh, 0o755); err != nil {
		return fmt.Errorf("创建合成视图目录失败: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dsh, managedMarker), []byte("managed by qg\n"), 0o644); err != nil {
		return fmt.Errorf("写入合成视图标记失败: %w", err)
	}

	skills := filepath.Join(dsh, "skills")
	if err := os.MkdirAll(skills, 0o755); err != nil {
		return fmt.Errorf("创建技能目录失败: %w", err)
	}

	if err := linkSkills(filepath.Join(layout.Base, "skills"), skills); err != nil {
		return err
	}
	// 分支技能后链接，同名的覆盖基座。
	if err := linkSkills(filepath.Join(layout.Branch, "skills"), skills); err != nil {
		return err
	}

	if err := linkPath(filepath.Join(layout.Branch, "knowledge"), filepath.Join(dsh, "knowledge"), layout); err != nil {
		return err
	}

	return linkPath(layout.Base, filepath.Join(dsh, "base"), layout)
}

// checkSynthesisTarget 在覆盖 .dsh 前确认目标缺失、链接到项目基座，
// 或是 qg 可识别的合成视图。用户自己的目录和外部链接必须保留。
func checkSynthesisTarget(dsh, base string) error {
	info, err := os.Lstat(dsh)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("检查 .dsh 失败: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(dsh)
		if err != nil {
			return fmt.Errorf("读取 .dsh 链接失败: %w", err)
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(dsh), target)
		}
		if filepath.Clean(target) == filepath.Clean(base) {
			return nil
		}
		return fmt.Errorf("%s 链接到 %s，不是 qg 管理的项目基座，未覆盖", dsh, target)
	}
	if isManagedSynthesis(dsh) {
		return nil
	}
	return fmt.Errorf("%s 已存在且不是 qg 创建的合成视图，未覆盖；请先备份或移走该目录后重试", dsh)
}

// isManagedSynthesis 识别新旧版本的 qg 合成视图。
func isManagedSynthesis(dsh string) bool {
	if isFile(filepath.Join(dsh, managedMarker)) {
		return true
	}
	return isSymlink(filepath.Join(dsh, "knowledge")) &&
		isSymlink(filepath.Join(dsh, "base")) &&
		isDir(filepath.Join(dsh, "skills"))
}

// linkSkills 把 srcDir 下的每个技能目录链接进 dstDir，同名先移除。
func linkSkills(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取技能目录失败: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			continue
		}

		dst := filepath.Join(dstDir, entry.Name())
		if err := os.Remove(dst); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("替换技能链接失败: %w", err)
		}
		if err := os.Symlink(filepath.Join(srcDir, entry.Name()), dst); err != nil {
			return fmt.Errorf("创建技能链接失败: %w", err)
		}
	}

	return nil
}

// linkPath 在 dst 建一条指向 src 的符号链接。
// dst 已存在且不是符号链接时跳过并记警告，绝不覆盖真实文件。
func linkPath(src, dst string, layout *Layout) error {
	if info, err := os.Lstat(dst); err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			layout.Warnings = append(layout.Warnings, fmt.Sprintf("%s 已存在且不是符号链接，未覆盖", dst))
			return nil
		}
		if err := os.Remove(dst); err != nil {
			return fmt.Errorf("移除旧链接失败: %w", err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	if err := os.Symlink(src, dst); err != nil {
		return fmt.Errorf("创建符号链接失败: %w", err)
	}

	return nil
}

// ensureExclude 把挂载物写进仓库本地的 info/exclude。
// 只影响本仓库的全部工作树，不产生任何版本库变更。
func ensureExclude(layout Layout) error {
	common, err := git.CommonDir(layout.Worktree)
	if err != nil {
		return err
	}

	path := filepath.Join(common, "info", "exclude")
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("读取 git exclude 失败: %w", err)
	}

	content := string(existing)
	missing := []string{}
	for _, entry := range excludeEntries {
		if !containsLine(content, entry) {
			missing = append(missing, entry)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += strings.Join(missing, "\n") + "\n"

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建 git info 目录失败: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("写入 git exclude 失败: %w", err)
	}

	return nil
}

func containsLine(content, target string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == target {
			return true
		}
	}
	return false
}
