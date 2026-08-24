package logstream

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gogo/goserverless/internal/timeutil"
)

type Event struct {
	Function  string `json:"function"`
	Version   int    `json:"version"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	ColdStart bool   `json:"cold_start"`
	At        string `json:"at"`
}

type subscriber struct {
	ch   chan []byte
	name string
}

type Hub struct {
	mu   sync.Mutex
	subs map[*subscriber]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: map[*subscriber]struct{}{}}
}

func (h *Hub) Publish(fn string, version int, msg string, cold bool) {
	if msg == "" {
		return
	}
	ev := Event{
		Function:  fn,
		Version:   version,
		Level:     "info",
		Message:   msg,
		ColdStart: cold,
		At:        timeutil.Format(timeutil.Now()),
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.subs {
		if s.name != "" && s.name != fn {
			continue
		}
		select {
		case s.ch <- raw:
		default:
		}
	}
}

func (h *Hub) Subscribe(name string) (<-chan []byte, func()) {
	s := &subscriber{ch: make(chan []byte, 64), name: name}
	h.mu.Lock()
	h.subs[s] = struct{}{}
	h.mu.Unlock()
	return s.ch, func() {
		h.mu.Lock()
		delete(h.subs, s)
		h.mu.Unlock()
		close(s.ch)
	}
}

func (h *Hub) KeepAlive(ch chan []byte) {
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			select {
			case ch <- []byte(`{"keepalive":true}`):
			default:
				return
			}
		}
	}()
}
