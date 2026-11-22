package cmt

import "testing"

func TestWTCommandRegistersCodexFlag(t *testing.T) {
	cmd := newWTCommand()
	flag := cmd.Flags().Lookup("codex")
	if flag == nil {
		t.Fatal("qg wt 应注册 --codex 旗标")
	}
	if flag.Shorthand != "c" {
		t.Fatalf("--codex 的短旗标 = %q，期望 c", flag.Shorthand)
	}
}
