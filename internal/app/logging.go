package app

import (
	"log/slog"
	"os"
	"strings"
)

// sensitiveKeys 列出绝不能出现在日志中的属性键。
// 匹配不区分大小写并采用子串方式，因此 "ilink_bot_token"、
// "model_api_key" 等嵌套字段也会被捕获。
var sensitiveKeys = []string{
	"authorization",
	"bot_token",
	"token",
	"api_key",
	"apikey",
	"secret",
	"password",
	"context_token",
	"qrcode",
	"get_updates_buf",
	"x-wechat-uin",
	"text",
	"content",
	"message",
}

const redacted = "[REDACTED]"

// NewLogger 创建结构化 JSON 日志器，并脱敏敏感属性值。
// level 控制最低日志级别。
func NewLogger(level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: redactAttr,
	})
	return slog.New(handler)
}

func redactAttr(_ []string, a slog.Attr) slog.Attr {
	if isSensitiveKey(a.Key) {
		a.Value = slog.StringValue(redacted)
	}
	return a
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(k, s) {
			return true
		}
	}
	return false
}
