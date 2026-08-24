package golang

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	rt "github.com/gogo/goserverless/internal/runtime"
)

func TestRewritePackage(t *testing.T) {
	got := rewriteGoPackage("package main\nfunc Handler() {}")
	if !strings.HasPrefix(got, "package handler") {
		t.Fatal(got)
	}
	got = rewriteGoPackage("func Handler() {}")
	if !strings.Contains(got, "package handler") {
		t.Fatal(got)
	}
}

func TestPrepareWritesWorkdir(t *testing.T) {
	root := t.TempDir()
	r := New(root)
	wd, err := r.Prepare(context.Background(), rt.Source{
		FunctionName: "hello",
		Version:      1,
		Code:         "package main\nfunc Handler() {}\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(wd.Path, "main.go")); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(wd.Path, "handler", "handler.go"))
	if !strings.Contains(string(b), "package handler") {
		t.Fatalf("user package not rewritten: %s", b)
	}
}

func TestInvokeHintUsesHandlerBinary(t *testing.T) {
	r := New("/tmp")
	h := r.InvokeHint("/artifacts/hello/v1")
	if len(h.Command) != 1 || !strings.HasSuffix(h.Command[0], "/handler") {
		t.Fatalf("%v", h.Command)
	}
}
