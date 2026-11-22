//go:build darwin

package codex

import "path/filepath"

// platformCLICandidates 返回 macOS 上 ChatGPT/Codex 应用包内置的 CLI 路径。
func platformCLICandidates(home string) []string {
	var candidates []string
	for _, root := range []string{"/Applications", filepath.Join(home, "Applications")} {
		candidates = append(candidates,
			filepath.Join(root, "ChatGPT.app", "Contents", "Resources", "codex-cli", "bin", "codex"),
			filepath.Join(root, "Codex.app", "Contents", "Resources", "codex-cli", "bin", "codex"),
		)
	}
	return candidates
}
