package connector

import (
	"context"
	"sync"

	"github.com/Altergom/weixin-ai-api/internal/ilink"
)

// MessageQueue 按接收顺序保存微信入站消息，供本地 API 长轮询读取。
type MessageQueue struct {
	mu     sync.Mutex
	items  []ilink.TextMessage
	notify chan struct{}
}

func NewMessageQueue() *MessageQueue {
	return &MessageQueue{notify: make(chan struct{}, 1)}
}

func (q *MessageQueue) Enqueue(message ilink.TextMessage) {
	q.mu.Lock()
	q.items = append(q.items, message)
	q.mu.Unlock()
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

func (q *MessageQueue) Next(ctx context.Context) (ilink.TextMessage, error) {
	for {
		q.mu.Lock()
		if len(q.items) > 0 {
			message := q.items[0]
			q.items = q.items[1:]
			q.mu.Unlock()
			return message, nil
		}
		q.mu.Unlock()
		select {
		case <-ctx.Done():
			return ilink.TextMessage{}, ctx.Err()
		case <-q.notify:
		}
	}
}
