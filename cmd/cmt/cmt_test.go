package cmt

import (
	"strings"
	"testing"
)

func TestReadMultiline(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"空输入", "", ""},
		{"直接回车", "\n", ""},
		{"单行", "feat: 支持新功能\n", "feat: 支持新功能"},
		{"多行", "feat: 支持新功能\n补充说明\n\n", "feat: 支持新功能\n补充说明"},
		{"空白行截断", "  first  \n   \nsecond\n", "first"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readMultiline(strings.NewReader(tc.in))
			if err != nil {
				t.Fatalf("readMultiline() error = %v", err)
			}
			if got != tc.want {
				t.Errorf("readMultiline() = %q, want %q", got, tc.want)
			}
		})
	}
}
