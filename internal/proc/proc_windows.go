//go:build windows

package proc

import (
	"os/exec"
	"strconv"
	"syscall"
)

// SetProcessGroup Windows 无进程组概念，空实现
func SetProcessGroup(cmd *exec.Cmd) {}

// KillProcessGroup 超时后终止整个进程树。
// sh/cmd 下还挂着 curl 等子进程，仅杀 shell 本体会留孤儿，用 taskkill /T 连子进程一起杀。
func KillProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
	kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = kill.Run()
	_ = cmd.Process.Kill()
}
