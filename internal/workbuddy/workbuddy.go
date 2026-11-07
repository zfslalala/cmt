// Package workbuddy 通过 workbuddy:// 深链接在指定目录打开 WorkBuddy 新任务。
package workbuddy

import (
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"runtime"
)

// OpenTask 让 WorkBuddy 以 dir 为工作目录打开一个新任务。
// prompt 预填进输入框草稿态，不会自动发送；为空时用默认提示语。
// 链接递交给系统协议处理程序后立即返回。
func OpenTask(dir string, prompt string) error {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("解析目录失败: %w", err)
	}
	prompt = ""
	link := fmt.Sprintf("workbuddy://task?action=start&prompt=%s&cwd=%s",
		url.QueryEscape(prompt), url.QueryEscape(dir))

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", link)
	case "windows":
		// rundll32 走系统已注册的协议处理程序，exec 直传参数无需 shell 转义
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", link)
	default:
		cmd = exec.Command("xdg-open", link)
	}
	// open/rundll32 递交链接后立即退出，用 Run 便于感知协议是否可用
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("无法打开 WorkBuddy（未安装或协议未注册？）: %w", err)
	}
	return nil
}
