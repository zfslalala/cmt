package git

import (
	"strings"
)

// BranchInfo 分支来源信息（基于 reflog 推断）
type BranchInfo struct {
	Branch    string // 当前分支名
	Source    string // 来源分支，未知则为空
	CreatedAt string // 切出时间，未知则为空
	HasLog    bool   // reflog 是否有记录
}

// GetBranchInfo 获取分支来源信息：来源分支、切出时间。
// 来源与时间通过 reflog 最早一条记录推断（reflog 默认保留 90 天，过期后无法查到）。
func GetBranchInfo(branch string) (*BranchInfo, error) {
	info := &BranchInfo{Branch: branch}

	output, err := runGit("reflog", "show", "--date=iso", branch)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return info, nil
	}
	info.HasLog = true

	// 最后一行是该分支最早的活动，即创建/切出记录
	line := lines[len(lines)-1]

	// 提取日期：selector 形如 "ref@{2026-08-15 11:43:32 +0800}"
	if at := strings.Index(line, "@{"); at >= 0 {
		if end := strings.Index(line[at:], "}"); end >= 0 {
			info.CreatedAt = line[at+2 : at+end]
			line = line[:at] + line[at+end+1:]
		}
	}

	// subject 位于 "<sha> <ref>: " 之后
	if idx := strings.Index(line, ": "); idx >= 0 {
		info.Source = parseSourceFromSubject(line[idx+2:])
	}

	return info, nil
}

// parseSourceFromSubject 从 reflog subject 解析来源分支
func parseSourceFromSubject(subject string) string {
	// 本地新建分支：checkout: moving from X to Y
	if rest, ok := strings.CutPrefix(subject, "checkout: moving from "); ok {
		if idx := strings.Index(rest, " to "); idx > 0 {
			return rest[:idx]
		}
		return rest
	}
	// 其他创建方式：branch: Created from X
	if rest, ok := strings.CutPrefix(subject, "branch: Created from "); ok {
		return rest
	}
	return ""
}
