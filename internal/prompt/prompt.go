package prompt

import (
	"fmt"
	"strings"

	"github.com/zfslalala/cmt/internal/git"
)

const systemPrompt = `你是一个专业的 Git 提交信息生成助手。

## 规范
遵循 Conventional Commits 规范：
- feat: 新功能
- fix: 修复 bug
- docs: 文档变更
- style: 代码格式（不影响功能）
- refactor: 重构（既不是新功能也不是修复）
- perf: 性能优化
- test: 测试相关
- chore: 构建/工具相关
- ci: CI/CD 相关
- build: 构建系统或外部依赖变更
- revert: 回退之前的提交

## 输出格式

### 精简模式
类型: 描述

示例：
- feat: 添加用户登录功能
- fix: 修复请求超时问题
- docs: 更新 README 安装说明

**注意：不要使用括号，不要包含编号或 ticket 号**

### 详细模式
类型: 描述

详细说明变更的内容和原因...

- 变更点1
- 变更点2

Footer: 相关信息

## 要求
- 默认使用中文
- 直接输出 commit message，不要有其他解释
- 描述要简洁明了，不超过50个字符
- 使用祈使句，如"添加"而不是"添加了"`

func GetSystemPrompt() string {
	return systemPrompt
}

func BuildUserPrompt(changes *git.StagedChanges, verbose bool) string {
	var sb strings.Builder

	// 构建文件摘要
	sb.WriteString("## 变更文件\n")
	for _, file := range changes.Files {
		symbol := git.GetStatusSymbol(file.Status)
		if file.Status == "R" || file.Status == "C" {
			sb.WriteString(fmt.Sprintf("- %s %s → %s\n", symbol, file.OldPath, file.NewPath))
		} else {
			sb.WriteString(fmt.Sprintf("- %s %s\n", symbol, file.NewPath))
		}
	}

	// 添加 diff 内容
	sb.WriteString("\n## 详细变更（diff）\n```\n")
	sb.WriteString(changes.DiffContent)
	sb.WriteString("\n```\n")

	// 添加输出要求
	mode := "精简"
	if verbose {
		mode = "详细"
	}
	sb.WriteString(fmt.Sprintf("\n## 输出要求\n请生成%s模式的 commit message。", mode))

	return sb.String()
}
