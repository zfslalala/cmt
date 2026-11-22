//go:build !darwin && !windows

package codex

// platformCLICandidates 在 Linux 上依赖 PATH 与通用用户目录。
func platformCLICandidates(string) []string {
	return nil
}
