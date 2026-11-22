package kb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// journalTemplate 是分支日志的初始内容，CleanupBranchKnowledge 也用同一份文本
// 判断分支知识是否从未被写入过。
func journalTemplate(branch string) string {
	return fmt.Sprintf("# 分支知识日志\n\n分支：%s\n\n<!-- 按时间倒序，最新在上。合入基座后可清空。 -->\n", branch)
}

// CleanupBranchKnowledge 删除一个已结束分支的知识目录。
// 只在目录从未被写入过时删除；有任何实际内容时保留并返回 false，避免丢经验。
func CleanupBranchKnowledge(layout Layout) (bool, error) {
	if layout.IsMain || layout.Branch == "" {
		return false, nil
	}

	if _, err := os.Stat(layout.Branch); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("检查分支知识目录失败: %w", err)
	}

	pristine, err := isPristineBranch(layout)
	if err != nil {
		return false, err
	}
	if !pristine {
		return false, nil
	}

	if err := os.RemoveAll(layout.Branch); err != nil {
		return false, fmt.Errorf("删除分支知识目录失败: %w", err)
	}

	return true, nil
}

// isPristineBranch 判断分支知识目录是否仍是刚创建时的空壳：
// 没有技能、没有决策，只有一份未被改动过的 journal 模板。
func isPristineBranch(layout Layout) (bool, error) {
	skills := filepath.Join(layout.Branch, "skills")
	if entries, err := os.ReadDir(skills); err == nil && len(entries) > 0 {
		return false, nil
	}

	decisions := filepath.Join(layout.Branch, "knowledge", "decisions")
	if entries, err := os.ReadDir(decisions); err == nil && len(entries) > 0 {
		return false, nil
	}

	knowledge := filepath.Join(layout.Branch, "knowledge")
	entries, err := os.ReadDir(knowledge)
	if err != nil {
		return false, fmt.Errorf("读取分支知识目录失败: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() != "journal.md" && entry.Name() != "decisions" {
			return false, nil
		}
	}

	journal := filepath.Join(knowledge, "journal.md")
	data, err := os.ReadFile(journal)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("读取分支日志失败: %w", err)
	}

	return string(data) == journalTemplate(layout.BranchName), nil
}
