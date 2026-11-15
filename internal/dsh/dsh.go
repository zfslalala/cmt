// Package dsh 在指定目录启动 DeepSeek Harness Web 会话。
package dsh

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ErrNotFound 表示本机未检测到 dsh 命令。
var ErrNotFound = errors.New("未检测到 dsh 命令")

// OpenWorkspace 以 dir 为默认目录启动 `dsh web --port 0`。
// 端口交给操作系统分配，避免同时打开多个工作树时抢占默认端口。
// 启动后立即返回，不等待 Web 服务退出；浏览器由 dsh 自行打开。
func OpenWorkspace(dir string) error {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("解析目录失败: %w", err)
	}
	if st, err := os.Stat(dir); err != nil {
		return fmt.Errorf("工作树目录不可用: %w", err)
	} else if !st.IsDir() {
		return fmt.Errorf("工作树路径不是目录: %s", dir)
	}

	exe, err := exec.LookPath("dsh")
	if err != nil {
		return fmt.Errorf("%w，请先在 DeepSeek Harness 菜单「管理 dsh 命令…」中安装", ErrNotFound)
	}

	cmd := newCommand(exe, []string{"web", "--port", "0"})
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 dsh web 失败: %w", err)
	}
	// 不等待 Web 服务退出，释放子进程资源
	_ = cmd.Process.Release()
	return nil
}
