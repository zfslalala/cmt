//go:build !windows

package dsh

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestOpenWorkspaceMissingCommand(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	err := OpenWorkspace(t.TempDir())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("OpenWorkspace() error = %v, want ErrNotFound", err)
	}
}

func TestOpenWorkspaceStartsWebInDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("使用 shell 脚本的假 dsh，仅覆盖 Unix")
	}

	binDir := t.TempDir()
	script := "#!/bin/sh\n{ pwd -P; printf '%s\\n' \"$@\"; } > \"$FAKE_DSH_OUT\"\n"
	if err := os.WriteFile(filepath.Join(binDir, "dsh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_DSH_OUT", out)

	dir := t.TempDir()
	if err := OpenWorkspace(dir); err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}

	var got string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(out)
		if err == nil {
			got = string(data)
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got == "" {
		t.Fatal("假 dsh 未被启动")
	}

	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{realDir, "web", "--port", "0"}, "\n") + "\n"
	if got != want {
		t.Errorf("dsh 调用 = %q, want %q", got, want)
	}
}
