//go:build !windows

// Package proc 提供跨平台的进程组管理：把子进程放进独立进程组，
// 超时/取消时能连子进程一起终止，避免孤儿进程残留。
package proc

import (
	"os/exec"
	"syscall"
)

// SetProcessGroup 让命令运行在独立进程组中
func SetProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// KillProcessGroup 终止整个进程组
func KillProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
