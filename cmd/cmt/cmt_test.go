package cmt

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zfslalala/cmt/pkg/config"
)

func TestReadMultiline(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"空输入", "", ""},
		{"直接回车", "\n", ""},
		{"单行", "feat: 支持新功能\n", "feat: 支持新功能"},
		{"多行", "feat: 支持新功能\n补充说明\n\n", "feat: 支持新功能\n补充说明"},
		{"空白行截断", "  first  \n   \nsecond\n", "first"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readMultiline(strings.NewReader(tc.in))
			if err != nil {
				t.Fatalf("readMultiline() error = %v", err)
			}
			if got != tc.want {
				t.Errorf("readMultiline() = %q, want %q", got, tc.want)
			}
		})
	}
}

// setupRepo 创建带一次提交的临时仓库，返回其父目录与仓库根。
func setupRepo(t *testing.T) (string, string) {
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

	// 隔离知识库回退，避免读到机器上真实的 ~/.ai-kb。
	t.Setenv("AI_KB_ROOT", filepath.Join(parent, "empty-kb"))

	return parent, repo
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v 失败: %v\n%s", args, err, out)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()

	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前目录失败: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("切换目录到 %s 失败: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatalf("恢复目录到 %s 失败: %v", old, err)
		}
	})
}

func TestLoadLLMConfigReadsQGHjson(t *testing.T) {
	_, repo := setupRepo(t)

	content := `{"cmt":{"apiKey":"test-key","model":"test-model"}}`
	if err := os.WriteFile(filepath.Join(repo, "qg.hjson"), []byte(content), 0o644); err != nil {
		t.Fatalf("写入 qg.hjson 失败: %v", err)
	}
	chdir(t, repo)

	cfg, err := loadLLMConfig()
	if err != nil {
		t.Fatalf("loadLLMConfig 失败: %v", err)
	}
	if cfg.APIKey != "test-key" || cfg.Model != "test-model" {
		t.Fatalf("配置读取不符: %+v", cfg)
	}
	if cfg.APIBase != config.DefaultAPIBase {
		t.Fatalf("未配置 apiUrl 时应使用默认值，实际 %q", cfg.APIBase)
	}
}

func TestLoadLLMConfigRequiresAPIKey(t *testing.T) {
	_, repo := setupRepo(t)

	if err := os.WriteFile(filepath.Join(repo, "qg.hjson"), []byte(`{"cmt":{"model":"test-model"}}`), 0o644); err != nil {
		t.Fatalf("写入 qg.hjson 失败: %v", err)
	}
	chdir(t, repo)

	_, err := loadLLMConfig()
	if err == nil {
		t.Fatal("缺少 cmt.apiKey 时应报错")
	}
	if !strings.Contains(err.Error(), "cmt.apiKey") {
		t.Fatalf("报错应指向 cmt.apiKey，实际: %v", err)
	}
}
