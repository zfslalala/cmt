// Package codex 通过 Codex CLI 的 `codex app <path>` 在工作区中打开 Desktop。
package codex

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNotFound 表示本机既没有 PATH 中的 codex，也没有可用的 Codex Desktop 安装。
var ErrNotFound = errors.New("未找到 Codex CLI；请安装 Codex 桌面端，或将 codex 加入 PATH")

// OpenWorkspace 在 Codex Desktop 中打开 dir 对应的工作区。
// 返回实际使用的方式，当前统一为 Codex Desktop。
func OpenWorkspace(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("解析工作树路径失败: %w", err)
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("工作树目录不可用: %w", err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("工作树路径不是目录: %s", abs)
	}

	cli, err := findCLI()
	if err != nil {
		return "", err
	}

	var output bytes.Buffer
	cmd := exec.Command(cli, "app", abs)
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(output.String())
		if strings.Contains(strings.ToLower(detail), "unrecognized subcommand") ||
			strings.Contains(strings.ToLower(detail), "unknown subcommand") {
			return "", fmt.Errorf("当前 Codex CLI 不支持 `codex app`，请更新 Codex 后重试")
		}
		if detail != "" {
			return "", fmt.Errorf("调用 Codex Desktop 失败: %w: %s", err, detail)
		}
		return "", fmt.Errorf("调用 Codex Desktop 失败: %w", err)
	}

	return "Codex Desktop", nil
}

// findCLI 依次查找 QG_CODEX_CLI、PATH 中的 codex、各平台常见安装路径。
func findCLI() (string, error) {
	if path := strings.TrimSpace(os.Getenv("QG_CODEX_CLI")); path != "" {
		if !isFile(path) {
			return "", fmt.Errorf("QG_CODEX_CLI 指向的文件不存在: %s", path)
		}
		return path, nil
	}
	if path, err := exec.LookPath("codex"); err == nil {
		return path, nil
	}
	for _, path := range cliCandidates() {
		if isFile(path) {
			return path, nil
		}
	}
	return "", ErrNotFound
}

func cliCandidates() []string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "bin", "codex"),
		filepath.Join(home, ".codex", "bin", "codex"),
	}
	return append(candidates, platformCLICandidates(home)...)
}

func isFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
