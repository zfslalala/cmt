package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zfslalala/cmt/pkg/config"
)

func testClient(baseURL string) *Client {
	cfg := config.Default()
	cfg.APIBase = baseURL
	cfg.APIKey = "test-key"
	cfg.Model = "test-model"
	return NewClient(&cfg)
}

func TestChatOpenAITrimsMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"  feat: add tests  \n"}}]}`)
	}))
	defer server.Close()

	got, err := testClient(server.URL).Chat("system", "user")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if got != "feat: add tests" {
		t.Fatalf("message = %q", got)
	}
}

func TestChatRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(strings.Repeat("x", maxResponseBodyBytes+1)))
	}))
	defer server.Close()

	_, err := testClient(server.URL).Chat("system", "user")
	if err == nil || !strings.Contains(err.Error(), "响应超过") {
		t.Fatalf("期望响应上限错误，实际: %v", err)
	}
}

func TestChatClipsErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(strings.Repeat("e", maxErrorDetailBytes+100)))
	}))
	defer server.Close()

	_, err := testClient(server.URL).Chat("system", "user")
	if err == nil {
		t.Fatal("非 2xx 响应应报错")
	}
	if !strings.Contains(err.Error(), "502") || !strings.Contains(err.Error(), "响应体已截断") {
		t.Fatalf("错误信息应包含状态码与截断提示，实际: %v", err)
	}
}

func TestChatContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := testClient(server.URL).ChatContext(ctx, "system", "user")
	if err == nil {
		t.Fatal("已取消的 context 应返回错误")
	}
}

func TestChatDoesNotFollowRedirects(t *testing.T) {
	var leaked atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		leaked.Store(true)
		fmt.Fprint(w, `{"choices":[{"message":{"content":"redirected"}}]}`)
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()

	if _, err := testClient(redirector.URL).Chat("system", "user"); err == nil {
		t.Fatal("重定向响应应作为非 2xx 错误返回")
	}
	if leaked.Load() {
		t.Fatal("不应跟随重定向并泄露 Authorization")
	}
}
