package actor

import (
	"context"
	"log"
	"time"
)

type Message interface{ MsgType() string }

type Handler interface {
	HandleMessage(msg Message)
	Tick(dt float64)
	OnStop()
}

type Actor[T Handler] struct {
	handler  T
	msgQueue chan Message
	tickRate time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
}

func New[T Handler](handler T, tickRate time.Duration, queueSize int) *Actor[T] {
	ctx, cancel := context.WithCancel(context.Background())
	if queueSize <= 0 {
		queueSize = 512
	}
	return &Actor[T]{
		handler:  handler,
		msgQueue: make(chan Message, queueSize),
		tickRate: tickRate,
		ctx:      ctx,
		cancel:   cancel,
		done:     make(chan struct{}),
	}
}

func (a *Actor[T]) Run() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[actor] panic recovered: %v", r)
		}
		a.handler.OnStop()
		close(a.done)
	}()

	ticker := time.NewTicker(a.tickRate)
	defer ticker.Stop()

	lastTick := time.Now()
	for {
		select {
		case <-a.ctx.Done():
			return
		case msg := <-a.msgQueue:
			a.handler.HandleMessage(msg)
		case now := <-ticker.C:
			dt := now.Sub(lastTick).Seconds()
			lastTick = now
			if len(ticker.C) > 0 {
				continue // 跳过积压 tick，保持节奏
			}
			a.handler.Tick(dt)
		}
	}
}

func (a *Actor[T]) Send(ctx context.Context, msg Message) bool {
	select {
	case a.msgQueue <- msg:
		return true
	case <-a.ctx.Done():
		return false
	case <-ctx.Done():
		return false
	}
}

// Call 发送消息并等待回包；reply 为接收方向通道（由调用方创建并传入消息）。
func (a *Actor[T]) Call(ctx context.Context, msg Message, reply <-chan interface{}) bool {
	if !a.Send(ctx, msg) {
		return false
	}
	select {
	case <-a.ctx.Done():
		return false
	case <-ctx.Done():
		return false
	case <-reply:
		return true
	}
}

func (a *Actor[T]) Stop()                 { a.cancel() }
func (a *Actor[T]) Done() <-chan struct{} { return a.done }
