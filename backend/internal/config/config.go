package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env            string
	LogLevel       string
	HTTPAddr       string
	DatabaseURL    string
	RedisAddr      string
	DockerHost     string
	ArtifactRoot   string
	SocketRoot     string
	ArtifactVolume string
	SandboxGoImage    string
	SandboxNodeImage  string
	BuilderGoImage    string
	PoolWarmSize      int
	PoolIdleTTL       time.Duration
	AuthUser          string
	AuthPass          string
	PublicBaseURL     string
	InvokeBodyLimit   int64
	DefaultTimeout    time.Duration
	DefaultMemoryMB   int
	DefaultCPUNano    int64
	MaxConcurrency    int
}

func Load() (*Config, error) {
	c := &Config{
		Env:              envOr("APP_ENV", "development"),
		LogLevel:         envOr("LOG_LEVEL", "info"),
		HTTPAddr:         envOr("HTTP_ADDR", ":8080"),
		DatabaseURL:      envOr("DATABASE_URL", ""),
		RedisAddr:        envOr("REDIS_ADDR", "127.0.0.1:6379"),
		DockerHost:       envOr("DOCKER_HOST", "unix:///var/run/docker.sock"),
		ArtifactRoot:     envOr("ARTIFACT_ROOT", "/var/lib/goserverless/artifacts"),
		SocketRoot:       envOr("SOCKET_ROOT", "/var/lib/goserverless/sockets"),
		ArtifactVolume:   envOr("ARTIFACT_VOLUME", "goserverless_gscfdata"),
		SandboxGoImage:   envOr("SANDBOX_GO_IMAGE", "goserverless/sandbox-go:local"),
		SandboxNodeImage: envOr("SANDBOX_NODE_IMAGE", "goserverless/sandbox-node:local"),
		BuilderGoImage:   envOr("BUILDER_GO_IMAGE", "goserverless/builder-go:local"),
		PoolWarmSize:     envInt("POOL_WARM_SIZE", 3),
		PoolIdleTTL:      time.Duration(envInt("POOL_IDLE_TTL_SEC", 300)) * time.Second,
		AuthUser:         envOr("AUTH_USER", "admin"),
		AuthPass:         envOr("AUTH_PASS", "admin123"),
		PublicBaseURL:    strings.TrimRight(envOr("PUBLIC_BASE_URL", "http://localhost:42818"), "/"),
		InvokeBodyLimit:  int64(envInt("INVOKE_BODY_LIMIT", 1<<20)),
		DefaultTimeout:   time.Duration(envInt("DEFAULT_TIMEOUT_SEC", 30)) * time.Second,
		DefaultMemoryMB:  envInt("DEFAULT_MEMORY_MB", 128),
		DefaultCPUNano:   int64(envInt("DEFAULT_CPU_NANO", 500_000_000)),
		MaxConcurrency:   envInt("MAX_CONCURRENCY", 10),
	}
	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if c.PoolWarmSize < 0 || c.PoolWarmSize > 20 {
		return nil, fmt.Errorf("POOL_WARM_SIZE must be 0-20")
	}
	if c.PoolIdleTTL < 10*time.Second {
		return nil, fmt.Errorf("POOL_IDLE_TTL_SEC too small")
	}
	return c, nil
}

func (c *Config) IsProd() bool {
	return strings.EqualFold(c.Env, "production")
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
