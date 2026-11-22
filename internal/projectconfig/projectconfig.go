// Package projectconfig 负责定位和解析项目级 qg.hjson 配置。
package projectconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hjson/hjson-go/v4"
	"github.com/zfslalala/cmt/internal/git"
	"github.com/zfslalala/cmt/internal/kb"
	"github.com/zfslalala/cmt/pkg/config"
)

// Config 对应 qg.hjson 的结构（仓库根或知识库项目基座）。
type Config struct {
	Builds map[string]Entry `json:"builds"`
	Wt     struct {
		// Base 新分支的默认起点（如 release），qg wt 不带 --from 时使用。
		Base string `json:"base"`
	} `json:"wt"`
	Cmt Cmt `json:"cmt"`
}

// Cmt 对应 qg.hjson 的 cmt 段：qg cmt 调用大模型所需的配置。
type Cmt struct {
	APIKey string `json:"apiKey"`
	APIUrl string `json:"apiUrl"`
	Model  string `json:"model"`
}

// Entry 是单个分支的构建触发配置。
type Entry struct {
	Curl []string `json:"curl"`
}

// Load 读取 qg.hjson；两处候选文件都不存在时返回 (nil, nil)。
// 查找顺序：当前工作树根目录 → 知识库中该项目基座。
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取 %s 失败: %w", path, err)
	}

	var cfg Config
	if err := hjson.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", path, err)
	}
	return &cfg, nil
}

func configPath() (string, error) {
	root, err := git.RepoRoot()
	if err != nil {
		return "", fmt.Errorf("定位仓库根失败: %w", err)
	}

	local := filepath.Join(root, "qg.hjson")
	if isFile(local) {
		return local, nil
	}

	base, err := kb.BaseDir(root)
	if err != nil {
		return "", fmt.Errorf("定位知识库项目目录失败: %w", err)
	}

	fallback := filepath.Join(base, "qg.hjson")
	if isFile(fallback) {
		return fallback, nil
	}
	return "", nil
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// WorktreeBase 读取 qg.hjson 中 wt.base；文件不存在或未配置时返回空串。
// 配置文件损坏时返回错误，避免静默退回 HEAD。
func WorktreeBase() (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}
	if cfg == nil {
		return "", nil
	}
	return strings.TrimSpace(cfg.Wt.Base), nil
}

// EntryFor 返回指定分支的构建配置；未配置时 ok 为 false。
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

// LLMConfig 返回 qg cmt 使用的大模型配置。
// 未配置的 apiUrl / model 用默认值补齐；apiKey 是否缺失由调用方判断。
func (c *Config) LLMConfig() config.Config {
	cfg := config.Default()
	if c == nil {
		return cfg
	}
	if c.Cmt.APIKey != "" {
		cfg.APIKey = c.Cmt.APIKey
	}
	if c.Cmt.APIUrl != "" {
		cfg.APIBase = c.Cmt.APIUrl
	}
	if c.Cmt.Model != "" {
		cfg.Model = c.Cmt.Model
	}
	return cfg
}
