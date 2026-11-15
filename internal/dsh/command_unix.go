//go:build !windows

package dsh

import "os/exec"

// newCommand 在 Unix 上直接执行 dsh（PATH 中通常是可执行脚本或二进制）。
func newCommand(exe string, args []string) *exec.Cmd {
	return exec.Command(exe, args...)
}
