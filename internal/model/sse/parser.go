// sse 包负责解析 OpenAI 兼容的 Server-Sent Events 响应。
package sse

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

var ErrStreamIncomplete = errors.New("model SSE stream ended before [DONE]")

type chatCompletionChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// ParseChatCompletion 读取 OpenAI 兼容的聊天补全流。
// 只有服务端发送 data: [DONE] 后才返回拼接完成的文本。
// 每收到一个非空内容增量，就调用一次回调函数。
func ParseChatCompletion(reader io.Reader, onDelta func(string) error) (string, error) {
	if reader == nil {
		return "", errors.New("model SSE reader is required")
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4*1024), 1024*1024)

	var reply strings.Builder
	var dataLines []string

	flushEvent := func() (done bool, err error) {
		if len(dataLines) == 0 {
			return false, nil
		}
		data := strings.Join(dataLines, "\n")
		dataLines = nil
		if data == "[DONE]" {
			return true, nil
		}

		var chunk chatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return false, fmt.Errorf("decode model SSE event: %w", err)
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content == "" {
				continue
			}
			reply.WriteString(choice.Delta.Content)
			if onDelta != nil {
				if err := onDelta(choice.Delta.Content); err != nil {
					return false, fmt.Errorf("handle model SSE delta: %w", err)
				}
			}
		}
		return false, nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			done, err := flushEvent()
			if err != nil {
				return "", err
			}
			if done {
				return reply.String(), nil
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data:"))
			if len(dataLines) > 0 && strings.HasPrefix(dataLines[len(dataLines)-1], " ") {
				dataLines[len(dataLines)-1] = strings.TrimPrefix(dataLines[len(dataLines)-1], " ")
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read model SSE stream: %w", err)
	}

	done, err := flushEvent()
	if err != nil {
		return "", err
	}
	if done {
		return reply.String(), nil
	}
	return "", ErrStreamIncomplete
}
