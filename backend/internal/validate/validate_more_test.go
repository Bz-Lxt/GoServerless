package validate

import (
	"strings"
	"testing"

	"github.com/gogo/goserverless/internal/model"
)

func TestCodeLimits(t *testing.T) {
	if err := Code(""); err == nil {
		t.Fatal("empty")
	}
	if err := Code(strings.Repeat("a", MaxCodeBytes+1)); err == nil {
		t.Fatal("too large")
	}
	if err := Code("ok"); err != nil {
		t.Fatal(err)
	}
}

func TestTimeoutMemoryConcurrency(t *testing.T) {
	if err := Timeout(0); err == nil {
		t.Fatal("0")
	}
	if err := Timeout(301); err == nil {
		t.Fatal("301")
	}
	if err := MemoryMB(100); err == nil {
		t.Fatal("100")
	}
	if err := Concurrency(0); err == nil {
		t.Fatal("0 conc")
	}
	if err := Concurrency(51); err == nil {
		t.Fatal("51")
	}
}

func TestHTTPTriggerRejectsCronExpr(t *testing.T) {
	err := Triggers([]model.Trigger{{Kind: model.TriggerHTTP, CronExpr: "* * * * *"}})
	if err == nil {
		t.Fatal("http+cron_expr")
	}
}

func TestEnvPairLimit(t *testing.T) {
	m := map[string]string{}
	for i := 0; i < 33; i++ {
		m["K"+strings.Repeat("X", i%5)+string(rune('A'+i%26))] = "v"
	}
	if err := Env(m); err == nil {
		t.Fatal("33 pairs")
	}
}
