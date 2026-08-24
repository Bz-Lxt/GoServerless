package golang_test

import (
	"context"
	"os"
	"testing"

	rt "github.com/gogo/goserverless/internal/runtime"
	runtimego "github.com/gogo/goserverless/internal/runtime/golang"
)

type writingBuilder struct {
	payload []byte
}

func (b writingBuilder) BuildGo(_ context.Context, _, output string) (string, error) {
	return "build ok", os.WriteFile(output, b.payload, 0o755)
}

func TestBuildSuccessReturnsNilError(t *testing.T) {
	var runtimeImpl rt.Runtime = runtimego.New(t.TempDir())
	workdir := t.TempDir()
	want := []byte("compiled-handler")

	artifact, err := runtimeImpl.Build(context.Background(), rt.Workdir{Path: workdir}, writingBuilder{payload: want})
	if err != nil {
		t.Fatal("successful build returned a non-nil error")
	}
	got, err := os.ReadFile(artifact.AbsPath)
	if err != nil {
		t.Fatalf("read returned artifact: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("returned artifact = %q, want %q", got, want)
	}
}
