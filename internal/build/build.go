package build

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/zfslalala/cmt/internal/proc"
)

// commandTimeout 单条命令的超时时间
const commandTimeout = 30 * time.Second

// EnsureFailFlag 给 curl 命令注入 -f(--fail),使 HTTP 4xx/5xx 判定为失败
// (curl 退出码变 22)。非 curl 开头的命令原样返回,按退出码判定。
func EnsureFailFlag(cmdStr string) string {
	trimmed := strings.TrimLeft(cmdStr, " \t")
	if !strings.HasPrefix(trimmed, "curl ") && trimmed != "curl" {
		return cmdStr
	}
	return "curl -f" + strings.TrimPrefix(trimmed, "curl")
}

// RunCommand 通过 shell 执行单条命令(兼容 Windows)。
// Windows 优先用 Git 自带的 sh(POSIX 语法,浏览器「Copy as cURL (bash)」可原样粘贴),
// 找不到时回退 cmd(以原始命令行传入,由 cmd 自己解析引号)。
// 命令运行在独立进程组中,超时时终止整个进程组,避免孤儿进程残留。
func RunCommand(cmdStr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := shellCommand(cmdStr)
	proc.SetProcessGroup(cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()

	select {
	case err := <-waitErr:
		return err
	case <-ctx.Done():
		proc.KillProcessGroup(cmd)
		<-waitErr
		return fmt.Errorf("命令超时(%s)", commandTimeout)
	}
}
