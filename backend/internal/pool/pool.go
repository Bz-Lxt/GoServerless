package pool

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gogo/goserverless/internal/config"
	"github.com/gogo/goserverless/internal/dockerx"
	"github.com/gogo/goserverless/internal/idgen"
	"github.com/gogo/goserverless/internal/logger"
	"github.com/gogo/goserverless/internal/model"
	rt "github.com/gogo/goserverless/internal/runtime"
)

// SandboxCreator abstracts container lifecycle so the pool can be tested
// without a live Docker daemon. *dockerx.Client satisfies this interface.
type SandboxCreator interface {
	CreateSandbox(ctx context.Context, opt dockerx.CreateSandboxOpts) (string, error)
	RemoveContainer(ctx context.Context, id string) error
}

type AcquireResult struct {
	Slot      *Slot
	ColdStart bool
	Wakeup    time.Duration
}

type Pool struct {
	cfg     *config.Config
	docker  SandboxCreator
	images  rt.Images
	hostVol string

	mu    sync.Mutex
	slots map[string]*Slot
}

func New(cfg *config.Config, d SandboxCreator, images rt.Images, hostVol string) *Pool {
	return &Pool{
		cfg:     cfg,
		docker:  d,
		images:  images,
		hostVol: hostVol,
		slots:   map[string]*Slot{},
	}
}

func (p *Pool) Start(ctx context.Context) {
	go p.reaper(ctx)
	if p.cfg.PoolWarmSize <= 0 {
		return
	}
	// Pre-warm mixed runtimes: prefer go, then node, then go again.
	runtimes := []model.RuntimeName{model.RuntimeGo, model.RuntimeNodeJS, model.RuntimeGo}
	n := p.cfg.PoolWarmSize
	for i := 0; i < n; i++ {
		rtName := runtimes[i%len(runtimes)]
		if _, err := p.create(ctx, rtName); err != nil {
			logger.Warn(ctx, "warm pool slot failed", "runtime", rtName, "err", err)
		}
	}
	logger.Info(ctx, "container pool warmed", "size", len(p.slots))
}

func (p *Pool) Acquire(ctx context.Context, runtime model.RuntimeName) (*AcquireResult, error) {
	start := time.Now()
	p.mu.Lock()
	for _, s := range p.slots {
		if s.Runtime == runtime && s.Snapshot() == SlotWarm {
			s.Mark(SlotBusy)
			s.Touch()
			p.mu.Unlock()
			if err := p.waitReady(ctx, s, 800*time.Millisecond); err != nil {
				p.discard(s)
				acq, coldErr := p.cold(ctx, runtime, start)
				if coldErr != nil {
					// The cold-fallback failure is authoritative:
					// it already carries the correct 503/UNAVAILABLE
					// classification that the gateway relies on for
					// node failover. Wrap it with %w so that
					// *model.Error stays in the unwrap chain and
					// model.HTTPStatus/CodeOf surface 503/UNAVAILABLE.
					// The warm-readiness failure is a plain fmt.Errorf;
					// it must NOT be the %w target or it hijacks the
					// chain and degrades the response to 500/INTERNAL.
					return nil, fmt.Errorf("warm sandbox unavailable: %v; cold fallback failed: %w", err, coldErr)
				}
				return acq, nil
			}
			return &AcquireResult{Slot: s, ColdStart: false, Wakeup: time.Since(start)}, nil
		}
	}
	p.mu.Unlock()
	return p.cold(ctx, runtime, start)
}

func (p *Pool) Release(s *Slot) {
	if s == nil {
		return
	}
	s.Touch()
	if s.Snapshot() == SlotDead {
		p.discard(s)
		return
	}
	s.Mark(SlotWarm)
}

func (p *Pool) Discard(s *Slot) { p.discard(s) }

func (p *Pool) Stats() (active, idle int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, s := range p.slots {
		switch s.Snapshot() {
		case SlotBusy, SlotBooting:
			active++
		case SlotWarm:
			idle++
		}
	}
	return
}

func (p *Pool) Drain(ctx context.Context) {
	p.mu.Lock()
	all := make([]*Slot, 0, len(p.slots))
	for _, s := range p.slots {
		all = append(all, s)
	}
	p.mu.Unlock()
	for _, s := range all {
		p.discard(s)
	}
}

func (p *Pool) cold(ctx context.Context, runtime model.RuntimeName, start time.Time) (*AcquireResult, error) {
	s, err := p.create(ctx, runtime)
	if err != nil {
		return nil, err
	}
	s.Mark(SlotBusy)
	if err := p.waitReady(ctx, s, 3*time.Second); err != nil {
		p.discard(s)
		return nil, err
	}
	return &AcquireResult{Slot: s, ColdStart: true, Wakeup: time.Since(start)}, nil
}

func (p *Pool) create(ctx context.Context, runtime model.RuntimeName) (*Slot, error) {
	id := "slot-" + idgen.Token(6)
	socketDir := filepath.Join(p.cfg.SocketRoot, id)
	if err := os.MkdirAll(socketDir, 0o777); err != nil {
		return nil, err
	}
	_ = os.Chmod(socketDir, 0o777)
	image := p.images.GoSandbox
	if runtime == model.RuntimeNodeJS {
		image = p.images.NodeSandbox
	}
	hostSock := dockerx.JoinHost(p.hostVol, "sockets", id)
	hostArt := dockerx.JoinHost(p.hostVol, "artifacts")
	name := "gscf-" + id
	cid, err := p.docker.CreateSandbox(ctx, dockerx.CreateSandboxOpts{
		Name:        name,
		Image:       image,
		Runtime:     string(runtime),
		SlotID:      id,
		MemoryMB:    p.cfg.DefaultMemoryMB,
		CPUNano:     p.cfg.DefaultCPUNano,
		SocketDir:   hostSock,
		ArtifactDir: hostArt,
	})
	if err != nil {
		return nil, model.Unavailable("create sandbox: " + err.Error())
	}
	s := &Slot{
		ID:         id,
		Runtime:    runtime,
		Container:  cid,
		SocketPath: filepath.Join(socketDir, "agent.sock"),
		State:      SlotBooting,
		LastUsed:   time.Now(),
		CreatedAt:  time.Now(),
	}
	p.mu.Lock()
	p.slots[id] = s
	p.mu.Unlock()
	s.Mark(SlotWarm)
	return s, nil
}

func (p *Pool) discard(s *Slot) {
	if s == nil {
		return
	}
	s.Mark(SlotDead)
	p.mu.Lock()
	delete(p.slots, s.ID)
	p.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := p.docker.RemoveContainer(ctx, s.Container); err != nil {
		logger.Warn(ctx, "remove sandbox failed", "slot", s.ID, "err", err)
	}
	_ = os.RemoveAll(filepath.Join(p.cfg.SocketRoot, s.ID))
}

func (p *Pool) waitReady(ctx context.Context, s *Slot, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: 80 * time.Millisecond}
			return d.DialContext(ctx, "unix", s.SocketPath)
		},
	}
	client := &http.Client{Transport: transport, Timeout: 120 * time.Millisecond}
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://sandbox/health", nil)
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(15 * time.Millisecond)
	}
	return fmt.Errorf("sandbox agent not ready: %s", s.SocketPath)
}

func (p *Pool) reaper(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.reapOnce()
		}
	}
}

func (p *Pool) reapOnce() {
	ttl := p.cfg.PoolIdleTTL
	p.mu.Lock()
	victims := make([]*Slot, 0)
	for _, s := range p.slots {
		if s.Snapshot() == SlotWarm && s.IdleFor() > ttl {
			victims = append(victims, s)
		}
	}
	p.mu.Unlock()
	for _, s := range victims {
		logger.Info(context.Background(), "reap idle sandbox", "slot", s.ID, "idle", s.IdleFor().String())
		p.discard(s)
	}
}
