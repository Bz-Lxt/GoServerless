package validate

import (
	"strings"
	"testing"

	"github.com/gogo/goserverless/internal/model"
)

func TestFunctionName(t *testing.T) {
	ok := []string{"hello", "fn-1", "abc", "a" + strings.Repeat("b", 38)}
	for _, n := range ok {
		if err := FunctionName(n); err != nil {
			t.Fatalf("%s should pass: %v", n, err)
		}
	}
	bad := []string{"", "Ab", "1abc", "a_b", "A-b", "ab", strings.Repeat("a", 41)}
	for _, n := range bad {
		if err := FunctionName(n); err == nil {
			t.Fatalf("%s should fail", n)
		}
	}
}

func TestEnvKeys(t *testing.T) {
	if err := Env(map[string]string{"FOO_BAR": "1"}); err != nil {
		t.Fatal(err)
	}
	if err := Env(map[string]string{"FOO BAR": "1"}); err == nil {
		t.Fatal("space key must fail")
	}
	if err := Env(map[string]string{"PATH;rm": "x"}); err == nil {
		t.Fatal("unsafe key must fail")
	}
}

func TestCron(t *testing.T) {
	if err := CronExpr("*/5 * * * *"); err != nil {
		t.Fatal(err)
	}
	if err := CronExpr("not-a-cron"); err == nil {
		t.Fatal("expected invalid cron")
	}
}

func TestTriggersRequireHTTP(t *testing.T) {
	err := Triggers([]model.Trigger{{Kind: model.TriggerCron, CronExpr: "*/5 * * * *"}})
	if err == nil {
		t.Fatal("http trigger required")
	}
	err = Triggers([]model.Trigger{
		{Kind: model.TriggerHTTP},
		{Kind: model.TriggerCron, CronExpr: "0 * * * *"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateInput(t *testing.T) {
	in := model.CreateFunctionInput{
		Name: "hello-fn", Runtime: model.RuntimeGo, Code: "func Handler() {}",
		TimeoutSec: 30, MemoryMB: 128, MaxConcurrency: 10,
	}
	if err := CreateInput(in); err != nil {
		t.Fatal(err)
	}
	in.MemoryMB = 77
	if err := CreateInput(in); err == nil {
		t.Fatal("bad memory")
	}
}
