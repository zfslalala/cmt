// Package config 定义 qg 的运行时配置值与默认值。
//
// 配置的唯一来源是 qg.hjson（查找顺序见 internal/build.Load：当前工作树根，
// 缺失时回退知识库项目基座）。本包只定义配置值与默认值，
// 不关心配置文件落在哪里、如何解析，也不做校验。
package config

// Config 是一次调用所需的大模型配置。
type Config struct {
	APIKey      string
	APIBase     string
	Model       string
	Temperature float64
	MaxTokens   int
}

// 大模型配置的默认值；qg.hjson 的 cmt 段未配置对应字段时使用。
const (
	DefaultAPIBase = "https://api.openai.com/v1"
	DefaultModel   = "gpt-4o-mini"
)

// Default 返回带默认值的大模型配置。APIKey 没有默认值，由调用方
// 从 qg.hjson 的 cmt.apiKey 提供，缺失时必须显式报错。
func Default() Config {
	return Config{
		APIBase:     DefaultAPIBase,
		Model:       DefaultModel,
		Temperature: 0.7,
		MaxTokens:   1000,
	}
}
