package git

import "testing"

func TestParseWorktreesPorcelain(t *testing.T) {
	output := `worktree /repo/main
HEAD aaaa
branch refs/heads/main

worktree /repo/feature
HEAD bbbb
branch refs/heads/feat/x
locked user requested

worktree /repo/stale
HEAD cccc
branch refs/heads/feat/stale
prunable gitdir file points to non-existent location
`

	got := parseWorktreesPorcelain(output)
	if len(got) != 3 {
		t.Fatalf("解析到 %d 个工作树，期望 3 个: %+v", len(got), got)
	}
	if got[0].Path != "/repo/main" || got[0].Branch != "main" || got[0].Locked || got[0].Prunable {
		t.Fatalf("主工作树解析错误: %+v", got[0])
	}
	if got[1].Branch != "feat/x" || !got[1].Locked || got[1].LockedReason != "user requested" {
		t.Fatalf("locked 工作树解析错误: %+v", got[1])
	}
	if !got[2].Prunable || got[2].PrunableReason == "" {
		t.Fatalf("prunable 工作树解析错误: %+v", got[2])
	}
}
