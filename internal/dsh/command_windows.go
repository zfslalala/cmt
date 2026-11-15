//go:build windows

package dsh

import (
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// newCommand Windows 下 .cmd/.bat 需经 cmd.exe 执行。
// 用 SysProcAttr.CmdLine 传原始命令行，避免 Go 的参数转义破坏路径引号。
func newCommand(exe string, args []string) *exec.Cmd {
	switch strings.ToLower(filepath.Ext(exe)) {
	case ".cmd", ".bat":
		parts := make([]string, 0, len(args)+1)
		parts = append(parts, quote(exe))
		for _, arg := range args {
			parts = append(parts, quote(arg))
		}
		cmd := exec.Command("cmd.exe")
		cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: "cmd.exe /c " + strings.Join(parts, " ")}
		return cmd
	default:
		return exec.Command(exe, args...)
	}
}

func quote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
