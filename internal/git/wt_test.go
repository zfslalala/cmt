package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeBranchName(t *testing.T) {
	cases := map[string]string{
		"feature/login":  "feature-login",
		"feature login":  "feature-login",
		"feature\\login": "feature-login",
		"fix:bug":        "fix-bug",
		"a<b>c|d?e*f\"g": "a-b-c-d-e-f-g",
		"plain":          "plain",
		"release/v1.0":   "release-v1.0",
	}
	for input, want := range cases {
		if got := SanitizeBranchName(input); got != want {
			t.Errorf("SanitizeBranchName(%q) = %q, want %q", input, got, want)
		}
	}
}

// 从 release 切出的新分支不应继承 origin/release 作为 upstream，
// 否则 git push 会尝试推到起点分支。
func TestCreateWorktreeDoesNotTrackStartPoint(t *testing.T) {
	repo := rmTestRepo(t)
	runIn := func(wd string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = wd
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return string(out)
	}

	runIn(repo, "checkout", "-q", "-b", "release")
	writeFile(t, repo+"/release.txt", "release")
	runIn(repo, "add", ".")
	runIn(repo, "commit", "-q", "-m", "release")
	runIn(repo, "push", "-q", "-u", "origin", "release")
	runIn(repo, "checkout", "-q", "main")

	chdir(t, repo)
	path, start, err := CreateWorktree("feat/from-release", "release")
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	if start != "origin/release" {
		t.Fatalf("resolvedStart = %q, want origin/release", start)
	}

	upstream := func() (string, bool) {
		cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
		cmd.Dir = path
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", false
		}
		return strings.TrimSpace(string(out)), true
	}

	if got, ok := upstream(); ok {
		t.Fatalf("新分支不应有 upstream，实际为 %q", got)
	}

	// 首次推送应绑定到 origin/<新分支>，而不是 origin/release
	chdir(t, path)
	if err := Push(); err != nil {
		t.Fatalf("Push: %v", err)
	}
	mergeRef := strings.TrimSpace(runIn(path, "config", "--get", "branch.feat/from-release.merge"))
	if mergeRef != "refs/heads/feat/from-release" {
		t.Fatalf("upstream merge = %q, want refs/heads/feat/from-release", mergeRef)
	}
}

func TestCreateWorktreeFromSubdirectoryUsesRepoParent(t *testing.T) {
	repo := rmTestRepo(t)
	subdir := filepath.Join(repo, "src", "nested")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("创建子目录: %v", err)
	}
	chdir(t, subdir)

	path, _, err := CreateWorktree("feat/from-subdir", "")
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	want, err := filepath.EvalSymlinks(filepath.Dir(repo))
	if err != nil {
		t.Fatal(err)
	}
	want = filepath.Join(want, "feat-from-subdir")
	if path != want {
		t.Fatalf("工作树路径 = %q，期望仓库同级 %q", path, want)
	}
}
