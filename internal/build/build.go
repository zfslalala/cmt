package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hjson/hjson-go/v4"
	"github.com/zfslalala/cmt/internal/git"
	"github.com/zfslalala/cmt/internal/proc"
)

// Config 对应仓库根 qg.hjson 的结构
type Config struct {
	Builds map[string]Entry `json:"builds"`
	Wt     struct {
		// Base 新分支的默认起点（如 release），qg wt 不带 --from 时使用
		Base string `json:"base"`
	} `json:"wt"`
}

// Entry 是单个分支的构建触发配置
type Entry struct {
	Curl []string `json:"curl"`
}

// commandTimeout 单条命令的超时时间
const commandTimeout = 30 * time.Second

// Load 从 git 仓库根读取 qg.hjson;文件不存在时返回 (nil, nil)。
// 使用 HJSON 解析(JSON 超集):兼容标准 JSON 写法,
// 并支持三引号多行字符串,可直接粘贴浏览器复制的 curl 命令而无需转义。
func Load() (*Config, error) {
	root, err := git.RepoRoot()
	if err != nil {
		return nil, fmt.Errorf("定位仓库根失败: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "qg.hjson"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cfg Config
	if err := hjson.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析 qg.hjson 失败: %w", err)
	}
	return &cfg, nil
}

// WorktreeBase 读取 qg.hjson 中 wt.base 配置（新分支默认起点）。
// 文件不存在或未配置时返回空串；配置文件损坏时仅告警不阻断。
func WorktreeBase() string {
	cfg, err := Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ 读取 qg.hjson 失败: %v\n", err)
		return ""
	}
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.Wt.Base)
}

// EntryFor 返回指定分支的构建配置;未配置时 ok 为 false
func (c *Config) EntryFor(branch string) (Entry, bool) {
	if c == nil {
		return Entry{}, false
	}
	entry, ok := c.Builds[branch]
	if !ok || len(entry.Curl) == 0 {
		return Entry{}, false
	}
	return entry, true
}

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
		return fmt.Errorf("命令超时(%s): %s", commandTimeout, cmdStr)
	}
}
