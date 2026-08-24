package nodejs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	rt "github.com/gogo/goserverless/internal/runtime"
)

func TestPrepareAndHint(t *testing.T) {
	r := New(t.TempDir())
	wd, err := r.Prepare(context.Background(), rt.Source{
		FunctionName: "jsfn",
		Version:      1,
		Code:         "module.exports = async () => ({statusCode:200,body:'ok'});",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(wd.Path, "wrapper.js")); err != nil {
		t.Fatal(err)
	}
	art, err := r.Build(context.Background(), wd, nil)
	if err != nil {
		t.Fatal(err)
	}
	if art.Filename != "wrapper.js" {
		t.Fatal(art.Filename)
	}
	h := r.InvokeHint("/artifacts/jsfn/v1")
	if len(h.Command) != 2 || h.Command[0] != "node" {
		t.Fatalf("%v", h.Command)
	}
}
