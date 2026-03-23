package bus

import (
	"sort"
	"sync"
	"time"
)

type InboundUsageSnapshot struct {
	GeneratedAt string             `json:"generated_at"`
	Days        []InboundDayStats  `json:"days"`
	Total       InboundTotals      `json:"total"`
	WindowDays  int                `json:"window_days"`
}

type InboundTotals struct {
	Messages int `json:"messages"`
	High     int `json:"high"`
	Low      int `json:"low"`
}

type InboundDayStats struct {
	Date      string         `json:"date"`
	Messages  int            `json:"messages"`
	High      int            `json:"high"`
	Low       int            `json:"low"`
	ByHour    [24]int        `json:"by_hour"`
	ByChannel map[string]int `json:"by_channel,omitempty"`
}

type inboundUsageTracker struct {
	mu   sync.Mutex
	days map[string]*InboundDayStats
}

func newInboundUsageTracker() *inboundUsageTracker {
	return &inboundUsageTracker{
		days: make(map[string]*InboundDayStats),
	}
}

func (t *inboundUsageTracker) record(now time.Time, msg InboundMessage) {
	day := now.Format("2006-01-02")
	hour := now.Hour()
	if hour < 0 || hour > 23 {
		hour = 0
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	ds, ok := t.days[day]
	if !ok {
		ds = &InboundDayStats{
			Date:      day,
			ByChannel: make(map[string]int),
		}
		t.days[day] = ds
	}

	ds.Messages++
	ds.ByHour[hour]++

	if isInterrupt(msg) {
		ds.High++
	} else {
		ds.Low++
	}

	if msg.Channel != "" {
		ds.ByChannel[msg.Channel]++
	}

	if len(t.days) > 45 {
		t.pruneLocked(now, 30)
	}
}

func (t *inboundUsageTracker) pruneLocked(now time.Time, keepDays int) {
	if keepDays <= 0 {
		keepDays = 1
	}
	cutoff := now.AddDate(0, 0, -keepDays).Format("2006-01-02")
	for day := range t.days {
		if day < cutoff {
			delete(t.days, day)
		}
	}
}

func (t *inboundUsageTracker) snapshot(now time.Time, windowDays int) InboundUsageSnapshot {
	if windowDays <= 0 {
		windowDays = 1
	}
	if windowDays > 30 {
		windowDays = 30
	}

	t.mu.Lock()
	t.pruneLocked(now, windowDays)

	keys := make([]string, 0, len(t.days))
	for day := range t.days {
		keys = append(keys, day)
	}
	sort.Strings(keys)

	out := InboundUsageSnapshot{
		GeneratedAt: now.Format(time.RFC3339),
		WindowDays:  windowDays,
	}
	out.Days = make([]InboundDayStats, 0, len(keys))
	for _, day := range keys {
		ds := t.days[day]
		if ds == nil {
			continue
		}
		copyDS := InboundDayStats{
			Date:     ds.Date,
			Messages: ds.Messages,
			High:     ds.High,
			Low:      ds.Low,
			ByHour:   ds.ByHour,
		}
		if len(ds.ByChannel) > 0 {
			copyDS.ByChannel = make(map[string]int, len(ds.ByChannel))
			for k, v := range ds.ByChannel {
				copyDS.ByChannel[k] = v
			}
		}
		out.Days = append(out.Days, copyDS)
		out.Total.Messages += copyDS.Messages
		out.Total.High += copyDS.High
		out.Total.Low += copyDS.Low
	}
	t.mu.Unlock()

	return out
}

