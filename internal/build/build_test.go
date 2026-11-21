package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zfslalala/cmt/pkg/config"
)

// initRepo 在 parent 下创建仓库 name 并提交一次，返回仓库根目录。
func initRepo(t *testing.T, parent, name string) string {
	t.Helper()

	repo := filepath.Join(parent, name)
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("创建仓库目录失败: %v", err)
	}

	runGit(t, repo, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("demo\n"), 0o644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-qm", "init")

	return repo
}

// writeConfig 在 dir 下写入一份只关心 wt.base 的 qg.hjson。
func writeConfig(t *testing.T, dir, base string) {
	t.Helper()

	// HJSON 是 JSON 超集，测试里直接用合法 JSON 更稳。
	content := `{"builds":{"main":{"curl":["echo ` + base + `"]}},"wt":{"base":"` + base + `"}}`
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("创建配置目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "qg.hjson"), []byte(content), 0o644); err != nil {
		t.Fatalf("写入 qg.hjson 失败: %v", err)
	}
}

// chdir 切换进程工作目录，因为 Load 依赖当前目录定位仓库根。
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

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v 失败: %v\n%s", args, err, out)
	}
	return string(out)
}

func TestLoadPrefersRepoRootOverKnowledgeBase(t *testing.T) {
	parent := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(parent); err == nil {
		parent = resolved
	}
	t.Setenv("AI_KB_ROOT", "")

	repo := initRepo(t, parent, "demo")
	writeConfig(t, repo, "root-base")
	writeConfig(t, filepath.Join(parent, "_ai-kb", "projects", "demo", "base"), "kb-base")

	chdir(t, repo)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if cfg == nil {
		t.Fatal("期望读取到仓库根的 qg.hjson，实际为 nil")
	}
	if cfg.Wt.Base != "root-base" {
		t.Fatalf("期望仓库根配置优先，wt.base = %q", cfg.Wt.Base)
	}
}

func TestLoadFallsBackToKnowledgeBase(t *testing.T) {
	parent := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(parent); err == nil {
		parent = resolved
	}
	t.Setenv("AI_KB_ROOT", "")

	repo := initRepo(t, parent, "demo")
	writeConfig(t, filepath.Join(parent, "_ai-kb", "projects", "demo", "base"), "kb-base")

	chdir(t, repo)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if cfg == nil {
		t.Fatal("仓库根缺少 qg.hjson 时应回退到知识库基座，实际为 nil")
	}
	if cfg.Wt.Base != "kb-base" {
		t.Fatalf("期望读取知识库基座配置，wt.base = %q", cfg.Wt.Base)
	}
	if entry, ok := cfg.EntryFor("main"); !ok || len(entry.Curl) != 1 || entry.Curl[0] != "echo kb-base" {
		t.Fatalf("期望命中知识库基座的构建命令，实际 entry = %+v, ok = %v", entry, ok)
	}
}

func TestLoadInLinkedWorktreeUsesMainWorktreeKnowledgeBase(t *testing.T) {
	parent := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(parent); err == nil {
		parent = resolved
	}
	t.Setenv("AI_KB_ROOT", "")

	repo := initRepo(t, parent, "demo")
	writeConfig(t, filepath.Join(parent, "_ai-kb", "projects", "demo", "base"), "kb-base")

	wtPath := filepath.Join(parent, "demo-feat")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", wtPath)

	chdir(t, wtPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if cfg == nil {
		t.Fatal("工作树缺少 qg.hjson 时应回退到主工作树对应项目的基座，实际为 nil")
	}
	if cfg.Wt.Base != "kb-base" {
		t.Fatalf("期望读取主工作树项目的基座配置，wt.base = %q", cfg.Wt.Base)
	}
}

func TestLoadReturnsNilWhenNoConfigAnywhere(t *testing.T) {
	parent := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(parent); err == nil {
		parent = resolved
	}

	// 指向一个空目录，避免机器上真实存在的 ~/.ai-kb 干扰结果。
	t.Setenv("AI_KB_ROOT", filepath.Join(parent, "empty-kb"))

	repo := initRepo(t, parent, "demo")
	chdir(t, repo)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if cfg != nil {
		t.Fatalf("两处都没有 qg.hjson 时应返回 nil，实际 %+v", cfg)
	}
}

func TestLoadReadsCmtSection(t *testing.T) {
	parent := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(parent); err == nil {
		parent = resolved
	}
	t.Setenv("AI_KB_ROOT", "")

	repo := initRepo(t, parent, "demo")
	content := `{"cmt":{"apiKey":"test-key","apiUrl":"https://llm.example.com/v1","model":"test-model"}}`
	if err := os.WriteFile(filepath.Join(repo, "qg.hjson"), []byte(content), 0o644); err != nil {
		t.Fatalf("写入 qg.hjson 失败: %v", err)
	}

	chdir(t, repo)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if cfg == nil {
		t.Fatal("期望读取到 qg.hjson，实际为 nil")
	}
	if cfg.Cmt.APIKey != "test-key" || cfg.Cmt.APIUrl != "https://llm.example.com/v1" || cfg.Cmt.Model != "test-model" {
		t.Fatalf("cmt 段解析结果不符: %+v", cfg.Cmt)
	}
}

func TestLLMConfigUsesQGHjsonValues(t *testing.T) {
	cfg := &Config{Cmt: Cmt{APIKey: "k", APIUrl: "https://llm.example.com", Model: "m"}}

	got := cfg.LLMConfig()
	if got.APIKey != "k" || got.APIBase != "https://llm.example.com" || got.Model != "m" {
		t.Fatalf("期望使用 qg.hjson 的值，实际 %+v", got)
	}
}

func TestLLMConfigFillsDefaultsForMissingFields(t *testing.T) {
	cfg := &Config{Cmt: Cmt{APIKey: "k"}}

	got := cfg.LLMConfig()
	if got.APIKey != "k" {
		t.Fatalf("apiKey = %q", got.APIKey)
	}
	if got.APIBase != config.DefaultAPIBase || got.Model != config.DefaultModel {
		t.Fatalf("期望补齐默认值，实际 %+v", got)
	}
	if got.Temperature != 0.7 || got.MaxTokens != 1000 {
		t.Fatalf("默认温度/最大 token 不符: %+v", got)
	}
}

func TestLLMConfigOnNilConfig(t *testing.T) {
	var cfg *Config

	got := cfg.LLMConfig()
	if got.APIKey != "" {
		t.Fatalf("未读取到配置时 apiKey 应为空，实际 %q", got.APIKey)
	}
	if got.APIBase != config.DefaultAPIBase || got.Model != config.DefaultModel {
		t.Fatalf("期望返回默认值，实际 %+v", got)
	}
}

func TestWorktreeBaseReportsInvalidConfig(t *testing.T) {
	parent := t.TempDir()
	t.Setenv("AI_KB_ROOT", "")
	repo := initRepo(t, parent, "demo")
	if err := os.WriteFile(filepath.Join(repo, "qg.hjson"), []byte("{ invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	chdir(t, repo)

	if _, err := WorktreeBase(); err == nil {
		t.Fatal("损坏的 qg.hjson 应返回错误，避免静默退回 HEAD")
	}
}
