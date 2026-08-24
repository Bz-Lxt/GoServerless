package invoker_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gogo/goserverless/internal/config"
	"github.com/gogo/goserverless/internal/dockerx"
	"github.com/gogo/goserverless/internal/invoker"
	"github.com/gogo/goserverless/internal/model"
	"github.com/gogo/goserverless/internal/pool"
	rt "github.com/gogo/goserverless/internal/runtime"
	rtgo "github.com/gogo/goserverless/internal/runtime/golang"
)

type readyLoader struct {
	fn  model.Function
	ver model.FunctionVersion
}

func (l readyLoader) ReadyFunction(context.Context, string) (*model.Function, *model.FunctionVersion, error) {
	fn := l.fn
	ver := l.ver
	return &fn, &ver, nil
}

func TestInvokeReleasesConcurrencyAfterAcquireFailure(t *testing.T) {
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_API_VERSION", "")
	t.Setenv("DOCKER_TLS_VERIFY", "")
	t.Setenv("DOCKER_CERT_PATH", "")

	requestEntered := make(chan struct{}, 1)
	releaseRequest := make(chan struct{})
	var blockFirst sync.Once
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseRequest) }) }

	daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_ping" {
			w.Header().Set("API-Version", "1.45")
			w.Header().Set("OSType", "linux")
			w.WriteHeader(http.StatusOK)
			return
		}
		select {
		case requestEntered <- struct{}{}:
		default:
		}
		blockFirst.Do(func() { <-releaseRequest })
		http.Error(w, "sandbox host unavailable", http.StatusServiceUnavailable)
	}))
	defer daemon.Close()

	dockerHost := "tcp://" + strings.TrimPrefix(daemon.URL, "http://")
	dockerClient, err := dockerx.New(dockerHost)
	if err != nil {
		t.Fatalf("create docker client: %v", err)
	}
	defer dockerClient.Close()
	defer release()

	reg := rt.NewRegistry()
	reg.Register(rtgo.New(t.TempDir()))
	cfg := &config.Config{
		SocketRoot:      t.TempDir(),
		DefaultMemoryMB: 128,
		DefaultCPUNano:  500_000_000,
	}
	p := pool.New(cfg, dockerClient, rt.Images{GoSandbox: "sandbox-go:test"}, t.TempDir())
	loader := readyLoader{
		fn: model.Function{
			Name:           "payment-sync",
			Runtime:        model.RuntimeGo,
			Status:         model.StatusReady,
			TimeoutSec:     1,
			MemoryMB:       128,
			MaxConcurrency: 1,
		},
		ver: model.FunctionVersion{Version: 1, Status: model.StatusReady},
	}
	inv := invoker.New(reg, p, loader)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	event := invoker.HTTPEvent{Method: http.MethodPost, Path: "/", Headers: map[string]string{}, Query: map[string]string{}}

	firstDone := make(chan error, 1)
	go func() {
		_, err := inv.Invoke(ctx, loader.fn.Name, model.TriggerHTTP, event)
		firstDone <- err
	}()

	select {
	case <-requestEntered:
	case <-ctx.Done():
		t.Fatalf("first invoke did not reach sandbox acquisition: %v", ctx.Err())
	}

	_, concurrentErr := inv.Invoke(ctx, loader.fn.Name, model.TriggerHTTP, event)
	release()

	var firstErr error
	select {
	case firstErr = <-firstDone:
	case <-ctx.Done():
		t.Fatalf("first invoke did not finish: %v", ctx.Err())
	}
	if !errors.Is(concurrentErr, model.ErrConcurrency) {
		t.Fatalf("concurrent invoke error = %v, want concurrency limit", concurrentErr)
	}
	if !errors.Is(firstErr, model.ErrUnavailable) {
		t.Fatalf("first invoke error = %v, want runtime unavailable", firstErr)
	}

	_, retryErr := inv.Invoke(ctx, loader.fn.Name, model.TriggerHTTP, event)
	if !errors.Is(retryErr, model.ErrUnavailable) {
		t.Fatalf("retry after completed invoke error = %v, want runtime unavailable", retryErr)
	}
}
