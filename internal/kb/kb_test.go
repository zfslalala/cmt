package kb

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setup 创建一个带一次提交的临时 git 仓库，返回其父目录与仓库根。
func setup(t *testing.T) (string, string) {
	t.Helper()

	parent := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(parent); err == nil {
		parent = resolved
	}

	repo := filepath.Join(parent, "demo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("创建仓库目录失败: %v", err)
	}

	runGit(t, repo, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("demo\n"), 0o644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-qm", "init")

	return parent, repo
}

// setupKB 在 parent 下创建一个带基座的知识库，并把 AI_KB_ROOT 指向它。
func setupKB(t *testing.T, parent, project string) (string, string) {
	t.Helper()

	root := filepath.Join(parent, defaultRootName)
	base := filepath.Join(root, "projects", project, baseDirName)

	if err := os.MkdirAll(filepath.Join(base, "skills", "arch"), 0o755); err != nil {
		t.Fatalf("创建基座技能目录失败: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, "knowledge", "decisions"), 0o755); err != nil {
		t.Fatalf("创建基座知识目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "AGENTS.md"), []byte("# demo\n"), 0o644); err != nil {
		t.Fatalf("写入基座 AGENTS.md 失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "skills", "arch", "SKILL.md"), []byte("---\nname: arch\n---\n"), 0o644); err != nil {
		t.Fatalf("写入基座技能失败: %v", err)
	}

	t.Setenv("AI_KB_ROOT", root)

	return root, base
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v 失败: %v\n%s", args, err, output)
	}

	return string(output)
}

func assertLinkTarget(t *testing.T, path, want string) {
	t.Helper()

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("期望存在符号链接 %s: %v", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s 不是符号链接", path)
	}

	got, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("读取链接失败: %v", err)
	}
	if got != want {
		t.Fatalf("链接目标不对\n期望: %s\n实际: %s", want, got)
	}
}

func TestAttachMainWorktreeUsesBase(t *testing.T) {
	parent, repo := setup(t)
	_, base := setupKB(t, parent, "demo")

	layout, err := Attach(repo)
	if err != nil {
		t.Fatalf("挂载失败: %v", err)
	}
	if !layout.IsMain {
		t.Fatalf("应判定为主工作树")
	}

	assertLinkTarget(t, filepath.Join(repo, "AGENTS.md"), filepath.Join(base, "AGENTS.md"))
	assertLinkTarget(t, filepath.Join(repo, DshDir), base)
}

func TestAttachFeatureWorktreeSynthesizes(t *testing.T) {
	parent, repo := setup(t)
	root, base := setupKB(t, parent, "demo")

	worktree := filepath.Join(parent, "feat-x")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", worktree)

	layout, err := Attach(worktree)
	if err != nil {
		t.Fatalf("挂载失败: %v", err)
	}
	if layout.IsMain {
		t.Fatalf("特性分支不应判定为主工作树")
	}

	info, err := os.Lstat(filepath.Join(worktree, DshDir))
	if err != nil {
		t.Fatalf("缺少 .dsh: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("特性分支的 .dsh 应为真实目录")
	}
	if _, err := os.Stat(filepath.Join(worktree, DshDir, managedMarker)); err != nil {
		t.Fatalf("缺少合成视图标记: %v", err)
	}

	assertLinkTarget(t,
		filepath.Join(worktree, DshDir, "skills", "arch"),
		filepath.Join(base, "skills", "arch"))

	branch := filepath.Join(root, "projects", "demo", branchesDirName, "feat-x")
	assertLinkTarget(t,
		filepath.Join(worktree, DshDir, "knowledge"),
		filepath.Join(branch, "knowledge"))
	assertLinkTarget(t, filepath.Join(worktree, DshDir, "base"), base)

	if _, err := os.Stat(filepath.Join(branch, "knowledge", "journal.md")); err != nil {
		t.Fatalf("缺少分支日志模板: %v", err)
	}
}

func TestAttachFeatureWorktreePreservesForeignDsh(t *testing.T) {
	parent, repo := setup(t)
	setupKB(t, parent, "demo")

	worktree := filepath.Join(parent, "feat-x")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", worktree)

	userFile := filepath.Join(worktree, DshDir, "user.txt")
	if err := os.MkdirAll(filepath.Dir(userFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userFile, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Attach(worktree); err == nil || !strings.Contains(err.Error(), "未覆盖") {
		t.Fatalf("应拒绝覆盖用户 .dsh，实际: %v", err)
	}
	data, err := os.ReadFile(userFile)
	if err != nil {
		t.Fatalf("用户文件被删除: %v", err)
	}
	if string(data) != "keep me" {
		t.Fatalf("用户文件被修改: %q", data)
	}
}

func TestAttachFeatureWorktreePreservesForeignDshLink(t *testing.T) {
	parent, repo := setup(t)
	setupKB(t, parent, "demo")

	worktree := filepath.Join(parent, "feat-x")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", worktree)

	foreign := filepath.Join(parent, "foreign-dsh")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(foreign, filepath.Join(worktree, DshDir)); err != nil {
		t.Fatal(err)
	}

	if _, err := Attach(worktree); err == nil || !strings.Contains(err.Error(), "不是 qg 管理") {
		t.Fatalf("应拒绝覆盖外部 .dsh 链接，实际: %v", err)
	}
	assertLinkTarget(t, filepath.Join(worktree, DshDir), foreign)
}

func TestProjectKeyRejectsPathTraversal(t *testing.T) {
	_, repo := setup(t)
	keyFile := filepath.Join(repo, ".git", "info", projectKeyFile)
	if err := os.MkdirAll(filepath.Dir(keyFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, []byte("../escape"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := projectKey(repo); err == nil {
		t.Fatal("包含路径分隔符的项目键应被拒绝")
	}
}

func TestBranchSkillOverridesBase(t *testing.T) {
	parent, repo := setup(t)
	root, base := setupKB(t, parent, "demo")

	worktree := filepath.Join(parent, "feat-x")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", worktree)

	override := filepath.Join(root, "projects", "demo", branchesDirName, "feat-x", "skills", "arch")
	if err := os.MkdirAll(override, 0o755); err != nil {
		t.Fatalf("创建分支技能失败: %v", err)
	}

	if _, err := Attach(worktree); err != nil {
		t.Fatalf("挂载失败: %v", err)
	}

	assertLinkTarget(t, filepath.Join(worktree, DshDir, "skills", "arch"), override)

	if _, err := Attach(worktree); err != nil {
		t.Fatalf("重复挂载失败: %v", err)
	}
	assertLinkTarget(t, filepath.Join(worktree, DshDir, "skills", "arch"), override)

	if _, err := os.Stat(filepath.Join(base, "skills", "arch")); err != nil {
		t.Fatalf("基座技能不应被移除: %v", err)
	}
}

func TestAttachIsIdempotentAndExcludes(t *testing.T) {
	parent, repo := setup(t)
	setupKB(t, parent, "demo")

	for i := 0; i < 2; i++ {
		if _, err := Attach(repo); err != nil {
			t.Fatalf("第 %d 次挂载失败: %v", i+1, err)
		}
	}

	common, err := filepath.Abs(filepath.Join(repo, ".git"))
	if err != nil {
		t.Fatalf("解析 git 目录失败: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(common, "info", "exclude"))
	if err != nil {
		t.Fatalf("读取 exclude 失败: %v", err)
	}

	for _, entry := range excludeEntries {
		count := 0
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == entry {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("exclude 条目 %q 出现 %d 次，期望 1 次", entry, count)
		}
	}
}

func TestStatusReportsMountState(t *testing.T) {
	parent, repo := setup(t)
	setupKB(t, parent, "demo")

	items, project, err := Status(repo)
	if err != nil {
		t.Fatalf("查询状态失败: %v", err)
	}
	if project != "demo" {
		t.Fatalf("项目键不对: %s", project)
	}
	if len(items) != 1 || items[0].Mounted {
		t.Fatalf("挂载前应报告未挂载: %+v", items)
	}

	if _, err := Attach(repo); err != nil {
		t.Fatalf("挂载失败: %v", err)
	}

	items, _, err = Status(repo)
	if err != nil {
		t.Fatalf("查询状态失败: %v", err)
	}
	if len(items) != 1 || !items[0].Mounted {
		t.Fatalf("挂载后应报告已挂载: %+v", items)
	}
}

func TestWriteLocalNoteCreatesAndKeepsExisting(t *testing.T) {
	parent, repo := setup(t)
	setupKB(t, parent, "demo")

	worktree := filepath.Join(parent, "feat-x")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", worktree)

	layout, err := Attach(worktree)
	if err != nil {
		t.Fatalf("挂载失败: %v", err)
	}

	note := filepath.Join(worktree, LocalNoteFile)
	written, err := WriteLocalNote(layout, NoteContent{Branch: "feat/x", Note: "加一个 kb 命令"})
	if err != nil {
		t.Fatalf("写说明文件失败: %v", err)
	}
	if !written {
		t.Fatalf("首次应写入 %s", LocalNoteFile)
	}

	data, err := os.ReadFile(note)
	if err != nil {
		t.Fatalf("读取说明文件失败: %v", err)
	}
	if !strings.Contains(string(data), "加一个 kb 命令") {
		t.Fatalf("说明文件缺少需求说明:\n%s", data)
	}
	if !strings.Contains(string(data), "feat/x") {
		t.Fatalf("说明文件缺少分支名:\n%s", data)
	}

	if err := os.WriteFile(note, []byte("keep me\n"), 0o644); err != nil {
		t.Fatalf("改写说明文件失败: %v", err)
	}
	written, err = WriteLocalNote(layout, NoteContent{Branch: "feat/x", Note: "不该生效"})
	if err != nil {
		t.Fatalf("重复写说明文件失败: %v", err)
	}
	if written {
		t.Fatalf("已存在时不应覆盖")
	}
	data, _ = os.ReadFile(note)
	if string(data) != "keep me\n" {
		t.Fatalf("说明文件被覆盖: %s", data)
	}
}

func TestMainWorktreeHasNoLocalNote(t *testing.T) {
	parent, repo := setup(t)
	setupKB(t, parent, "demo")

	layout, err := Attach(repo)
	if err != nil {
		t.Fatalf("挂载失败: %v", err)
	}

	written, err := WriteLocalNote(layout, NoteContent{Branch: "main"})
	if err != nil {
		t.Fatalf("写说明文件失败: %v", err)
	}
	if written {
		t.Fatalf("主工作树不应生成 %s", LocalNoteFile)
	}
}

func TestCleanupBranchKnowledge(t *testing.T) {
	parent, repo := setup(t)
	setupKB(t, parent, "demo")

	worktree := filepath.Join(parent, "feat-x")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", worktree)

	layout, err := Attach(worktree)
	if err != nil {
		t.Fatalf("挂载失败: %v", err)
	}

	removed, err := CleanupBranchKnowledge(layout)
	if err != nil {
		t.Fatalf("清理空分支知识失败: %v", err)
	}
	if !removed {
		t.Fatalf("未写入过的分支知识应被清理")
	}
	if _, err := os.Stat(layout.Branch); !os.IsNotExist(err) {
		t.Fatalf("分支知识目录应已删除: %v", err)
	}

	// 有实际内容时必须保留
	layout, err = Attach(worktree)
	if err != nil {
		t.Fatalf("重新挂载失败: %v", err)
	}
	journal := filepath.Join(layout.Branch, "knowledge", "journal.md")
	if err := os.WriteFile(journal, []byte("## 2026-11-15 · 一条经验\n"), 0o644); err != nil {
		t.Fatalf("写入经验失败: %v", err)
	}

	removed, err = CleanupBranchKnowledge(layout)
	if err != nil {
		t.Fatalf("清理分支知识失败: %v", err)
	}
	if removed {
		t.Fatalf("有实际内容时不应删除")
	}
	if _, err := os.Stat(journal); err != nil {
		t.Fatalf("日志应被保留: %v", err)
	}
}

func TestAttachCreatesMissingProject(t *testing.T) {
	parent, repo := setup(t)
	// 不预先建知识库，模拟首次使用或知识库被删
	root := filepath.Join(parent, defaultRootName)
	t.Setenv("AI_KB_ROOT", root)

	layout, err := Attach(repo)
	if err != nil {
		t.Fatalf("缺少基座时应自动创建，实际失败: %v", err)
	}
	if !layout.Bootstrapped {
		t.Fatalf("应标记为已自动创建")
	}

	base := filepath.Join(root, "projects", "demo", baseDirName)
	for _, path := range []string{
		filepath.Join(base, "AGENTS.md"),
		filepath.Join(base, "skills"),
		filepath.Join(base, "knowledge", "journal.md"),
		filepath.Join(base, "knowledge", "glossary.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("骨架缺少 %s: %v", path, err)
		}
	}

	data, err := os.ReadFile(filepath.Join(base, "AGENTS.md"))
	if err != nil {
		t.Fatalf("读取 AGENTS.md 失败: %v", err)
	}
	if !strings.Contains(string(data), "demo") {
		t.Fatalf("骨架 AGENTS.md 应含项目名:\\n%s", data)
	}
	assertLinkTarget(t, filepath.Join(repo, "AGENTS.md"), filepath.Join(base, "AGENTS.md"))

	// 已存在的内容不能被第二次挂载覆盖
	custom := filepath.Join(base, "AGENTS.md")
	if err := os.WriteFile(custom, []byte("# 手写内容\n"), 0o644); err != nil {
		t.Fatalf("改写失败: %v", err)
	}
	layout, err = Attach(repo)
	if err != nil {
		t.Fatalf("重复挂载失败: %v", err)
	}
	if layout.Bootstrapped {
		t.Fatalf("第二次不应再创建骨架")
	}
	data, _ = os.ReadFile(custom)
	if string(data) != "# 手写内容\n" {
		t.Fatalf("已存在的 AGENTS.md 被覆盖: %s", data)
	}
}

func TestAttachRebuildsWholeKnowledgeBase(t *testing.T) {
	parent, repo := setup(t)
	root := filepath.Join(parent, defaultRootName)
	home := filepath.Join(parent, "dsh-home")
	t.Setenv("AI_KB_ROOT", root)
	t.Setenv("DSH_HOME", home)

	layout, err := Attach(repo)
	if err != nil {
		t.Fatalf("知识库整体缺失时应能重建: %v", err)
	}
	if !layout.GlobalCreated {
		t.Fatalf("应标记为创建了全局层")
	}
	if !layout.Bootstrapped {
		t.Fatalf("应标记为创建了项目骨架")
	}

	for _, path := range []string{
		filepath.Join(root, "AGENTS.md"),
		filepath.Join(root, "skills"),
		filepath.Join(root, "projects", "demo", baseDirName, "AGENTS.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("缺少 %s: %v", path, err)
		}
	}

	assertLinkTarget(t, filepath.Join(home, "AGENTS.md"), filepath.Join(root, "AGENTS.md"))
	assertLinkTarget(t, filepath.Join(home, "skills"), filepath.Join(root, "skills"))
}

func TestAttachRepairsDanglingGlobalLinks(t *testing.T) {
	parent, repo := setup(t)
	root := filepath.Join(parent, defaultRootName)
	home := filepath.Join(parent, "dsh-home")
	t.Setenv("AI_KB_ROOT", root)
	t.Setenv("DSH_HOME", home)

	if _, err := Attach(repo); err != nil {
		t.Fatalf("首次挂载失败: %v", err)
	}

	// 模拟整个知识库被删：全局链接变悬空
	if err := os.RemoveAll(root); err != nil {
		t.Fatalf("删除知识库失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("此时全局链接应已悬空")
	}

	if _, err := Attach(repo); err != nil {
		t.Fatalf("知识库被删后应能重建: %v", err)
	}

	assertLinkTarget(t, filepath.Join(home, "AGENTS.md"), filepath.Join(root, "AGENTS.md"))
	if _, err := os.Stat(filepath.Join(home, "AGENTS.md")); err != nil {
		t.Fatalf("重建后全局链接应可解析: %v", err)
	}
}

func TestAttachDoesNotStealForeignGlobalFile(t *testing.T) {
	parent, repo := setup(t)
	root := filepath.Join(parent, defaultRootName)
	home := filepath.Join(parent, "dsh-home")
	t.Setenv("AI_KB_ROOT", root)
	t.Setenv("DSH_HOME", home)

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("创建 dsh home 失败: %v", err)
	}
	mine := filepath.Join(home, "AGENTS.md")
	if err := os.WriteFile(mine, []byte("# 我自己写的\n"), 0o644); err != nil {
		t.Fatalf("写文件失败: %v", err)
	}

	layout, err := Attach(repo)
	if err != nil {
		t.Fatalf("挂载失败: %v", err)
	}

	data, _ := os.ReadFile(mine)
	if string(data) != "# 我自己写的\n" {
		t.Fatalf("不应覆盖用户自己的 ~/.dsh/AGENTS.md")
	}
	if len(layout.Warnings) == 0 {
		t.Fatalf("跳过的全局文件应产生警告")
	}
}
