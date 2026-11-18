//go:build !windows

package qoder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeCLI(t *testing.T, script string) (cli, argsFile string) {
	t.Helper()
	dir := t.TempDir()
	cli = filepath.Join(dir, "qoder")
	if err := os.WriteFile(cli, []byte(script), 0o755); err != nil {
		t.Fatalf("写入假 CLI 失败: %v", err)
	}
	return cli, filepath.Join(dir, "args")
}

func readArgs(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取参数失败: %v", err)
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func TestOpenWorkspaceUsesCLIWithNewWindow(t *testing.T) {
	worktree := t.TempDir()
	cli, argsFile := fakeCLI(t, "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$QODER_TEST_ARGS\"\n")
	t.Setenv("QG_QODER_CLI", cli)
	t.Setenv("QODER_TEST_ARGS", argsFile)

	via, err := OpenWorkspace(worktree)
	if err != nil {
		t.Fatalf("OpenWorkspace 失败: %v", err)
	}
	if via != "Qoder CLI" {
		t.Fatalf("打开方式 = %q，期望 Qoder CLI", via)
	}

	got := readArgs(t, argsFile)
	want := []string{"--new-window", worktree}
	if len(got) != len(want) {
		t.Fatalf("参数 = %v，期望 %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("参数 = %v，期望 %v", got, want)
		}
	}
}

func TestOpenWorkspaceRetriesWithoutNewWindow(t *testing.T) {
	worktree := t.TempDir()
	cli, argsFile := fakeCLI(t, `#!/bin/sh
if [ "$1" = "--new-window" ]; then
  echo "unknown option" >&2
  exit 2
fi
printf '%s\n' "$@" > "$QODER_TEST_ARGS"
`)
	t.Setenv("QG_QODER_CLI", cli)
	t.Setenv("QODER_TEST_ARGS", argsFile)

	if _, err := OpenWorkspace(worktree); err != nil {
		t.Fatalf("OpenWorkspace 失败: %v", err)
	}

	got := readArgs(t, argsFile)
	if len(got) != 1 || got[0] != worktree {
		t.Fatalf("回退参数 = %v，期望 [%s]", got, worktree)
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
