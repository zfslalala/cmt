//go:build !windows

package build

import (
	"os/exec"
)

// shellCommand 在 Unix 上用 sh 解析执行命令串
func shellCommand(cmdStr string) *exec.Cmd {
	return exec.Command("sh", "-c", cmdStr)
}
