package agentrun

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestValidEnvKey(t *testing.T) {
	if !ValidEnvKey("FOO_1") || ValidEnvKey("FOO BAR") || ValidEnvKey("1A") {
		t.Fatal("key rules")
	}
}

func TestSafeEnvDropsUnsafe(t *testing.T) {
	env := SafeEnv(map[string]string{"OK": "1", "BAD;x": "2"})
	joined := ""
	for _, e := range env {
		joined += e + "\n"
	}
	if !contains(joined, "OK=1") || contains(joined, "BAD") {
		t.Fatal(joined)
	}
}

func TestRunEchoJSON(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	res := Run(context.Background(), Request{
		Command:   []string{"/bin/sh", "-c", `printf '{"status":201,"headers":{"x":"y"},"body":"hi"}'`},
		TimeoutMS: 2000,
	})
	if res.Status != 201 || res.Body != "hi" || res.Headers["x"] != "y" {
		t.Fatalf("%+v", res)
	}
}

func TestRunTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	res := Run(context.Background(), Request{
		Command:   []string{"/bin/sleep", "2"},
		TimeoutMS: 80,
	})
	if res.Status != 504 {
		t.Fatalf("status %d", res.Status)
	}
}

func TestEmptyCommand(t *testing.T) {
	res := Run(context.Background(), Request{})
	if res.Status != 500 {
		t.Fatal(res)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (func() bool {
		return (time.Now().UnixNano() >= 0) && (stringIndex(s, sub) >= 0)
	})())
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
