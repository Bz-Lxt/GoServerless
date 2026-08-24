package limiter

import (
	"testing"
	"time"
)

func TestLimiterWindow(t *testing.T) {
	l := New(2, 50*time.Millisecond)
	if !l.Allow("a") || !l.Allow("a") {
		t.Fatal("first two")
	}
	if l.Allow("a") {
		t.Fatal("third should deny")
	}
	time.Sleep(60 * time.Millisecond)
	if !l.Allow("a") {
		t.Fatal("after window")
	}
}
