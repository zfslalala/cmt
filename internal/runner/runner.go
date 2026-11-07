// Package runner 提供 qg run 的脚本解析与执行能力。
package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/zfslalala/cmt/internal/build"
)

// ScriptsDir 返回脚本目录：QG_SCRIPTS_DIR 环境变量优先，否则 ~/.qg/scripts
func ScriptsDir() (string, error) {
	if dir := os.Getenv("QG_SCRIPTS_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}
	return filepath.Join(home, ".qg", "scripts"), nil
}

// Resolve 将用户输入解析为脚本文件路径。
// 依次尝试：显式路径（自动补 .go 扩展名）→ 脚本目录下的 <名字>.go
func Resolve(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("脚本名不能为空")
	}

	// 含路径分隔符或已带 .go 扩展名，按显式路径处理
	if strings.ContainsAny(name, `/\`) || strings.HasSuffix(name, ".go") {
		p := name
		if !strings.HasSuffix(p, ".go") {
			p += ".go"
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return filepath.Abs(p)
		}
		return "", fmt.Errorf("脚本不存在: %s", p)
	}

	// 约定目录下按名字查找
	dir, err := ScriptsDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, name+".go")
	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		return p, nil
	}
	return "", fmt.Errorf("脚本不存在: %s（脚本目录: %s）", name, dir)
}

// Run 执行 Go 脚本：先 go build 到临时目录，再运行产物。
// 不用 go run 是因为它不透传脚本退出码（固定返回 1 并打印 "exit status N"）。
// stdout/stderr 实时直连控制台，stdin 直连终端。
// timeout <= 0 表示运行阶段不限时；超时终止整个进程组。
// 返回脚本退出码（脚本内部 os.Exit 的值；编译失败返回 go build 的退出码）。
func Run(scriptPath string, args []string, timeout time.Duration) (int, error) {
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		return 1, fmt.Errorf("解析脚本路径失败: %w", err)
	}

	goBin, err := exec.LookPath("go")
	if err != nil {
		return 1, fmt.Errorf("未找到 go 命令，请先安装 Go 工具链（https://go.dev/dl/）")
	}

	// 编译到临时目录，结束后清理
	tmpDir, err := os.MkdirTemp("", "qg-run-")
	if err != nil {
		return 1, fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	binPath := filepath.Join(tmpDir, "qg-script")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	buildCmd := exec.Command(goBin, "build", "-o", binPath, absPath)
	buildCmd.Stderr = os.Stderr // 编译错误实时显示
	build.SetProcessGroup(buildCmd)
	if err := buildCmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("编译脚本失败: %w", err)
	}

	cmd := exec.Command(binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	build.SetProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("启动脚本失败: %w", err)
	}

	if timeout <= 0 {
		err = cmd.Wait()
	} else {
		err = waitWithTimeout(cmd, timeout)
	}
	if err == nil {
		return 0, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	return 1, err
}

// waitWithTimeout 带超时等待，超时杀整个进程组
func waitWithTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()

	select {
	case err := <-waitErr:
		return err
	case <-time.After(timeout):
		build.KillProcessGroup(cmd)
		<-waitErr
		return fmt.Errorf("脚本超时(%s)，已终止", timeout)
	}
}
