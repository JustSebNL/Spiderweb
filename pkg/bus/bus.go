package bus

import (
	"context"
	"sync"
	"time"
)

type MessageBus struct {
	inboundHigh chan InboundMessage
	inboundLow  chan InboundMessage
	outbound    chan OutboundMessage
	handlers    map[string]MessageHandler
	usage       *inboundUsageTracker
	closed      bool
	mu          sync.RWMutex
}

func NewMessageBus() *MessageBus {
	return NewMessageBusWithCapacities(100, 100)
}

func NewMessageBusWithCapacities(highCapacity, lowCapacity int) *MessageBus {
	if highCapacity <= 0 {
		highCapacity = 100
	}
	if lowCapacity <= 0 {
		lowCapacity = 100
	}
	return &MessageBus{
		inboundHigh: make(chan InboundMessage, highCapacity),
		inboundLow:  make(chan InboundMessage, lowCapacity),
		outbound:    make(chan OutboundMessage, 100),
		handlers:    make(map[string]MessageHandler),
		usage:       newInboundUsageTracker(),
	}
}

func (mb *MessageBus) PublishInbound(msg InboundMessage) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	if mb.closed {
		return
	}
	if isInterrupt(msg) {
		mb.inboundHigh <- msg
		if mb.usage != nil {
			mb.usage.record(time.Now(), msg)
		}
		return
	}
	mb.inboundLow <- msg
	if mb.usage != nil {
		mb.usage.record(time.Now(), msg)
	}
}

func (mb *MessageBus) TryPublishInbound(msg InboundMessage) bool {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	if mb.closed {
		return false
	}

	if isInterrupt(msg) {
		select {
		case mb.inboundHigh <- msg:
			if mb.usage != nil {
				mb.usage.record(time.Now(), msg)
			}
			return true
		default:
			return false
		}
	}
	select {
	case mb.inboundLow <- msg:
		if mb.usage != nil {
			mb.usage.record(time.Now(), msg)
		}
		return true
	default:
		return false
	}
}

func (mb *MessageBus) InboundStats() (length int, capacity int, closed bool) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	return len(mb.inboundHigh) + len(mb.inboundLow), cap(mb.inboundHigh) + cap(mb.inboundLow), mb.closed
}

func (mb *MessageBus) InboundStatsByQueue() (highLen int, highCap int, lowLen int, lowCap int, closed bool) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	return len(mb.inboundHigh), cap(mb.inboundHigh), len(mb.inboundLow), cap(mb.inboundLow), mb.closed
}

func (mb *MessageBus) ConsumeInbound(ctx context.Context) (InboundMessage, bool) {
	select {
	case msg := <-mb.inboundHigh:
		return msg, true
	default:
	}
	select {
	case msg := <-mb.inboundHigh:
		return msg, true
	case msg := <-mb.inboundLow:
		return msg, true
	case <-ctx.Done():
		return InboundMessage{}, false
	}
}

func (mb *MessageBus) PublishOutbound(msg OutboundMessage) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	if mb.closed {
		return
	}
	mb.outbound <- msg
}

func (mb *MessageBus) SubscribeOutbound(ctx context.Context) (OutboundMessage, bool) {
	select {
	case msg := <-mb.outbound:
		return msg, true
	case <-ctx.Done():
		return OutboundMessage{}, false
	}
}

func (mb *MessageBus) RegisterHandler(channel string, handler MessageHandler) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.handlers[channel] = handler
}

func (mb *MessageBus) GetHandler(channel string) (MessageHandler, bool) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	handler, ok := mb.handlers[channel]
	return handler, ok
}

func (mb *MessageBus) Close() {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	if mb.closed {
		return
	}
	mb.closed = true
	close(mb.inboundHigh)
	close(mb.inboundLow)
	close(mb.outbound)
}

func isInterrupt(msg InboundMessage) bool {
	if msg.Metadata == nil {
		return false
	}
	if msg.Metadata["valve"] == "interrupt" {
		return true
	}
	if msg.Metadata["priority"] == "high" {
		return true
	}
	return false
}

func (mb *MessageBus) InboundUsageSnapshot(windowDays int) InboundUsageSnapshot {
	now := time.Now()
	mb.mu.RLock()
	u := mb.usage
	mb.mu.RUnlock()
	if u == nil {
		return InboundUsageSnapshot{GeneratedAt: now.Format(time.RFC3339), WindowDays: windowDays}
	}
	return u.snapshot(now, windowDays)
}
