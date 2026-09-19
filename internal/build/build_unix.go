//go:build !windows

package build

import (
	"os/exec"
	"syscall"
)

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
