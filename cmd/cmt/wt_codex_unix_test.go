//go:build !windows

package cmt

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWtOpensCodexWorkspace(t *testing.T) {
	parent, repo := setupRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", ".")
	runGit(t, repo, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-qm", "init")

	cliDir := t.TempDir()
	cli := filepath.Join(cliDir, "codex")
	if err := os.WriteFile(cli, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$CODEX_TEST_ARGS\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	argsFile := filepath.Join(cliDir, "args")
	t.Setenv("QG_CODEX_CLI", cli)
	t.Setenv("CODEX_TEST_ARGS", argsFile)
	t.Setenv("DSH_HOME", filepath.Join(cliDir, "dsh-home"))
	chdir(t, repo)

	if err := runWt("feat/codex", wtOptions{openCodex: true, note: "验证多 agent 知识库"}); err != nil {
		t.Fatalf("runWt: %v", err)
	}

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("读取 Codex 参数失败: %v", err)
	}
	got := strings.Split(strings.TrimSpace(string(data)), "\n")
	wantPath := filepath.Join(parent, "feat-codex")
	if len(got) != 2 || got[0] != "app" || got[1] != wantPath {
		t.Fatalf("Codex 参数 = %v，期望 [app %s]", got, wantPath)
	}
	agents, err := os.ReadFile(filepath.Join(wantPath, "AGENTS.md"))
	if err != nil {
		t.Fatalf("读取工作树 AGENTS.md 失败: %v", err)
	}
	if !strings.Contains(string(agents), "验证多 agent 知识库") || !strings.Contains(string(agents), "qg-managed: agent-instructions") {
		t.Fatalf("工作树 AGENTS.md 未合并分支说明:\n%s", agents)
	}
}

func TestCreatedWorktreeIsReadableByCodex(t *testing.T) {
	codexBin, err := exec.LookPath("codex")
	if err != nil {
		t.Skip("未安装 codex，跳过真实 prompt 集成测试")
	}

	parent, repo := setupRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", ".")
	runGit(t, repo, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-qm", "init")

	kbRoot := os.Getenv("AI_KB_ROOT")
	base := filepath.Join(kbRoot, "projects", "demo", "base")
	for _, dir := range []string{
		filepath.Join(base, "skills", "arch"),
		filepath.Join(base, "knowledge", "decisions"),
		filepath.Join(kbRoot, "skills", "knowledge-capture"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(base, "AGENTS.md"), []byte("# Shared Base\n\nCODEX_KB_BASE_MARKER_4A71\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "skills", "arch", "SKILL.md"), []byte("---\nname: arch\ndescription: UNIQUE_CODEX_KB_SKILL_8E32\n---\n\n# Arch\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cliDir := t.TempDir()
	fakeCodex := filepath.Join(cliDir, "codex")
	if err := os.WriteFile(fakeCodex, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("QG_CODEX_CLI", fakeCodex)
	chdir(t, repo)

	if err := runWt("feat/kb", wtOptions{openCodex: true, note: "CODEX_BRANCH_NOTE_MARKER_2F19"}); err != nil {
		t.Fatalf("runWt: %v", err)
	}

	worktree := filepath.Join(parent, "feat-kb")
	cmd := exec.Command(codexBin, "-C", worktree, "debug", "prompt-input", "probe")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("codex prompt-input 失败: %v\n%s", err, output)
	}
	text := string(output)
	for _, want := range []string{"CODEX_KB_BASE_MARKER_4A71", "CODEX_BRANCH_NOTE_MARKER_2F19", "UNIQUE_CODEX_KB_SKILL_8E32"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Codex prompt 未加载知识库内容 %q", want)
		}
	}
}
