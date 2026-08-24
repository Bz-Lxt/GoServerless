package gateway_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gogo/goserverless/internal/config"
	"github.com/gogo/goserverless/internal/dockerx"
	"github.com/gogo/goserverless/internal/gateway"
	"github.com/gogo/goserverless/internal/invoker"
	"github.com/gogo/goserverless/internal/model"
	"github.com/gogo/goserverless/internal/pool"
	rt "github.com/gogo/goserverless/internal/runtime"
)

type readyLoader struct{}

func (readyLoader) ReadyFunction(_ context.Context, name string) (*model.Function, *model.FunctionVersion, error) {
	return &model.Function{
		ID:             "fn-1",
		Name:           name,
		Runtime:        model.RuntimeGo,
		Status:         model.StatusReady,
		CurrentVersion: 1,
		TimeoutSec:     1,
		MaxConcurrency: 1,
	}, &model.FunctionVersion{
		ID:         "ver-1",
		FunctionID: "fn-1",
		Version:    1,
		Status:     model.StatusReady,
	}, nil
}

type stubRuntime struct{}

func (stubRuntime) Name() model.RuntimeName { return model.RuntimeGo }

func (stubRuntime) SandboxImage(rt.Images) string { return "sandbox-go:test" }

func (stubRuntime) Prepare(context.Context, rt.Source) (rt.Workdir, error) {
	return rt.Workdir{}, nil
}

func (stubRuntime) Build(context.Context, rt.Workdir, rt.Builder) (rt.Artifact, error) {
	return rt.Artifact{}, nil
}

func (stubRuntime) Pack(context.Context, rt.Artifact) (rt.Packed, error) {
	return rt.Packed{}, nil
}

func (stubRuntime) InvokeHint(string) rt.InvokeHint {
	return rt.InvokeHint{
		Runtime:    model.RuntimeGo,
		Command:    []string{"/unused"},
		WorkingDir: "/artifacts/demo/v1",
	}
}

func (stubRuntime) DefaultTemplate() string { return "" }

func TestGatewayPreservesColdFallbackUnavailable(t *testing.T) {
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_TLS_VERIFY", "")
	t.Setenv("DOCKER_CERT_PATH", "")
	t.Setenv("DOCKER_API_VERSION", "")

	var creates atomic.Int32
	daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/_ping"):
			w.Header().Set("API-Version", "1.49")
			w.Header().Set("OSType", "linux")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "OK")
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/create"):
			_, _ = io.Copy(io.Discard, r.Body)
			w.Header().Set("Content-Type", "application/json")
			if creates.Add(1) == 1 {
				w.WriteHeader(http.StatusCreated)
				_, _ = io.WriteString(w, `{"Id":"warm-container","Warnings":[]}`)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"message":"replacement create failed"}`)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/warm-container/start"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/containers/"):
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected Docker request", http.StatusNotFound)
		}
	}))
	defer daemon.Close()

	dockerHost := "tcp://" + strings.TrimPrefix(daemon.URL, "http://")
	dockerClient, err := dockerx.New(dockerHost)
	if err != nil {
		t.Fatalf("new Docker client: %v", err)
	}
	defer dockerClient.Close()

	root := t.TempDir()
	cfg := &config.Config{
		SocketRoot:      filepath.Join(root, "sockets"),
		PoolWarmSize:    1,
		PoolIdleTTL:     time.Hour,
		DefaultMemoryMB: 128,
		DefaultCPUNano:  500_000_000,
	}
	images := rt.Images{GoSandbox: "sandbox-go:test"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := pool.New(cfg, dockerClient, images, filepath.Join(root, "host-volume"))
	p.Start(ctx)
	defer p.Drain(context.Background())

	active, idle := p.Stats()
	if active != 0 || idle != 1 {
		t.Fatalf("prewarm failed: active=%d idle=%d", active, idle)
	}

	registry := rt.NewRegistry()
	registry.Register(stubRuntime{})
	invocation := invoker.New(registry, p, readyLoader{})
	gw := gateway.New(invocation, nil, 1<<20)
	router := chi.NewRouter()
	router.Handle("/api/v1/run/{function_name}", gw)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/run/demo", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s, want 503", rec.Code, rec.Body.String())
	}
	var got struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v (body=%s)", err, rec.Body.String())
	}
	if got.Code != "UNAVAILABLE" {
		t.Fatalf("code=%q message=%q, want UNAVAILABLE", got.Code, got.Message)
	}
	if !strings.Contains(got.Message, "replacement create failed") {
		t.Fatalf("message does not retain replacement failure: %q", got.Message)
	}
}
