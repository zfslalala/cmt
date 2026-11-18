//go:build windows

package qoder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// cliCandidates 返回 Windows 上常见的 Qoder CLI / 启动器位置。
func cliCandidates() []string {
	var paths []string
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		paths = append(paths,
			filepath.Join(local, "Programs", "Qoder", "bin", "qoder.cmd"),
			filepath.Join(local, "Programs", "Qoder", "Qoder.exe"),
			filepath.Join(local, "Qoder", "Qoder Launcher", "Qoder Launcher.exe"),
		)
	}
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if dir := os.Getenv(env); dir != "" {
			paths = append(paths,
				filepath.Join(dir, "Qoder", "bin", "qoder.cmd"),
				filepath.Join(dir, "Qoder", "Qoder.exe"),
			)
		}
	}
	return paths
}

// openWithApp 在找不到 CLI 时直接启动 Qoder 可执行文件并传入目录。
func openWithApp(dir string) (string, error) {
	for _, path := range cliCandidates() {
		if !isFile(path) {
			continue
		}
		cmd := exec.Command(path, dir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", err
		}
		return "Qoder", nil
	}
	return "", fmt.Errorf("系统未安装 Qoder")
}
