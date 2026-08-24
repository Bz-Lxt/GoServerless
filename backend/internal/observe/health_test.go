package observe

import (
	"context"
	"errors"
	"testing"
)

func TestRegistryAggregates(t *testing.T) {
	r := NewRegistry()
	r.Add("ok", func(context.Context) error { return nil })
	r.Add("bad", func(context.Context) error { return errors.New("down") })
	rep := r.Run(context.Background(), "t")
	if rep.OK {
		t.Fatal("should not be ok")
	}
	if rep.Errors["bad"] != "down" {
		t.Fatal(rep.Errors)
	}
}
