package agent

import (
	"context"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/bus"
	"github.com/JustSebNL/Spiderweb/pkg/config"
	"github.com/JustSebNL/Spiderweb/pkg/constants"
)

type inboundKey struct {
	channel  string
	chatID   string
	senderID string
}

type pendingBatch struct {
	msgs       []bus.InboundMessage
	contentLen int
	flushAt    time.Time
}

type inboundValve struct {
	coalesceWindow time.Duration
	maxBatchMsgs   int
	maxBatchChars  int
	dedupeWindow   time.Duration
	followUpWindow time.Duration
	followUpItems  int
	followUpOn     bool
	enabled        bool
}

func newInboundValve(cfg config.IntakeConfig) *inboundValve {
	v := &inboundValve{
		coalesceWindow: time.Duration(cfg.CoalesceWindowMs) * time.Millisecond,
		maxBatchMsgs:   cfg.MaxBatchMessages,
		maxBatchChars:  cfg.MaxBatchChars,
		dedupeWindow:   time.Duration(cfg.DedupeWindowSeconds) * time.Second,
		followUpWindow: time.Duration(cfg.FollowUpWindowSec) * time.Second,
		followUpItems:  cfg.FollowUpMaxItems,
		followUpOn:     cfg.FollowUpEnabled,
		enabled:        cfg.Enabled,
	}
	if v.coalesceWindow < 0 {
		v.coalesceWindow = 0
	}
	if v.maxBatchMsgs <= 0 {
		v.maxBatchMsgs = 1
	}
	if v.maxBatchChars < 0 {
		v.maxBatchChars = 0
	}
	if v.dedupeWindow < 0 {
		v.dedupeWindow = 0
	}
	if v.followUpWindow < 0 {
		v.followUpWindow = 0
	}
	if v.followUpItems <= 0 {
		v.followUpItems = 5
	}
	return v
}

func (v *inboundValve) Start(ctx context.Context, mb *bus.MessageBus) <-chan bus.InboundMessage {
	out := make(chan bus.InboundMessage, 100)
	if v == nil || !v.enabled {
		close(out)
		return out
	}

	in := make(chan bus.InboundMessage, 100)

	go func() {
		defer close(in)
		for {
			msg, ok := mb.ConsumeInbound(ctx)
			if !ok {
				return
			}
			select {
			case in <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer close(out)
		v.run(ctx, in, out)
	}()

	return out
}

func (v *inboundValve) run(ctx context.Context, in <-chan bus.InboundMessage, out chan<- bus.InboundMessage) {
	pending := make(map[inboundKey]*pendingBatch)
	seen := make(map[inboundKey]map[uint64]time.Time)
	recent := make(map[inboundKey][]recentItem)

	tickEvery := 50 * time.Millisecond
	if v.coalesceWindow > 0 && v.coalesceWindow/2 < tickEvery {
		tickEvery = v.coalesceWindow / 2
	}
	if tickEvery < 10*time.Millisecond {
		tickEvery = 10 * time.Millisecond
	}

	ticker := time.NewTicker(tickEvery)
	defer ticker.Stop()

	lastPrune := time.Now()

	flushAll := func() {
		for key, batch := range pending {
			outMsg := buildBatchedMessage(key, batch.msgs)
			select {
			case out <- outMsg:
			case <-ctx.Done():
				return
			}
			delete(pending, key)
		}
	}

	for {
		select {
		case <-ctx.Done():
			flushAll()
			return
		case msg, ok := <-in:
			if !ok {
				flushAll()
				return
			}
			now := time.Now()
			key := inboundKey{channel: msg.Channel, chatID: msg.ChatID, senderID: msg.SenderID}

			hash := uint64(0)
			if v.dedupeWindow > 0 || (v.followUpOn && v.followUpWindow > 0) {
				hash = inboundHash(msg)
			}

			if v.followUpOn && v.followUpWindow > 0 {
				suspicion, refHash := followUpSuspicion(msg, now, recent[key], v.followUpWindow)
				if suspicion != constants.FollowUpNone {
					if msg.Metadata == nil {
						msg.Metadata = make(map[string]string)
					}
					msg.Metadata["follow_up"] = strconv.Itoa(int(suspicion))
					if refHash != 0 {
						msg.Metadata["follow_up_to"] = fmt.Sprintf("%x", refHash)
					}
				}
				recent[key] = appendRecent(recent[key], recentItem{hash: hash, at: now}, v.followUpItems)
			}

			if v.dedupeWindow > 0 {
				if _, exists := seen[key]; !exists {
					seen[key] = make(map[uint64]time.Time)
				}
				if t, hit := seen[key][hash]; hit && now.Sub(t) <= v.dedupeWindow {
					continue
				}
				seen[key][hash] = now
			}

			if v.coalesceWindow == 0 && v.maxBatchMsgs <= 1 && v.maxBatchChars == 0 {
				select {
				case out <- msg:
				case <-ctx.Done():
				}
				continue
			}

			batch, ok := pending[key]
			if !ok {
				pending[key] = &pendingBatch{
					msgs:       []bus.InboundMessage{msg},
					contentLen: len(msg.Content),
					flushAt:    now.Add(v.coalesceWindow),
				}
				continue
			}

			if v.maxBatchMsgs > 0 && len(batch.msgs) >= v.maxBatchMsgs {
				outMsg := buildBatchedMessage(key, batch.msgs)
				select {
				case out <- outMsg:
				case <-ctx.Done():
					return
				}
				pending[key] = &pendingBatch{
					msgs:       []bus.InboundMessage{msg},
					contentLen: len(msg.Content),
					flushAt:    now.Add(v.coalesceWindow),
				}
				continue
			}

			sepLen := 2
			nextLen := batch.contentLen + sepLen + len(msg.Content)
			if v.maxBatchChars > 0 && nextLen > v.maxBatchChars {
				outMsg := buildBatchedMessage(key, batch.msgs)
				select {
				case out <- outMsg:
				case <-ctx.Done():
					return
				}
				pending[key] = &pendingBatch{
					msgs:       []bus.InboundMessage{msg},
					contentLen: len(msg.Content),
					flushAt:    now.Add(v.coalesceWindow),
				}
				continue
			}

			batch.msgs = append(batch.msgs, msg)
			batch.contentLen = nextLen
			batch.flushAt = now.Add(v.coalesceWindow)

		case <-ticker.C:
			now := time.Now()
			for key, batch := range pending {
				if now.Before(batch.flushAt) {
					continue
				}
				outMsg := buildBatchedMessage(key, batch.msgs)
				select {
				case out <- outMsg:
				case <-ctx.Done():
					return
				}
				delete(pending, key)
			}

			if v.dedupeWindow > 0 && now.Sub(lastPrune) >= time.Second {
				cutoff := now.Add(-v.dedupeWindow)
				for key, m := range seen {
					for h, t := range m {
						if t.Before(cutoff) {
							delete(m, h)
						}
					}
					if len(m) == 0 {
						delete(seen, key)
					}
				}
				lastPrune = now
			}
		}
	}
}

func buildBatchedMessage(key inboundKey, msgs []bus.InboundMessage) bus.InboundMessage {
	if len(msgs) == 0 {
		return bus.InboundMessage{}
	}
	if len(msgs) == 1 {
		return msgs[0]
	}

	first := msgs[0]
	var b strings.Builder
	estimated := 0
	for i := range msgs {
		estimated += len(msgs[i].Content) + 2
	}
	b.Grow(estimated)
	for i := range msgs {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(msgs[i].Content)
	}

	out := first
	out.Channel = key.channel
	out.ChatID = key.chatID
	out.SenderID = key.senderID
	out.Content = b.String()

	if len(out.Media) > 0 {
		out.Media = append([]string{}, out.Media...)
	}
	for i := 1; i < len(msgs); i++ {
		if len(msgs[i].Media) > 0 {
			out.Media = append(out.Media, msgs[i].Media...)
		}
	}

	if out.Metadata == nil {
		out.Metadata = make(map[string]string)
	} else {
		copied := make(map[string]string, len(out.Metadata)+2)
		for k, v := range out.Metadata {
			copied[k] = v
		}
		out.Metadata = copied
	}
	out.Metadata["intake_batched"] = "true"
	out.Metadata["intake_batch_count"] = fmt.Sprintf("%d", len(msgs))

	maxFollow := 0
	ref := ""
	for i := range msgs {
		if msgs[i].Metadata == nil {
			continue
		}
		if v, ok := msgs[i].Metadata["follow_up"]; ok {
			if n, err := strconv.Atoi(v); err == nil && n > maxFollow {
				maxFollow = n
				ref = msgs[i].Metadata["follow_up_to"]
			}
		}
	}
	if maxFollow > 0 {
		out.Metadata["follow_up"] = strconv.Itoa(maxFollow)
		if ref != "" {
			out.Metadata["follow_up_to"] = ref
		}
	}

	return out
}

type recentItem struct {
	hash uint64
	at   time.Time
}

func appendRecent(items []recentItem, it recentItem, max int) []recentItem {
	if max <= 0 {
		return items
	}
	items = append(items, it)
	if len(items) <= max {
		return items
	}
	return items[len(items)-max:]
}

func followUpSuspicion(
	msg bus.InboundMessage,
	now time.Time,
	recent []recentItem,
	window time.Duration,
) (constants.FollowUpSuspicion, uint64) {
	if window <= 0 || len(recent) == 0 {
		return constants.FollowUpNone, 0
	}
	last := recent[len(recent)-1]
	if last.hash == 0 {
		return constants.FollowUpNone, 0
	}
	if now.Sub(last.at) > window {
		return constants.FollowUpNone, 0
	}

	text := strings.TrimSpace(msg.Content)
	if text == "" {
		return constants.FollowUpNone, 0
	}

	score := 0
	if len(text) <= 280 {
		score = 1
	}
	if startsWithConnector(text) {
		score++
	}
	if len(text) <= 100 {
		score++
	}
	if score <= 0 {
		return constants.FollowUpNone, 0
	}
	if score > 3 {
		score = 3
	}
	return constants.FollowUpSuspicion(score), last.hash
}

func startsWithConnector(text string) bool {
	s := strings.ToLower(strings.TrimSpace(text))
	if strings.HasPrefix(s, "re:") || strings.HasPrefix(s, "fw:") || strings.HasPrefix(s, "fwd:") {
		return true
	}
	prefixes := []string{
		"also", "and", "btw", "fyi", "update", "more", "another", "quick", "ps", "p.s.",
		"one more", "additionally", "follow up", "follow-up", "continuing",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func inboundHash(msg bus.InboundMessage) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(msg.Content))
	_, _ = h.Write([]byte{0})
	for i := range msg.Media {
		_, _ = h.Write([]byte(msg.Media[i]))
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64()
}
