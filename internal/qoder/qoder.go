// Package qoder 通过 Qoder CLI 或 Qoder 应用打开工作树，使其作为 Qoder 工作区使用。
package qoder

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNotFound 表示本机既没有 Qoder CLI，也没有可用于打开目录的 Qoder 应用。
var ErrNotFound = errors.New("未找到 Qoder CLI 或 Qoder 应用")

// OpenWorkspace 在 Qoder 中打开 dir，优先调用 qoder CLI（会创建工作区窗口）。
// 找不到 CLI 时回退到系统应用打开，返回实际使用的方式供调用方提示。
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

	if cli, ok := findCLI(); ok {
		if err := runCLI(cli, abs); err != nil {
			return "", fmt.Errorf("调用 Qoder CLI 打开工作树失败: %w", err)
		}
		return "Qoder CLI", nil
	}

	via, err := openWithApp(abs)
	if err != nil {
		return "", fmt.Errorf("%w（%v）；可在 Qoder 中手动打开: %s", ErrNotFound, err, abs)
	}
	return via, nil
}

// findCLI 依次查找 QG_QODER_CLI、PATH 中的 qoder、各平台常见安装路径。
func findCLI() (string, bool) {
	if path := strings.TrimSpace(os.Getenv("QG_QODER_CLI")); path != "" && isFile(path) {
		return path, true
	}
	if path, err := exec.LookPath("qoder"); err == nil {
		return path, true
	}
	for _, path := range cliCandidates() {
		if isFile(path) {
			return path, true
		}
	}
	return "", false
}

// runCLI 优先用 --new-window 打开新工作区；旧版 CLI 不认识该参数时退回直接传目录。
func runCLI(cli, dir string) error {
	var stderr bytes.Buffer
	first := exec.Command(cli, "--new-window", dir)
	first.Stdout = os.Stdout
	first.Stderr = &stderr
	if err := first.Run(); err == nil {
		return nil
	}

	second := exec.Command(cli, dir)
	second.Stdout = os.Stdout
	second.Stderr = os.Stderr
	if err := second.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("%w（--new-window: %s）", err, msg)
		}
		return err
	}
	return nil
}

func isFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
