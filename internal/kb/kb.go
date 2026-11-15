// Package kb 把一个仓库之外的知识库挂载到 git 工作树。
//
// 知识库磁盘布局：
//
//	<知识库>/projects/<项目>/base/              基座：已确认的知识
//	<知识库>/projects/<项目>/branches/<分支>/   分支自己的知识，可写、与基座隔离
//
// 主工作树直接指向基座，基座就是在主工作树上编辑的。特性分支挂载时在工作树里
// 生成 .dsh 合成视图：技能按名字由分支覆盖基座，knowledge 指向分支目录，
// base 指向基座作为只读参考。所有挂载物都写进 git 的 info/exclude，因此对
// git 不可见，也不会被提交。
package kb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zfslalala/cmt/internal/git"
)

const (
	// DshDir 是工作树里承载技能与知识的目录名。
	DshDir = ".dsh"
	// LocalNoteFile 是本工作树专属的说明文件，DSH 会把它叠加在 AGENTS.md 之后加载。
	LocalNoteFile = "AGENTS.local.md"
	// defaultRootName 是知识库的默认目录名，与仓库同级。
	defaultRootName = "_ai-kb"
	// baseDirName 是基座目录名。
	baseDirName = "base"
	// branchesDirName 是分支知识目录名。
	branchesDirName = "branches"
	// projectKeyFile 是仓库本地覆盖项目键的文件，位于共享 git 目录的 info 下。
	projectKeyFile = "kb-project"
)

// excludeEntries 会被写进 git 的 info/exclude，让挂载物对 git 不可见。
var excludeEntries = []string{
	"/AGENTS.md",
	"/AGENTS.local.md",
	"/CLAUDE.md",
	"/GEMINI.md",
	"/.dsh",
	"/.trae",
}

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
	// Warnings 收集不致命、但需要人知道的情况，例如已存在的真实文件未被覆盖。
	Warnings []string
}

// DshPath 返回工作树里 .dsh 的路径。
func (l Layout) DshPath() string {
	return filepath.Join(l.Worktree, DshDir)
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

	paths, err := git.WorktreePaths(worktree)
	if err != nil {
		return layout, err
	}
	main, err := filepath.Abs(paths[0])
	if err != nil {
		return layout, fmt.Errorf("解析主工作树路径失败: %w", err)
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

// projectKey 决定项目键：优先读仓库本地的 info/kb-project，
// 否则退化为主工作树的目录名。
func projectKey(mainWorktree string) (string, error) {
	if common, err := git.CommonDir(mainWorktree); err == nil {
		data, readErr := os.ReadFile(filepath.Join(common, "info", projectKeyFile))
		if readErr == nil {
			if key := strings.TrimSpace(string(data)); key != "" {
				return key, nil
			}
		}
	}

	key := filepath.Base(filepath.Clean(mainWorktree))
	if key == "" || key == "." || key == string(filepath.Separator) {
		return "", fmt.Errorf("无法从主工作树路径推导项目名: %s", mainWorktree)
	}

	return key, nil
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
// 已存在的真实文件不会被覆盖，只记进 Warnings。
func Attach(dir string) (Layout, error) {
	layout, err := Resolve(dir)
	if err != nil {
		return layout, err
	}

	if !isDir(layout.Base) {
		return layout, fmt.Errorf("知识库缺少基座目录: %s", layout.Base)
	}

	if !layout.IsMain {
		if err := ensureBranch(&layout); err != nil {
			return layout, err
		}
	}

	if err := linkPath(filepath.Join(layout.Base, "AGENTS.md"), filepath.Join(layout.Worktree, "AGENTS.md"), &layout); err != nil {
		return layout, err
	}

	if layout.IsMain {
		// 主工作树直接使用基座：基座就是在主工作树上编辑的。
		if err := linkPath(layout.Base, layout.DshPath(), &layout); err != nil {
			return layout, err
		}
	} else if err := synthesize(&layout); err != nil {
		return layout, err
	}

	if err := ensureExclude(layout); err != nil {
		return layout, err
	}

	return layout, nil
}

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
}

// Status 列出仓库全部工作树的挂载状态，不写任何文件。
func Status(dir string) ([]WorktreeStatus, string, error) {
	layout, err := Resolve(dir)
	if err != nil {
		return nil, "", err
	}

	paths, err := git.WorktreePaths(layout.Worktree)
	if err != nil {
		return nil, "", err
	}

	items := make([]WorktreeStatus, 0, len(paths))
	for _, path := range paths {
		item := WorktreeStatus{
			Path:   path,
			IsMain: filepath.Clean(path) == layout.MainWorktree,
		}

		agents := filepath.Join(path, "AGENTS.md")
		dsh := filepath.Join(path, DshDir)
		agentsOK := isSymlink(agents) && pathExists(agents)
		dshOK := pathExists(dsh)
		item.Mounted = agentsOK && dshOK

		switch {
		case !agentsOK && !dshOK:
			item.Detail = "未挂载"
		case !agentsOK:
			item.Detail = "缺少 AGENTS.md 链接"
		case !dshOK:
			item.Detail = "缺少 .dsh"
		case isSymlink(dsh):
			item.Detail = "直接使用基座"
		default:
			item.Detail = "合成视图"
		}

		items = append(items, item)
	}

	return items, layout.Project, nil
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

// journalTemplate 是分支日志的初始内容，CleanupBranchKnowledge 也用同一份文本
// 判断分支知识是否从未被写入过。
func journalTemplate(branch string) string {
	return fmt.Sprintf("# 分支知识日志\n\n分支：%s\n\n<!-- 按时间倒序，最新在上。合入基座后可清空。 -->\n", branch)
}

// NoteContent 描述工作树专属说明文件的内容。
type NoteContent struct {
	// Branch 是当前分支名。
	Branch string
	// Start 是分支起点，为空表示基于当时 HEAD。
	Start string
	// Note 是用户填写的需求说明，可为空。
	Note string
	// Created 是创建时间。
	Created time.Time
}

// WriteLocalNote 在特性分支的工作树里写 AGENTS.local.md。
// 主工作树不写；文件已存在时保留原文并返回 false，绝不覆盖。
func WriteLocalNote(layout Layout, content NoteContent) (bool, error) {
	if layout.IsMain {
		return false, nil
	}

	path := filepath.Join(layout.Worktree, LocalNoteFile)
	if _, err := os.Lstat(path); err == nil {
		return false, nil
	}

	start := content.Start
	if start == "" {
		start = "当前 HEAD"
	}
	note := strings.TrimSpace(content.Note)
	if note == "" {
		note = "<!-- 一句话写清楚这个分支要做什么，写完删掉这行注释。 -->"
	}

	text := fmt.Sprintf(`# 本工作树

- 分支：%s
- 起点：%s
- 创建：%s

## 本需求

%s

## 本分支约束

<!-- 例如：只改某个目录，不要动别的模块。没有就删掉这一节。 -->

## 知识库

项目知识在 .dsh/ 下，按根目录 AGENTS.md 的索引按需读取。
本分支的写入只会落进分支知识目录，不会污染基座。
`, content.Branch, start, content.Created.Format("2006-01-02"), note)

	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return false, fmt.Errorf("写入 %s 失败: %w", LocalNoteFile, err)
	}

	return true, nil
}

// CleanupBranchKnowledge 删除一个已结束分支的知识目录。
// 只在目录从未被写入过时删除；有任何实际内容时保留并返回 false，避免丢经验。
func CleanupBranchKnowledge(layout Layout) (bool, error) {
	if layout.IsMain || layout.Branch == "" {
		return false, nil
	}

	if _, err := os.Stat(layout.Branch); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("检查分支知识目录失败: %w", err)
	}

	pristine, err := isPristineBranch(layout)
	if err != nil {
		return false, err
	}
	if !pristine {
		return false, nil
	}

	if err := os.RemoveAll(layout.Branch); err != nil {
		return false, fmt.Errorf("删除分支知识目录失败: %w", err)
	}

	return true, nil
}

// isPristineBranch 判断分支知识目录是否仍是刚创建时的空壳：
// 没有技能、没有决策，只有一份未被改动过的 journal 模板。
func isPristineBranch(layout Layout) (bool, error) {
	skills := filepath.Join(layout.Branch, "skills")
	if entries, err := os.ReadDir(skills); err == nil && len(entries) > 0 {
		return false, nil
	}

	decisions := filepath.Join(layout.Branch, "knowledge", "decisions")
	if entries, err := os.ReadDir(decisions); err == nil && len(entries) > 0 {
		return false, nil
	}

	knowledge := filepath.Join(layout.Branch, "knowledge")
	entries, err := os.ReadDir(knowledge)
	if err != nil {
		return false, fmt.Errorf("读取分支知识目录失败: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() != "journal.md" && entry.Name() != "decisions" {
			return false, nil
		}
	}

	journal := filepath.Join(knowledge, "journal.md")
	data, err := os.ReadFile(journal)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("读取分支日志失败: %w", err)
	}

	return string(data) == journalTemplate(layout.BranchName), nil
}

// synthesize 重建工作树里的 .dsh 合成视图。
// 整体重建而不是增量更新，避免残留的旧链接指向已删除的基座技能。
func synthesize(layout *Layout) error {
	dsh := layout.DshPath()
	if err := os.RemoveAll(dsh); err != nil {
		return fmt.Errorf("清理旧的合成视图失败: %w", err)
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

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
