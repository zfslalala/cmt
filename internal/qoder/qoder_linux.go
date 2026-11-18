//go:build linux

package qoder

import "fmt"

// cliCandidates 返回 Linux 上常见的 qoder 命令位置。
func cliCandidates() []string {
	return []string{
		"/usr/bin/qoder",
		"/usr/local/bin/qoder",
		"/opt/Qoder/bin/qoder",
		"/snap/bin/qoder",
	}
}

// openWithApp 在 Linux 上没有可靠的 Qoder 应用打开入口，只支持 CLI。
func openWithApp(dir string) (string, error) {
	return "", fmt.Errorf("未找到 qoder 命令")
}
