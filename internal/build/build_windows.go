//go:build windows

package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// shellCommand 选定执行命令串所用的 shell:
//  1. 优先 Git 自带的 sh.exe —— POSIX 语法,与 Linux/macOS 行为一致,
//     浏览器「Copy as cURL (bash)」的内容可原样粘贴。
//  2. 找不到时回退 cmd.exe —— 必须用 SysProcAttr.CmdLine 传原始命令行,
//     绕过 Go 的参数转义(否则内层引号会被转义成 \" 原样进入 curl 的 argv)。
//     此路径要求命令为 cmd 兼容格式(双引号、单行)。
func shellCommand(cmdStr string) *exec.Cmd {
	if sh, ok := findGitSh(); ok {
		return exec.Command(sh, "-c", cmdStr)
	}
	cmd := exec.Command("cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: "cmd /c " + cmdStr, HideWindow: true}
	return cmd
}

// findGitSh 从 PATH 中 git.exe 的位置推导 Git 安装目录下的 usr\bin\sh.exe。
// git.exe 常见位于 <安装目录>\cmd\、<安装目录>\mingw64\bin\、<安装目录>\bin\,
// 逐级向上最多找 3 层,命中即返回。sh 与 qg 依赖的 git 同源,装了 Git 必然存在。
func findGitSh() (string, bool) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return "", false
	}
	dir := filepath.Dir(gitPath)
	for i := 0; i < 3; i++ {
		sh := filepath.Join(dir, "usr", "bin", "sh.exe")
		if _, err := os.Stat(sh); err == nil {
			return sh, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}
