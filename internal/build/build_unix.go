//go:build !windows

package build

import (
	"os/exec"
	"syscall"
)

// shellCommand 在 Unix 上用 sh 解析执行命令串
func shellCommand(cmdStr string) *exec.Cmd {
	return exec.Command("sh", "-c", cmdStr)
}

// setProcessGroup 让命令运行在独立进程组中
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup 终止整个进程组,避免 shell 被杀后子进程残留
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
