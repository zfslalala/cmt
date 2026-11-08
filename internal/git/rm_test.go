package git

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// rmTestRepo 搭建：主仓库（在 main 分支）+ 本地 bare 远程 origin
func rmTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runIn := func(wd string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = wd
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}

	// 本地 bare 远程
	origin := dir + "/origin.git"
	if err := os.Mkdir(origin, 0o755); err != nil {
		t.Fatal(err)
	}
	runIn(origin, "init", "--bare", "-q")

	// 主仓库
	repo := dir + "/repo"
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	runIn(repo, "init", "-q")
	runIn(repo, "config", "user.email", "t@t")
	runIn(repo, "config", "user.name", "t")
	writeFile(t, repo+"/a.txt", "v1")
	runIn(repo, "add", ".")
	runIn(repo, "commit", "-q", "-m", "init")
	runIn(repo, "branch", "-M", "main")
	runIn(repo, "remote", "add", "origin", origin)
	runIn(repo, "push", "-q", "-u", "origin", "main")

	return repo
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

// 当前分支检出于主仓库：先切 main 再删分支
func TestDeleteBranchCurrentInMainRepo(t *testing.T) {
	repo := rmTestRepo(t)
	runIn := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	runIn("checkout", "-q", "-b", "feat/x")
	chdir(t, repo)

	res, err := DeleteBranch("feat/x", false, false)
	if err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}
	if res.SwitchedTo != "main" {
		t.Errorf("SwitchedTo = %q, want main", res.SwitchedTo)
	}
	if branchExists("feat/x") {
		t.Error("分支未被删除")
	}
}

// 分支检出于链接工作树：工作树与分支一并删除
func TestDeleteBranchWithWorktree(t *testing.T) {
	repo := rmTestRepo(t)
	runIn := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	runIn("worktree", "add", "-b", "feat/wt", repo+"/../feat-wt")

	chdir(t, repo)
	res, err := DeleteBranch("feat/wt", false, false)
	if err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}
	if res.WorktreePath == "" {
		t.Error("工作树未被记录删除")
	}
	if _, err := os.Stat(repo + "/../feat-wt"); !os.IsNotExist(err) {
		t.Error("工作树目录仍存在")
	}
	if branchExists("feat/wt") {
		t.Error("分支未被删除")
	}
}

// 未合并分支默认拒绝，--force 放行
func TestDeleteBranchUnmerged(t *testing.T) {
	repo := rmTestRepo(t)
	runIn := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	runIn("checkout", "-q", "-b", "feat/late")
	if err := os.WriteFile(repo+"/late.txt", []byte("new commit"), 0o644); err != nil {
		t.Fatal(err)
	}
	runIn("add", ".")
	runIn("commit", "-q", "-m", "late work")
	runIn("checkout", "-q", "main")

	chdir(t, repo)
	if _, err := DeleteBranch("feat/late", false, false); err == nil {
		t.Error("未合并分支应被拒绝")
	}
	if !branchExists("feat/late") {
		t.Error("未合并分支不应被删除")
	}

	if _, err := DeleteBranch("feat/late", false, true); err != nil {
		t.Fatalf("force 删除失败: %v", err)
	}
	if branchExists("feat/late") {
		t.Error("force 后分支仍未删除")
	}
}

// 主仓库检出的分支拒绝删除
func TestDeleteBranchRefusesCheckedOut(t *testing.T) {
	repo := rmTestRepo(t)
	chdir(t, repo)
	if _, err := DeleteBranch("main", false, true); err == nil {
		t.Error("删除主仓库检出分支应被拒绝")
	}
}

// -o 删除远程分支
func TestDeleteBranchRemote(t *testing.T) {
	repo := rmTestRepo(t)
	runIn := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	runIn("checkout", "-q", "-b", "feat/remote")
	runIn("push", "-q", "-u", "origin", "feat/remote")
	runIn("checkout", "-q", "main")

	chdir(t, repo)
	res, err := DeleteBranch("feat/remote", true, false)
	if err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}
	if !res.RemoteDeleted {
		t.Errorf("远程分支未删除: %s", res.RemoteMessage)
	}

	// 验证远程确实没有该分支
	cmd := exec.Command("git", "ls-remote", "--heads", "origin", "feat/remote")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Errorf("远程仍有该分支: %s", out)
	}
}
