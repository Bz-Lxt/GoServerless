package artifact

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathsAndLifecycle(t *testing.T) {
	m := New(t.TempDir())
	dir, err := m.Ensure("hello", 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.WriteFile(dir, "handler", []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	sum, err := m.Checksum(filepath.Join(dir, "handler"))
	if err != nil || len(sum) != 64 {
		t.Fatalf("sum %s %v", sum, err)
	}
	if got := m.ContainerPath("hello", 2, "handler"); got != "/artifacts/hello/v2/handler" {
		t.Fatal(got)
	}
	if err := m.RemoveVersion("hello", 2); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("version dir should be gone")
	}
}

func TestRejectsTraversal(t *testing.T) {
	m := New(t.TempDir())
	if err := m.RemoveFunction("../x"); err == nil {
		t.Fatal("expected traversal reject")
	}
	if err := m.WriteFile(m.Root, "../x", []byte("a"), 0644); err == nil {
		t.Fatal("expected filename reject")
	}
}
