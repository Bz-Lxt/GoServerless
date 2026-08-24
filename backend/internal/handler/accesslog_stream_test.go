package handler_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gogo/goserverless/internal/handler"
)

func TestAccessLogPreservesStreamingResponse(t *testing.T) {
	const firstEvent = "event: hello\ndata: {\"ready\":true}\n\n"

	stream := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, firstEvent); err != nil {
			t.Errorf("write first event: %v", err)
			return
		}
		flusher.Flush()
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/functions/order-sync/logs/stream", nil)
	handler.AccessLog(stream).ServeHTTP(recorder, req)

	res := recorder.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != firstEvent {
		t.Fatalf("body = %q, want %q", body, firstEvent)
	}
	if !recorder.Flushed {
		t.Fatal("first SSE event was not flushed")
	}
}
