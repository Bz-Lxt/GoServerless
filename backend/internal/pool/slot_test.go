package pool

import (
	"testing"
	"time"

	"github.com/gogo/goserverless/internal/model"
)

func TestSlotStateMachine(t *testing.T) {
	s := &Slot{ID: "s1", Runtime: model.RuntimeGo, State: SlotWarm, LastUsed: time.Now()}
	if s.Snapshot() != SlotWarm {
		t.Fatal("warm")
	}
	s.Mark(SlotBusy)
	if s.Snapshot() != SlotBusy {
		t.Fatal("busy")
	}
	s.LastUsed = time.Now().Add(-6 * time.Minute)
	if s.IdleFor() < 5*time.Minute {
		t.Fatal("idle")
	}
}

func TestAcquireResultColdFlag(t *testing.T) {
	r := AcquireResult{ColdStart: true, Wakeup: 12 * time.Millisecond}
	if !r.ColdStart {
		t.Fatal("cold")
	}
	if r.Wakeup < time.Millisecond {
		t.Fatal("wakeup")
	}
}
