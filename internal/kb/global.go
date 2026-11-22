package kb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DSHHome 返回 DSH 的配置根目录：DSH_HOME 优先，否则 ~/.dsh。
func DSHHome() string {
	if value := strings.TrimSpace(os.Getenv("DSH_HOME")); value != "" {
		return value
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".dsh")
	}
	return ""
}

// CodexHome 返回 Codex 的配置根目录：CODEX_HOME 优先，否则 ~/.codex。
func CodexHome() string {
	if value := strings.TrimSpace(os.Getenv("CODEX_HOME")); value != "" {
		return value
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".codex")
	}
	return ""
}

// SharedAgentsHome 返回跨 agent 共享配置根目录。
func SharedAgentsHome() string {
	for _, name := range []string{"AGENTS_HOME", "DSH_AGENTS_HOME"} {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".agents")
	}
	return ""
}

// ensureGlobal 补全知识库全局层，并把它挂到 Codex、DSH 与共享 agents 配置目录。
// 整个知识库被删掉后，任意一次 qg kb 操作都能把它重建回来。
func ensureGlobal(layout *Layout) (bool, error) {
	if layout.Root == "" {
		return false, nil
	}

	created := false

	skills := filepath.Join(layout.Root, "skills")
	if !isDir(skills) {
		if err := os.MkdirAll(skills, 0o755); err != nil {
			return false, fmt.Errorf("创建知识库技能目录失败: %w", err)
		}
		created = true
	}

	agents := filepath.Join(layout.Root, "AGENTS.md")
	if !pathExists(agents) {
		if err := os.WriteFile(agents, []byte(globalAgentsTemplate()), 0o644); err != nil {
			return false, fmt.Errorf("写入全局 AGENTS.md 失败: %w", err)
		}
		created = true
	}

	if err := ensureGlobalHome(layout, DSHHome(), agents, skills, true); err != nil {
		return created, err
	}
	if err := ensureGlobalHome(layout, SharedAgentsHome(), agents, skills, false); err != nil {
		return created, err
	}
	if err := ensureGlobalHome(layout, CodexHome(), agents, skills, false); err != nil {
		return created, err
	}
	return created, nil
}

func ensureGlobalHome(layout *Layout, home, agents, skills string, linkSkills bool) error {
	home = strings.TrimSpace(home)
	if home == "" {
		return nil
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return fmt.Errorf("创建 agent 配置目录 %s 失败: %w", home, err)
	}
	if err := linkIfAbsentEmptyOrDangling(agents, filepath.Join(home, "AGENTS.md"), layout); err != nil {
		return err
	}
	if linkSkills {
		if err := linkIfAbsentOrDangling(skills, filepath.Join(home, "skills"), layout); err != nil {
			return err
		}
		return nil
	}
	// Codex 的全局 skills 目录还可能包含系统内置技能，只补缺失的项目技能。
	return linkSkillEntries(skills, filepath.Join(home, "skills"), layout)
}

// linkIfAbsentOrDangling 只在目标缺失、或是指向已消失目标的悬空链接时建立链接。
// 目标已是有效链接或真实文件时一律不动，避免踩坏用户自己的配置。
func linkIfAbsentOrDangling(src, dst string, layout *Layout) error {
	info, err := os.Lstat(dst)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return fmt.Errorf("检查 %s 失败: %w", dst, err)
	default:
		if info.Mode()&os.ModeSymlink == 0 {
			layout.Warnings = append(layout.Warnings, fmt.Sprintf("%s 已存在且不是符号链接，未覆盖", dst))
			return nil
		}
		current, err := os.Readlink(dst)
		if err != nil {
			return fmt.Errorf("读取链接失败: %w", err)
		}
		if _, err := os.Stat(dst); err == nil {
			if current != src {
				layout.Warnings = append(layout.Warnings, fmt.Sprintf("%s 指向别处（%s），未覆盖", dst, current))
			}
			return nil
		}
		// 悬空链接：目标已消失，重建
		if err := os.Remove(dst); err != nil {
			return fmt.Errorf("移除悬空链接失败: %w", err)
		}
	}

	if err := os.Symlink(src, dst); err != nil {
		return fmt.Errorf("创建全局链接失败: %w", err)
	}

	return nil
}

// linkIfAbsentEmptyOrDangling 额外允许接管内容为空的普通文件，
// 用于 Codex 默认创建的空 ~/.codex/AGENTS.md。
func linkIfAbsentEmptyOrDangling(src, dst string, layout *Layout) error {
	info, err := os.Lstat(dst)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return fmt.Errorf("检查 %s 失败: %w", dst, err)
	default:
		if info.Mode()&os.ModeSymlink != 0 {
			return linkIfAbsentOrDangling(src, dst, layout)
		}
		if info.Mode().IsRegular() && info.Size() == 0 {
			if err := os.Remove(dst); err != nil {
				return fmt.Errorf("移除空文件 %s 失败: %w", dst, err)
			}
			break
		}
		layout.Warnings = append(layout.Warnings, fmt.Sprintf("%s 已存在且不是空文件/符号链接，未覆盖", dst))
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	if err := os.Symlink(src, dst); err != nil {
		return fmt.Errorf("创建全局链接失败: %w", err)
	}
	return nil
}

// linkSkillEntries 把知识库中的单个技能链接进已有 skills 目录，
// 避免覆盖目标目录里的系统或用户技能。
func linkSkillEntries(srcDir, dstDir string, layout *Layout) error {
	entries, err := os.ReadDir(srcDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取全局技能目录失败: %w", err)
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("创建全局技能目录失败: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			continue
		}
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())
		if err := linkIfAbsentOrDangling(src, dst, layout); err != nil {
			return err
		}
	}
	return nil
}

// globalAgentsTemplate 是全局 AGENTS.md 的初始内容。
func globalAgentsTemplate() string {
	return `# 全局约定

> 这个文件由 qg 自动创建，还没有内容。
> 补充之前，遇到跨项目的习惯问题（回复语言、提交方式、验证命令）直接问我。

<!-- 建议从这里开始：
- 回复用中文，代码注释与提交信息用英文
- 不自动 commit / push，除非当轮明确要求
- 交付前跑通项目自己的验证命令
-->
`
}
