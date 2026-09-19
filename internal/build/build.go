package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/hjson/hjson-go/v4"
	"github.com/zfslalala/cmt/internal/git"
)

// Config 对应仓库根 qg.hjson 的结构
type Config struct {
	Builds map[string]Entry `json:"builds"`
}

// Entry 是单个分支的构建触发配置
type Entry struct {
	Curl []string `json:"curl"`
}

// commandTimeout 单条命令的超时时间
const commandTimeout = 30 * time.Second

// Load 从 git 仓库根读取 qg.hjson;文件不存在时返回 (nil, nil)。
// 使用 HJSON 解析(JSON 超集):兼容标准 JSON 写法,
// 并支持 ''' 多行字符串,可直接粘贴浏览器复制的 curl 命令而无需转义。
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

// RunCommand 通过 shell 执行单条命令(兼容 Windows)。
// 命令运行在独立进程组中,超时时终止整个进程组,避免孤儿进程残留。
func RunCommand(cmdStr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", cmdStr)
	} else {
		cmd = exec.Command("sh", "-c", cmdStr)
	}
	setProcessGroup(cmd)
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
		killProcessGroup(cmd)
		<-waitErr
		return fmt.Errorf("命令超时(%s): %s", commandTimeout, cmdStr)
	}
}
