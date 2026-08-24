package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gogo/goserverless/internal/agentrun"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	sock := os.Getenv("SOCKET_PATH")
	if sock == "" {
		sock = "/run/gscf/agent.sock"
	}
	_ = os.Remove(sock)
	if err := os.MkdirAll(filepath.Dir(sock), 0o755); err != nil {
		log.Error("mkdir socket dir", "err", err)
		os.Exit(1)
	}
	ln, err := net.Listen("unix", sock)
	if err != nil {
		log.Error("listen", "err", err)
		os.Exit(1)
	}
	_ = os.Chmod(sock, 0o666)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/invoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req agentrun.Request
		if err := json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(&req); err != nil {
			writeJSON(w, agentrun.Response{Status: 400, Error: "bad request: " + err.Error()})
			return
		}
		writeJSON(w, agentrun.Run(r.Context(), req))
	})

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Error("serve", "err", err)
			os.Exit(1)
		}
	}()
	log.Info("sandbox agent ready", "socket", sock)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
