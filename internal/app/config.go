package app

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config 只保存非敏感运行参数。bot_token、模型 API Key 等密钥不保存在这里，
// 而是放在本地凭据存储中。
type Config struct {
	Server ServerConfig `json:"server"`
	ILink  ILinkConfig  `json:"ilink"`
	Model  ModelConfig  `json:"model"`
}

// ServerConfig 配置仅供本机访问的 Gin HTTP 服务。
type ServerConfig struct {
	// Host 必须保持为回环地址；服务只供 GUI 使用。
	Host string `json:"host"`
	Port int    `json:"port"`
}

// ILinkConfig 保存非敏感的 iLink 客户端参数。bot_token 和扫码后得到的账号
// 专属 baseurl 保存在存储中，不放在这里。
type ILinkConfig struct {
	BaseURL        string `json:"base_url"`
	AppID          string `json:"app_id"`
	ClientVersion  uint32 `json:"client_version"`
	BotAgent       string `json:"bot_agent"`
	RequestTimeout Millis `json:"request_timeout_ms"`
}

// ModelConfig 保存非敏感模型参数。api_key 不写入配置，只从本地凭据存储读取。
type ModelConfig struct {
	// BaseURL 不得包含 /v1/chat/completions。
	BaseURL string `json:"base_url"`
	Name    string `json:"name"`
	// SystemPrompt 定义 AI 角色。为空时回退到通用默认提示词。
	SystemPrompt string `json:"system_prompt"`
	// APIKey 用于模型调用鉴权。它是敏感信息，绝不记录或返回。
	APIKey         string `json:"api_key"`
	RequestTimeout Millis `json:"request_timeout_ms"`
}

// Millis 是以普通整数形式序列化为 JSON 的毫秒时长。
type Millis time.Duration

func (m Millis) Duration() time.Duration { return time.Duration(m) }

func (m Millis) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(time.Duration(m) / time.Millisecond))
}

func (m *Millis) UnmarshalJSON(data []byte) error {
	var ms int64
	if err := json.Unmarshal(data, &ms); err != nil {
		return err
	}
	*m = Millis(time.Duration(ms) * time.Millisecond)
	return nil
}

// LoadConfig 从指定路径读取并校验配置文件。
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Server.Host == "" {
		c.Server.Host = "127.0.0.1"
	}
	if !isLoopback(c.Server.Host) {
		return fmt.Errorf("server.host must be loopback, got %q", c.Server.Host)
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port out of range: %d", c.Server.Port)
	}
	if c.ILink.BaseURL == "" {
		return fmt.Errorf("ilink.base_url is required")
	}
	if c.ILink.AppID == "" {
		return fmt.Errorf("ilink.app_id is required")
	}
	if c.Model.BaseURL == "" {
		return fmt.Errorf("model.base_url is required")
	}
	if c.Model.Name == "" {
		return fmt.Errorf("model.name is required")
	}
	return nil
}

func isLoopback(host string) bool {
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}
