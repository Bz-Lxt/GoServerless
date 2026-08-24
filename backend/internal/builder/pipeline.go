package builder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gogo/goserverless/internal/config"
	"github.com/gogo/goserverless/internal/dockerx"
	"github.com/gogo/goserverless/internal/idgen"
	"github.com/gogo/goserverless/internal/logger"
	"github.com/gogo/goserverless/internal/model"
	rt "github.com/gogo/goserverless/internal/runtime"
	"github.com/gogo/goserverless/internal/store"
	"github.com/gogo/goserverless/internal/timeutil"
)

type Pipeline struct {
	cfg     *config.Config
	st      *store.Store
	reg     *rt.Registry
	docker  *dockerx.Client
	hostVol string
	jobs    chan string
}

func New(cfg *config.Config, st *store.Store, reg *rt.Registry, d *dockerx.Client, hostVol string) *Pipeline {
	return &Pipeline{
		cfg:     cfg,
		st:      st,
		reg:     reg,
		docker:  d,
		hostVol: hostVol,
		jobs:    make(chan string, 64),
	}
}

func (p *Pipeline) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case name := <-p.jobs:
				if err := p.buildOne(ctx, name); err != nil {
					logger.Error(ctx, "build failed", "fn", name, "err", err)
				}
			}
		}
	}()
}

func (p *Pipeline) Enqueue(name string) {
	select {
	case p.jobs <- name:
	default:
		go func() { p.jobs <- name }()
	}
}

func (p *Pipeline) BuildGo(ctx context.Context, workdir, output string) (string, error) {
	rel, err := filepath.Rel(p.cfg.ArtifactRoot, workdir)
	if err != nil {
		rel = workdir
	}
	hostWork := dockerx.JoinHost(p.hostVol, "artifacts", rel)
	log, err := p.docker.RunBuilder(ctx, p.cfg.BuilderGoImage, hostWork, filepath.Base(output))
	return log, err
}

func (p *Pipeline) buildOne(ctx context.Context, name string) error {
	fn, err := p.st.GetFunctionByName(ctx, name)
	if err != nil {
		return err
	}
	ver, err := p.st.GetVersion(ctx, fn.ID, fn.CurrentVersion)
	if err != nil {
		return err
	}
	runtimeImpl, err := p.reg.Get(fn.Runtime)
	if err != nil {
		return err
	}

	src := rt.Source{
		FunctionName: fn.Name,
		Version:      ver.Version,
		Code:         ver.Code,
		WorkRoot:     p.cfg.ArtifactRoot,
	}
	wd, err := runtimeImpl.Prepare(ctx, src)
	if err != nil {
		return p.fail(ctx, fn, ver, "prepare: "+err.Error())
	}
	art, err := runtimeImpl.Build(ctx, wd, p)
	if err != nil {
		return p.fail(ctx, fn, ver, err.Error())
	}
	if art.BuildLog != "" {
		ver.BuildLog = strings.TrimSpace(ver.BuildLog + "\n" + art.BuildLog)
	}
	packed, err := runtimeImpl.Pack(ctx, art)
	if err != nil {
		return p.fail(ctx, fn, ver, "pack: "+err.Error())
	}

	finalDir := filepath.Join(p.cfg.ArtifactRoot, fn.Name, fmt.Sprintf("v%d", ver.Version))
	if err := os.MkdirAll(finalDir, 0o755); err != nil {
		return p.fail(ctx, fn, ver, err.Error())
	}
	if fn.Runtime == model.RuntimeGo {
		dest := filepath.Join(finalDir, "handler")
		if err := copyFile(packed.AbsPath, dest); err != nil {
			return p.fail(ctx, fn, ver, "copy artifact: "+err.Error())
		}
		_ = os.Chmod(dest, 0o755)
		ver.ArtifactPath = filepath.ToSlash(filepath.Join("/artifacts", fn.Name, fmt.Sprintf("v%d", ver.Version), "handler"))
	} else {
		ver.ArtifactPath = filepath.ToSlash(filepath.Join("/artifacts", fn.Name, fmt.Sprintf("v%d", ver.Version)))
	}
	ver.Status = model.StatusReady
	ver.BuildLog = strings.TrimSpace(ver.BuildLog + "\nbuild ok")
	if err := p.st.UpdateVersion(ctx, ver); err != nil {
		return err
	}
	fn.Status = model.StatusReady
	fn.UpdatedAt = timeutil.NowUTC()
	if err := p.st.UpdateFunction(ctx, fn); err != nil {
		return err
	}
	logger.Info(ctx, "function ready", "fn", fn.Name, "version", ver.Version)
	return nil
}

func (p *Pipeline) fail(ctx context.Context, fn *model.Function, ver *model.FunctionVersion, msg string) error {
	ver.Status = model.StatusFailed
	ver.BuildLog = strings.TrimSpace(ver.BuildLog + "\n" + msg)
	_ = p.st.UpdateVersion(ctx, ver)
	fn.Status = model.StatusFailed
	fn.UpdatedAt = timeutil.NowUTC()
	_ = p.st.UpdateFunction(ctx, fn)
	return fmt.Errorf("%w: %s", model.ErrBuildFailed, msg)
}

func (p *Pipeline) NewVersion(ctx context.Context, fn *model.Function, code string) (*model.FunctionVersion, error) {
	n, err := p.st.NextVersion(ctx, fn.ID)
	if err != nil {
		return nil, err
	}
	v := &model.FunctionVersion{
		ID:         idgen.UUID(),
		FunctionID: fn.ID,
		Version:    n,
		Status:     model.StatusBuilding,
		Code:       code,
		CreatedAt:  timeutil.NowUTC(),
	}
	if err := p.st.InsertVersion(ctx, v); err != nil {
		return nil, err
	}
	fn.CurrentVersion = n
	fn.Status = model.StatusBuilding
	if err := p.st.UpdateFunction(ctx, fn); err != nil {
		return nil, err
	}
	_ = p.st.TrimVersions(ctx, fn.ID, 10)
	return v, nil
}

func copyFile(src, dst string) error {
	in, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, in, 0o755)
}
