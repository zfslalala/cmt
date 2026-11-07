package git

import "testing"

func TestSanitizeBranchName(t *testing.T) {
	cases := map[string]string{
		"feature/login":  "feature-login",
		"feature login":  "feature-login",
		"feature\\login": "feature-login",
		"fix:bug":        "fix-bug",
		"a<b>c|d?e*f\"g": "a-b-c-d-e-f-g",
		"plain":          "plain",
		"release/v1.0":   "release-v1.0",
	}
	for input, want := range cases {
		if got := SanitizeBranchName(input); got != want {
			t.Errorf("SanitizeBranchName(%q) = %q, want %q", input, got, want)
		}
	}
}
