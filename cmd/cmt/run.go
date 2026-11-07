package cmt

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zfslalala/cmt/internal/runner"
)

// runRun 处理 qg run：选项必须在脚本名之前，脚本名之后的参数原样透传给脚本
func runRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: qg run <脚本> [参数...]\n脚本可直接给路径，或放脚本目录（默认 ~/.qg/scripts，可用 QG_SCRIPTS_DIR 覆盖）后用名字调用")
	}

	timeout := time.Duration(0)
	i := 0
loop:
	for ; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			i++
			break loop
		case !strings.HasPrefix(a, "-"):
			break loop
		case a == "-h" || a == "--help":
			fmt.Print(cmd.UsageString())
			return nil
		case a == "--timeout":
			i++
			if i >= len(args) {
				return fmt.Errorf("--timeout 需要一个秒数，例如 --timeout 300")
			}
			d, err := parseTimeoutSeconds(args[i])
			if err != nil {
				return err
			}
			timeout = d
		case strings.HasPrefix(a, "--timeout="):
			d, err := parseTimeoutSeconds(strings.TrimPrefix(a, "--timeout="))
			if err != nil {
				return err
			}
			timeout = d
		default:
			return fmt.Errorf("未知选项: %s（可用选项: --timeout <秒>）", a)
		}
	}

	if i >= len(args) {
		return fmt.Errorf("缺少脚本名")
	}
	script := args[i]
	scriptArgs := args[i+1:]

	path, err := runner.Resolve(script)
	if err != nil {
		return err
	}

	code, err := runner.Run(path, scriptArgs, timeout)
	if err != nil {
		return err
	}
	if code != 0 {
		os.Exit(code)
	}
	return nil
}

func parseTimeoutSeconds(s string) (time.Duration, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("无效的超时秒数: %s", s)
	}
	return time.Duration(n) * time.Second, nil
}
