package dockerx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gogo/goserverless/internal/dockerx"
)

func TestRunBuilderRemovesContainerAfterCancellation(t *testing.T) {
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_API_VERSION", "1.47")
	t.Setenv("DOCKER_CERT_PATH", "")
	t.Setenv("DOCKER_TLS_VERIFY", "")

	waitStarted := make(chan struct{})
	removed := make(chan struct{})
	releaseWait := make(chan struct{})
	var waitOnce, removeOnce sync.Once

	daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/_ping":
			w.Header().Set("API-Version", "1.47")
			w.Header().Set("Docker-Experimental", "false")
			w.Header().Set("OSType", "linux")
			_, _ = w.Write([]byte("OK"))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/create"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"Id":"builder-test","Warnings":[]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/builder-test/start"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/builder-test/wait"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			waitOnce.Do(func() { close(waitStarted) })
			select {
			case <-r.Context().Done():
			case <-releaseWait:
			}
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/builder-test/kill"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/containers/builder-test/logs"):
			w.Header().Set("Content-Type", "application/vnd.docker.raw-stream")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/containers/builder-test"):
			removeOnce.Do(func() { close(removed) })
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected Docker API request", http.StatusNotFound)
			t.Errorf("unexpected Docker API request: %s %s", r.Method, r.URL.RequestURI())
		}
	}))
	defer daemon.Close()
	defer close(releaseWait)

	client, err := dockerx.New(daemon.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := client.RunBuilder(ctx, "builder:latest", "/work", "handler")
		result <- err
	}()

	select {
	case <-waitStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("builder did not begin waiting for the container")
	}
	cancel()

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("RunBuilder succeeded after cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunBuilder did not return after cancellation")
	}

	select {
	case <-removed:
	default:
		t.Fatal("builder container still exists after RunBuilder returned")
	}
}
