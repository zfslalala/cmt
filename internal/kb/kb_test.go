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
