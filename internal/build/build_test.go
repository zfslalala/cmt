package build

import "testing"

func TestEnsureFailFlag(t *testing.T) {
	cases := map[string]string{
		"curl https://example.com":           "curl -f https://example.com",
		"  curl -X POST https://example.com": "curl -f -X POST https://example.com",
		"echo curl https://example.com":      "echo curl https://example.com",
		"curl":                               "curl -f",
	}
	for input, want := range cases {
		if got := EnsureFailFlag(input); got != want {
			t.Errorf("EnsureFailFlag(%q) = %q, want %q", input, got, want)
		}
	}
}
