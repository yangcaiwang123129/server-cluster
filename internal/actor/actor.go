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

func (this *Actor[T]) Run() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[actor] panic recovered: %v", r)
		}
		this.handler.OnStop()
		close(this.done)
	}()

	ticker := time.NewTicker(this.tickRate)
	defer ticker.Stop()

	lastTick := time.Now()
	for {
		select {
		case <-this.ctx.Done():
			return
		case msg := <-this.msgQueue:
			this.handler.HandleMessage(msg)
		case now := <-ticker.C:
			dt := now.Sub(lastTick).Seconds()
			lastTick = now
			if len(ticker.C) > 0 {
				continue // 跳过积压 tick，保持节奏
			}
			this.handler.Tick(dt)
		}
	}
}

func (this *Actor[T]) Send(ctx context.Context, msg Message) bool {
	select {
	case this.msgQueue <- msg:
		return true
	case <-this.ctx.Done():
		return false
	case <-ctx.Done():
		return false
	}
}

// Call 发送消息并等待回包；reply 为接收方向通道（由调用方创建并传入消息）。
func (this *Actor[T]) Call(ctx context.Context, msg Message, reply <-chan interface{}) bool {
	if !this.Send(ctx, msg) {
		return false
	}
	select {
	case <-this.ctx.Done():
		return false
	case <-ctx.Done():
		return false
	case <-reply:
		return true
	}
}

func (this *Actor[T]) Stop()                 { this.cancel() }
func (this *Actor[T]) Done() <-chan struct{} { return this.done }
