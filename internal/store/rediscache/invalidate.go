package rediscache

import (
	"context"
	"encoding/json"
)

// Message 失效广播消息；Fields 为需失效的缓存类别（KindCore/KindAssets/KindProfile）。
type Message struct {
	PlayerID int64    `json:"player_id"`
	Fields   []string `json:"fields"`
}

// PublishInvalidate 广播缓存失效；尽力而为，失败不影响主流程。
func (c *Cache) PublishInvalidate(ctx context.Context, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.rdb.Publish(ctx, c.channel, data).Err()
}

// SubscribeInvalidate 订阅失效广播，后台 goroutine 分发 handler；ctx 取消时退出。
func (c *Cache) SubscribeInvalidate(ctx context.Context, handler func(*Message)) {
	sub := c.rdb.Subscribe(ctx, c.channel)
	go func() {
		defer func() { _ = sub.Close() }()
		ch := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-ch:
				if !ok {
					return
				}
				var msg Message
				if err := json.Unmarshal([]byte(m.Payload), &msg); err == nil {
					handler(&msg)
				}
			}
		}
	}()
}
