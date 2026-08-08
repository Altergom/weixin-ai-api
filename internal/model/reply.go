package model

import "strings"

// MaxReplyRunes 限制单条 iLink 出站文本消息长度。模型完整回复先缓冲，
// 再分割成不超过该长度的片段发送，避免长回复被传输层拒绝或截断。
const MaxReplyRunes = 2000

// SplitReply 将完整回复切成不超过 MaxReplyRunes 且按 rune 安全的片段。
// 优先在换行或空白处分割，避免切断单词或多字节字符。空回复不产生片段。
func SplitReply(reply string) []string {
	text := strings.TrimSpace(reply)
	if text == "" {
		return nil
	}

	var chunks []string
	runes := []rune(text)
	for len(runes) > 0 {
		if len(runes) <= MaxReplyRunes {
			chunks = append(chunks, strings.TrimSpace(string(runes)))
			break
		}
		cut := breakPoint(runes, MaxReplyRunes)
		chunk := strings.TrimSpace(string(runes[:cut]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		runes = runes[cut:]
	}
	return chunks
}

// breakPoint 在不超过 limit 的范围内寻找最佳分割位置：优先最后一个换行，
// 其次最后一个空格，找不到分隔符时在 limit 处硬切。
func breakPoint(runes []rune, limit int) int {
	for i := limit; i > 0; i-- {
		if runes[i-1] == '\n' {
			return i
		}
	}
	for i := limit; i > 0; i-- {
		if runes[i-1] == ' ' {
			return i
		}
	}
	return limit
}
