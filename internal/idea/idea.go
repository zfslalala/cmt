// Package idea 提供 IntelliJ IDEA 的检测与命令行启动能力（macOS / Windows）。
package idea

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// ErrNotFound 表示本机未检测到 IntelliJ IDEA。
var ErrNotFound = errors.New("未检测到 IntelliJ IDEA")

// launcher 描述一种可用的启动方式：exec.Command(exe, append(args, dir)...)
type launcher struct {
	exe  string
	args []string
}

// Open 在 IntelliJ IDEA 中打开指定目录。
// 启动后立即返回，不等待 IDE 退出。
func Open(dir string) error {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("解析目录失败: %w", err)
	}

	l, err := findLauncher()
	if err != nil {
		return err
	}

	cmd := exec.Command(l.exe, append(append([]string{}, l.args...), dir)...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 IntelliJ IDEA 失败: %w", err)
	}
	// 不等待 IDE 退出，释放子进程资源
	_ = cmd.Process.Release()
	return nil
}

// findLauncher 按平台探测可用的 IDEA 启动方式
func findLauncher() (launcher, error) {
	switch runtime.GOOS {
	case "darwin":
		return findDarwin()
	case "windows":
		return findWindows()
	default:
		return findUnix()
	}
}

// findDarwin 探测 macOS 下的启动方式：
//  1. PATH 中的 idea（Toolbox 生成的启动脚本）
//  2. Toolbox 脚本目录下的 idea
//  3. /Applications 或 ~/Applications 下的 IntelliJ IDEA*.app，用 open -a 启动
func findDarwin() (launcher, error) {
	if p, err := exec.LookPath("idea"); err == nil {
		return launcher{exe: p}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	toolboxScript := filepath.Join(home, "Library", "Application Support", "JetBrains", "Toolbox", "scripts", "idea")
	if st, err := os.Stat(toolboxScript); err == nil && !st.IsDir() {
		return launcher{exe: toolboxScript}, nil
	}

	// 按常见命名优先级查找 .app
	preferred := []string{"IntelliJ IDEA.app", "IntelliJ IDEA Ultimate.app", "IntelliJ IDEA CE.app"}
	for _, dir := range []string{"/Applications", filepath.Join(home, "Applications")} {
		for _, name := range preferred {
			app := filepath.Join(dir, name)
			if st, err := os.Stat(app); err == nil && st.IsDir() {
				return launcher{exe: "open", args: []string{"-a", app}}, nil
			}
		}
		matches, _ := filepath.Glob(filepath.Join(dir, "IntelliJ IDEA*.app"))
		if len(matches) > 0 {
			sort.Strings(matches)
			return launcher{exe: "open", args: []string{"-a", matches[0]}}, nil
		}
	}
	return launcher{}, ErrNotFound
}

// findWindows 探测 Windows 下的启动方式：
//  1. PATH 中的 idea64.exe / idea（.bat/.cmd）
//  2. 常见安装目录下的 bin\idea64.exe（含 Toolbox 1.x / 2.x 布局）
//  3. Toolbox 脚本目录下的 idea*.cmd / idea*.bat
func findWindows() (launcher, error) {
	for _, name := range []string{"idea64.exe", "idea.cmd", "idea.bat", "idea"} {
		if p, err := exec.LookPath(name); err == nil {
			return launcher{exe: p}, nil
		}
	}

	localAppData := os.Getenv("LOCALAPPDATA")
	roots := make([]string, 0, 4)
	if localAppData != "" {
		roots = append(roots,
			filepath.Join(localAppData, "Programs"),
			filepath.Join(localAppData, "JetBrains", "Toolbox", "apps"),
			filepath.Join(localAppData, "JetBrains", "Toolbox", "scripts"),
		)
	}
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if pf := os.Getenv(env); pf != "" {
			roots = append(roots, filepath.Join(pf, "JetBrains"))
		}
	}

	patterns := []string{
		"IntelliJ IDEA*/bin/idea64.exe", // 独立安装 / Toolbox 2.x
		"*/bin/idea64.exe",              // Toolbox 2.x 布局
		"*/*/bin/idea64.exe",            // Toolbox 1.x: apps\IDEA-U\ch-0\<ver>\bin
		"*/*/*/bin/idea64.exe",
		"idea*.cmd", // Toolbox 脚本
		"idea*.bat",
	}

	var found []string
	for _, root := range roots {
		for _, pattern := range patterns {
			matches, _ := filepath.Glob(filepath.Join(root, pattern))
			found = append(found, matches...)
		}
	}
	if len(found) > 0 {
		sort.Strings(found)
		return launcher{exe: found[0]}, nil
	}
	return launcher{}, ErrNotFound
}

// findUnix 探测 Linux 下的启动方式（简单兜底）
func findUnix() (launcher, error) {
	if p, err := exec.LookPath("idea"); err == nil {
		return launcher{exe: p}, nil
	}
	for _, p := range []string{"/snap/bin/idea", "/usr/local/bin/idea"} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() && !strings.HasSuffix(p, "/") {
			return launcher{exe: p}, nil
		}
	}
	return launcher{}, ErrNotFound
}
