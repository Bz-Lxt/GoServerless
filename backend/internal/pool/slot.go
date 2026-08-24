package pool

import (
	"sync"
	"time"

	"github.com/gogo/goserverless/internal/model"
)

type SlotState string

const (
	SlotWarm   SlotState = "warm"
	SlotBusy   SlotState = "busy"
	SlotDead   SlotState = "dead"
	SlotBooting SlotState = "booting"
)

type Slot struct {
	ID         string
	Runtime    model.RuntimeName
	Container  string
	SocketPath string
	State      SlotState
	LastUsed   time.Time
	CreatedAt  time.Time
	mu         sync.Mutex
}

func (s *Slot) Touch() {
	s.mu.Lock()
	s.LastUsed = time.Now()
	s.mu.Unlock()
}

func (s *Slot) IdleFor() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Since(s.LastUsed)
}

func (s *Slot) Mark(st SlotState) {
	s.mu.Lock()
	s.State = st
	s.mu.Unlock()
}

func (s *Slot) Snapshot() SlotState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.State
}
