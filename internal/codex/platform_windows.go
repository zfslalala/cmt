//go:build windows

package codex

import (
	"os"
	"path/filepath"
)

// platformCLICandidates 返回 Windows 上常见的 ChatGPT/Codex 安装路径。
func platformCLICandidates(string) []string {
	var appRoots []string
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		appRoots = append(appRoots, filepath.Join(local, "Programs"))
	}
	for _, name := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if root := os.Getenv(name); root != "" {
			appRoots = append(appRoots, root)
		}
	}

	var candidates []string
	for _, root := range appRoots {
		candidates = append(candidates,
			filepath.Join(root, "ChatGPT", "resources", "codex-cli", "bin", "codex.exe"),
			filepath.Join(root, "Codex", "resources", "codex-cli", "bin", "codex.exe"),
		)
	}
	return candidates
}
