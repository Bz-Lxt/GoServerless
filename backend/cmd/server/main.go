package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gogo/goserverless/internal/builder"
	"github.com/gogo/goserverless/internal/cache"
	"github.com/gogo/goserverless/internal/config"
	"github.com/gogo/goserverless/internal/dockerx"
	"github.com/gogo/goserverless/internal/gateway"
	"github.com/gogo/goserverless/internal/handler"
	"github.com/gogo/goserverless/internal/httpx"
	"github.com/gogo/goserverless/internal/invoker"
	"github.com/gogo/goserverless/internal/logger"
	"github.com/gogo/goserverless/internal/logstream"
	"github.com/gogo/goserverless/internal/pool"
	rt "github.com/gogo/goserverless/internal/runtime"
	rtgo "github.com/gogo/goserverless/internal/runtime/golang"
	rtnode "github.com/gogo/goserverless/internal/runtime/nodejs"
	"github.com/gogo/goserverless/internal/scheduler"
	"github.com/gogo/goserverless/internal/service"
	"github.com/gogo/goserverless/internal/store"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe local health endpoint and exit")
	flag.Parse()
	if *healthcheck {
		os.Exit(probe())
	}

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	if cfg.IsProd() && cfg.LogLevel == "debug" {
		cfg.LogLevel = "info"
	}
	logger.Init(cfg.LogLevel)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error(ctx, "postgres", "err", err)
		os.Exit(1)
	}
	defer st.Close()
	if err := st.Migrate(ctx, store.SchemaSQL); err != nil {
		logger.Error(ctx, "migrate", "err", err)
		os.Exit(1)
	}

	rd, err := cache.Connect(cfg.RedisAddr)
	if err != nil {
		logger.Error(ctx, "redis", "err", err)
		os.Exit(1)
	}
	defer rd.Close()

	dk, err := dockerx.New(cfg.DockerHost)
	if err != nil {
		logger.Error(ctx, "docker", "err", err)
		os.Exit(1)
	}
	defer dk.Close()
	if err := dk.Ping(ctx); err != nil {
		logger.Error(ctx, "docker ping", "err", err)
		os.Exit(1)
	}
	if err := dk.EnsureDirs(cfg.ArtifactRoot, cfg.SocketRoot); err != nil {
		logger.Error(ctx, "mkdir", "err", err)
		os.Exit(1)
	}
	_ = os.Chmod(cfg.SocketRoot, 0o777)
	// Reap any builder/sandbox containers orphaned by a previous crash or
	// forced restart so they do not accumulate across rolling deploys.
	dk.ReapOrphans(ctx)
	hostVol, err := dk.VolumeMountpoint(ctx, cfg.ArtifactVolume)
	if err != nil {
		logger.Warn(ctx, "volume inspect failed, falling back to container paths", "err", err)
		hostVol = "/var/lib/goserverless"
	}

	images := rt.Images{
		GoSandbox:   cfg.SandboxGoImage,
		NodeSandbox: cfg.SandboxNodeImage,
		GoBuilder:   cfg.BuilderGoImage,
	}
	reg := rt.NewRegistry()
	reg.Register(rtgo.New(cfg.ArtifactRoot))
	reg.Register(rtnode.New(cfg.ArtifactRoot))

	p := pool.New(cfg, dk, images, hostVol)
	p.Start(ctx)
	defer p.Drain(context.Background())

	pipe := builder.New(cfg, st, reg, dk, hostVol)
	pipe.Start(ctx)

	fns := service.NewFunctions(cfg, st, reg, pipe, rd, p)
	hub := logstream.NewHub()
	rec := service.NewRecorder(st, hub, rd)
	inv := invoker.New(reg, p, fns)
	gw := gateway.New(inv, rec, cfg.InvokeBodyLimit)
	sched := scheduler.New(st, inv, rec)
	sched.Start(ctx)

	api := handler.NewAPI(fns, hub)
	authAPI := handler.NewAuthAPI(st, cfg.AuthUser, cfg.AuthPass)
	authMW := handler.NewAuth(st, cfg.AuthUser, cfg.AuthPass)
	srv := httpx.NewServer(cfg.HTTPAddr, httpx.NewRouter(api, authMW, authAPI, gw))

	go func() {
		t := time.NewTicker(10 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_ = st.SweepSessions(ctx)
			}
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		logger.Info(ctx, "api listening", "addr", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sig:
		logger.Info(ctx, "shutdown signal")
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, "server exit", "err", err)
		}
	}
	cancel()
	shCtx, shCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shCancel()
	_ = srv.Shutdown(shCtx)
}

func probe() int {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://127.0.0.1:8080/api/v1/health")
	if err != nil {
		return 1
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 200 {
		return 1
	}
	return 0
}
