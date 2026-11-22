package cmt

import "testing"

func TestRMCommandAcceptsOptionalBranch(t *testing.T) {
	cmd := newRMCommand()
	if err := cmd.Args(cmd, nil); err != nil {
		t.Fatalf("qg rm 应接受零参数: %v", err)
	}
	if err := cmd.Args(cmd, []string{"feat/x"}); err != nil {
		t.Fatalf("qg rm 应接受一个分支参数: %v", err)
	}
	if err := cmd.Args(cmd, []string{"feat/x", "feat/y"}); err == nil {
		t.Fatal("qg rm 不应接受两个分支参数")
	}
}
