package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zfslalala/cmt/pkg/config"
)

const (
	maxResponseBodyBytes = 1 << 20
	maxErrorDetailBytes  = 4 << 10
)

type Client struct {
	config *config.Config
	client *http.Client
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Anthropic API 请求格式
type AnthropicRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
}

type AnthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// OpenAI API 请求格式（兼容其他服务）
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		config: cfg,
		client: &http.Client{
			Timeout: 120 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				// 不跟随重定向，避免把 API Key 转发给未配置的目标地址。
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *Client) Chat(systemPrompt, userPrompt string) (string, error) {
	return c.ChatContext(context.Background(), systemPrompt, userPrompt)
}

func (c *Client) ChatContext(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// 判断是否是 Anthropic API
	if c.isAnthropicAPI() {
		return c.chatAnthropic(ctx, systemPrompt, userPrompt)
	}
	return c.chatOpenAI(ctx, systemPrompt, userPrompt)
}

func (c *Client) isAnthropicAPI() bool {
	return strings.Contains(c.config.APIBase, "anthropic")
}

// doRequest 发送 POST 请求,非 2xx 时返回含响应体摘要的错误
func (c *Client) doRequest(ctx context.Context, url string, headers map[string]string, body []byte) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("API 请求失败(%d): %s", resp.StatusCode, responseExcerpt(respBody))
	}
	if len(respBody) > maxResponseBodyBytes {
		return nil, fmt.Errorf("API 响应超过 %d KiB，已中止读取", maxResponseBodyBytes/1024)
	}
	return respBody, nil
}

func responseExcerpt(body []byte) string {
	truncated := len(body) > maxErrorDetailBytes
	if truncated {
		body = body[:maxErrorDetailBytes]
	}
	text := strings.TrimSpace(string(body))
	if truncated {
		text += "... (响应体已截断)"
	}
	return text
}

// anthropicMessagesURL 拼接 Anthropic messages 端点,兼容末尾带 / 或 /v1 的写法
func (c *Client) anthropicMessagesURL() string {
	base := strings.TrimSuffix(c.config.APIBase, "/")
	base = strings.TrimSuffix(base, "/v1")
	return base + "/v1/messages"
}

// openaiChatURL 拼接 OpenAI 兼容 chat 端点
func (c *Client) openaiChatURL() string {
	return strings.TrimSuffix(c.config.APIBase, "/") + "/chat/completions"
}

// chatAnthropic 调用 Anthropic API
func (c *Client) chatAnthropic(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	req := AnthropicRequest{
		Model:     c.config.Model,
		MaxTokens: c.config.MaxTokens,
		System:    systemPrompt,
		Messages:  []Message{{Role: "user", Content: userPrompt}},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	respBody, err := c.doRequest(ctx, c.anthropicMessagesURL(), map[string]string{
		"x-api-key":         c.config.APIKey,
		"anthropic-version": "2023-06-01",
	}, body)
	if err != nil {
		return "", err
	}

	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return "", err
	}

	// 遍历查找文本块(兼容含 thinking 块的响应)
	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			if text := strings.TrimSpace(block.Text); text != "" {
				return text, nil
			}
		}
	}
	return "", fmt.Errorf("API 未返回可用的文本内容")
}

// chatOpenAI 调用 OpenAI 兼容 API
func (c *Client) chatOpenAI(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	req := ChatRequest{
		Model: c.config.Model,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: c.config.Temperature,
		MaxTokens:   c.config.MaxTokens,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	respBody, err := c.doRequest(ctx, c.openaiChatURL(), map[string]string{
		"Authorization": "Bearer " + c.config.APIKey,
	}, body)
	if err != nil {
		return "", err
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", err
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("API 未返回可用结果")
	}
	message := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	if message == "" {
		return "", fmt.Errorf("API 返回了空的 commit message")
	}
	return message, nil
}
