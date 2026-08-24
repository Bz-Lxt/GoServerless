package config

import (
	"os"
	"testing"
)

func TestLoadRequiresDatabase(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected missing DATABASE_URL")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("POOL_WARM_SIZE", "3")
	t.Setenv("POOL_IDLE_TTL_SEC", "300")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.DefaultMemoryMB != 128 || c.PoolWarmSize != 3 {
		t.Fatalf("%+v", c)
	}
	if c.IsProd() {
		t.Fatal("dev")
	}
}

func TestPoolBounds(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://x")
	os.Setenv("POOL_WARM_SIZE", "99")
	defer os.Unsetenv("POOL_WARM_SIZE")
	if _, err := Load(); err == nil {
		t.Fatal("warm size 99")
	}
}
