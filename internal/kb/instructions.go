package kb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NoteContent 描述工作树专属说明文件的内容。
type NoteContent struct {
	// Branch 是当前分支名。
	Branch string
	// Start 是分支起点，为空表示基于当时 HEAD。
	Start string
	// Note 是用户填写的需求说明，可为空。
	Note string
	// Created 是创建时间。
	Created time.Time
}

// WriteLocalNote 在特性分支的工作树里写 AGENTS.local.md。
// 主工作树不写；文件已存在时保留原文并返回 false，绝不覆盖。
func WriteLocalNote(layout Layout, content NoteContent) (bool, error) {
	if layout.IsMain {
		return false, nil
	}

	path := filepath.Join(layout.Worktree, LocalNoteFile)
	if _, err := os.Lstat(path); err == nil {
		return false, nil
	}

	start := content.Start
	if start == "" {
		start = "当前 HEAD"
	}
	note := strings.TrimSpace(content.Note)
	if note == "" {
		note = "<!-- 一句话写清楚这个分支要做什么，写完删掉这行注释。 -->"
	}

	text := fmt.Sprintf(`# 本工作树

- 分支：%s
- 起点：%s
- 创建：%s

## 本需求

%s

## 本分支约束

<!-- 例如：只改某个目录，不要动别的模块。没有就删掉这一节。 -->

## 知识库

项目知识在 .dsh/ 下，按根目录 AGENTS.md 的索引按需读取。
本分支的写入只会落进分支知识目录，不会污染基座。
`, content.Branch, start, content.Created.Format("2006-01-02"), note)

	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return false, fmt.Errorf("写入 %s 失败: %w", LocalNoteFile, err)
	}

	return true, nil
}

// SyncWorktreeInstructions 刷新工作树根 AGENTS.md。
// 主工作树直接链接基座；特性分支生成合并文件，保证 Codex、DSH、Qoder
// 都能自动读到同一份共享基座与分支说明。
func SyncWorktreeInstructions(layout *Layout) error {
	if layout.IsMain {
		return linkPath(filepath.Join(layout.Base, "AGENTS.md"), layout.InstructionsPath(), layout)
	}
	return writeManagedInstructions(layout)
}

func writeManagedInstructions(layout *Layout) error {
	dst := layout.InstructionsPath()
	baseAgents := filepath.Join(layout.Base, "AGENTS.md")

	info, err := os.Lstat(dst)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return fmt.Errorf("检查 AGENTS.md 失败: %w", err)
	case info.Mode()&os.ModeSymlink != 0:
		target, err := resolvedLinkTarget(dst)
		if err != nil {
			return fmt.Errorf("读取 AGENTS.md 链接失败: %w", err)
		}
		if filepath.Clean(target) != filepath.Clean(baseAgents) {
			layout.Warnings = append(layout.Warnings, fmt.Sprintf("%s 链接到别处（%s），未覆盖", dst, target))
			return nil
		}
	case info.Mode().IsRegular():
		data, err := os.ReadFile(dst)
		if err != nil {
			return fmt.Errorf("读取 AGENTS.md 失败: %w", err)
		}
		if !strings.Contains(string(data), instructionsMarker) {
			layout.Warnings = append(layout.Warnings, fmt.Sprintf("%s 已存在且不是 qg 生成文件，未覆盖", dst))
			return nil
		}
	default:
		layout.Warnings = append(layout.Warnings, fmt.Sprintf("%s 已存在且不是普通文件，未覆盖", dst))
		return nil
	}

	content, err := buildManagedInstructions(layout)
	if err != nil {
		return err
	}
	if pathExists(dst) {
		if err := os.Remove(dst); err != nil {
			return fmt.Errorf("移除旧 AGENTS.md 失败: %w", err)
		}
	}
	if err := writeFileAtomic(dst, []byte(content), 0o644); err != nil {
		return fmt.Errorf("写入工作树 AGENTS.md 失败: %w", err)
	}
	return nil
}

func buildManagedInstructions(layout *Layout) (string, error) {
	baseAgents := filepath.Join(layout.Base, "AGENTS.md")
	baseContent, err := os.ReadFile(baseAgents)
	if err != nil {
		return "", fmt.Errorf("读取共享 AGENTS.md 失败: %w", err)
	}

	var b strings.Builder
	b.WriteString(instructionsMarker)
	b.WriteString("\n<!-- 由 qg 从 _ai-kb 自动生成，请勿直接编辑此文件。 -->\n")
	b.WriteString("<!-- 修改共享基座后运行 `qg kb attach`；本工作树说明写在 AGENTS.local.md。 -->\n\n")
	b.WriteString("<!-- source: ")
	b.WriteString(baseAgents)
	b.WriteString(" -->\n")
	b.WriteString(strings.TrimSpace(string(baseContent)))
	b.WriteString("\n")

	localPath := filepath.Join(layout.Worktree, LocalNoteFile)
	localContent, err := os.ReadFile(localPath)
	switch {
	case err == nil:
		if text := strings.TrimSpace(string(localContent)); text != "" {
			b.WriteString("\n\n<!-- source: ")
			b.WriteString(localPath)
			b.WriteString(" -->\n")
			b.WriteString(text)
			b.WriteString("\n")
		}
	case errors.Is(err, os.ErrNotExist):
	default:
		return "", fmt.Errorf("读取 %s 失败: %w", LocalNoteFile, err)
	}
	return b.String(), nil
}

func resolvedLinkTarget(path string) (string, error) {
	target, err := os.Readlink(path)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(path), target)
	}
	return filepath.Clean(target), nil
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)

	if err := file.Chmod(mode); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(temp, path)
}
