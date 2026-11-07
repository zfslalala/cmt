package git

import (
	"os"
	"os/exec"
	"testing"
)

// 用真实 git 仓库验证 -z 解析：路径含空格、重命名状态归一
func TestGetStagedFiles(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}

	run("init", "-q")
	run("config", "user.email", "t@t")
	run("config", "user.name", "t")

	write := func(name, content string) {
		if err := exec.Command("sh", "-c", "printf '%s' \""+content+"\" > \""+dir+"/"+name+"\"").Run(); err != nil {
			t.Fatalf("写文件 %s: %v", name, err)
		}
	}

	write("a file.txt", "hello")
	write("b.txt", "world")
	run("add", ".")
	run("commit", "-q", "-m", "init")

	// 修改 b.txt、重命名带空格的 a file.txt
	write("b.txt", "world v2")
	run("mv", "a file.txt", "renamed file.txt")
	run("add", "-A")

	// getStagedFiles 在进程工作目录执行 git，需切入临时仓库
	prevWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(prevWd) }()

	files, err := getStagedFiles()
	if err != nil {
		t.Fatalf("getStagedFiles: %v", err)
	}

	byStatus := map[string]FileChange{}
	for _, f := range files {
		byStatus[f.Status] = f
	}

	r, ok := byStatus["R"]
	if !ok {
		t.Fatalf("重命名状态未归一为 R, files=%+v", files)
	}
	if r.OldPath != "a file.txt" || r.NewPath != "renamed file.txt" {
		t.Errorf("重命名路径解析错误: %+v", r)
	}
	if m, ok := byStatus["M"]; !ok || m.NewPath != "b.txt" {
		t.Errorf("修改文件解析错误: %+v", byStatus)
	}
}
