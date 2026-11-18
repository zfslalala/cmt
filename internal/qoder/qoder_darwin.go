//go:build darwin

package qoder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// cliCandidates 返回 Qoder IDE 常见的 CLI 位置。
// Qoder IDE（com.qoder.ide）自带 app/bin/qoder，支持直接打开目录。
func cliCandidates() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"/Applications/Qoder IDE.app/Contents/Resources/app/bin/qoder",
		filepath.Join(home, "Applications", "Qoder IDE.app", "Contents", "Resources", "app", "bin", "qoder"),
		"/Applications/Qoder.app/Contents/Resources/app/bin/qoder",
		filepath.Join(home, "Applications", "Qoder.app", "Contents", "Resources", "app", "bin", "qoder"),
		"/usr/local/bin/qoder",
		"/opt/Qoder/bin/qoder",
	}
}

// openWithApp 在找不到 CLI 时用系统 open 打开目录。
// 优先 Qoder IDE；仅安装 Qoder Desktop 时也能唤起应用，但桌面版
// 当前不保证把目录创建工作区，调用方应提示用户手动确认。
func openWithApp(dir string) (string, error) {
	apps := []struct {
		bundle string
		name   string
	}{
		{"com.qoder.ide", "Qoder IDE"},
		{"com.qoder.app", "Qoder Desktop"},
	}
	for _, app := range apps {
		if err := exec.Command("open", "-b", app.bundle, dir).Run(); err == nil {
			return app.name, nil
		}
	}
	return "", fmt.Errorf("系统未安装 Qoder")
}
