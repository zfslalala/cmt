//go:build !windows

package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeCLI(t *testing.T, script string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	cli := filepath.Join(dir, "codex")
	if err := os.WriteFile(cli, []byte(script), 0o755); err != nil {
		t.Fatalf("写入假 Codex CLI 失败: %v", err)
	}
	return cli, filepath.Join(dir, "args")
}

func TestOpenWorkspaceUsesCodexApp(t *testing.T) {
	worktree := t.TempDir()
	cli, argsFile := fakeCLI(t, "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$CODEX_TEST_ARGS\"\n")
	t.Setenv("QG_CODEX_CLI", cli)
	t.Setenv("CODEX_TEST_ARGS", argsFile)

	via, err := OpenWorkspace(worktree)
	if err != nil {
		t.Fatalf("OpenWorkspace 失败: %v", err)
	}
	if via != "Codex Desktop" {
		t.Fatalf("打开方式 = %q，期望 Codex Desktop", via)
	}

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("读取参数失败: %v", err)
	}
	got := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{"app", worktree}
	if len(got) != len(want) {
		t.Fatalf("参数 = %v，期望 %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("参数 = %v，期望 %v", got, want)
		}
	}
}

func TestOpenWorkspaceRejectsFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	if _, err := OpenWorkspace(file); err == nil {
		t.Fatal("打开普通文件应失败")
	}
}

func TestOpenWorkspaceReportsOldCLI(t *testing.T) {
	worktree := t.TempDir()
	cli, _ := fakeCLI(t, "#!/bin/sh\necho 'error: unrecognized subcommand app' >&2\nexit 2\n")
	t.Setenv("QG_CODEX_CLI", cli)

	_, err := OpenWorkspace(worktree)
	if err == nil || !strings.Contains(err.Error(), "请更新 Codex") {
		t.Fatalf("旧版 CLI 应提示更新，实际: %v", err)
	}
}
