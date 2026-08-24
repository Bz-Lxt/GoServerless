package invoker

import (
	"testing"

	"github.com/gogo/goserverless/internal/model"
)

func TestConcurrencyGate(t *testing.T) {
	inv := &Invoker{inflight: map[string]int{}}
	fn := &model.Function{Name: "n", MaxConcurrency: 1}
	if err := inv.enter(fn); err != nil {
		t.Fatal(err)
	}
	if err := inv.enter(fn); err == nil {
		t.Fatal("expected concurrency error")
	}
	inv.leave("n")
	if err := inv.enter(fn); err != nil {
		t.Fatal(err)
	}
}
