package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/z/cmt/pkg/config"
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
	Messages  []Message `json:"messages"`
}

type AnthropicResponse struct {
	Content []struct {
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
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) Chat(systemPrompt, userPrompt string) (string, error) {
	// 判断是否是 Anthropic API
	if c.isAnthropicAPI() {
		return c.chatAnthropic(systemPrompt, userPrompt)
	}
	return c.chatOpenAI(systemPrompt, userPrompt)
}

func (c *Client) isAnthropicAPI() bool {
	return len(c.config.APIBase) > 0 && 
		(contains(c.config.APIBase, "anthropic") || 
		 contains(c.config.APIBase, "minimaxi.com/anthropic"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Anthropic API 调用
func (c *Client) chatAnthropic(systemPrompt, userPrompt string) (string, error) {
	req := AnthropicRequest{
		Model:     c.config.Model,
		MaxTokens: c.config.MaxTokens,
		Messages: []Message{
			{Role: "user", Content: systemPrompt + "\n\n" + userPrompt},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequest("POST", c.config.APIBase+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API 请求失败: %s", string(respBody))
	}

	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return "", err
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("无返回结果")
	}

	return anthropicResp.Content[0].Text, nil
}

// OpenAI 兼容 API 调用
func (c *Client) chatOpenAI(systemPrompt, userPrompt string) (string, error) {
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

	httpReq, err := http.NewRequest("POST", c.config.APIBase+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API 请求失败: %s", string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("无返回结果")
	}

	return chatResp.Choices[0].Message.Content, nil
}
