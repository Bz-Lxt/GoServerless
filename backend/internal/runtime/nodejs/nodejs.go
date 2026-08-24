package nodejs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gogo/goserverless/internal/model"
	rt "github.com/gogo/goserverless/internal/runtime"
	"github.com/gogo/goserverless/internal/runtime/templates"
)

type Runtime struct {
	artifactRoot string
}

func New(artifactRoot string) *Runtime {
	return &Runtime{artifactRoot: artifactRoot}
}

func (r *Runtime) Name() model.RuntimeName { return model.RuntimeNodeJS }

func (r *Runtime) SandboxImage(images rt.Images) string { return images.NodeSandbox }

func (r *Runtime) DefaultTemplate() string { return templates.NodeHandler }

func (r *Runtime) Prepare(_ context.Context, src rt.Source) (rt.Workdir, error) {
	dir := filepath.Join(r.artifactRoot, src.FunctionName, fmt.Sprintf("v%d", src.Version))
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return rt.Workdir{}, fmt.Errorf("clean: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return rt.Workdir{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, "handler.js"), []byte(src.Code), 0o644); err != nil {
		return rt.Workdir{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, "wrapper.js"), []byte(templates.NodeWrapper), 0o644); err != nil {
		return rt.Workdir{}, err
	}
	return rt.Workdir{Path: dir}, nil
}

func (r *Runtime) Build(_ context.Context, w rt.Workdir, _ rt.Builder) (rt.Artifact, error) {
	// Node is interpret-only: Build is a documented no-op pack of the workdir.
	return rt.Artifact{
		Path:     w.Path,
		AbsPath:  w.Path,
		Filename: "wrapper.js",
	}, nil
}

func (r *Runtime) Pack(_ context.Context, a rt.Artifact) (rt.Packed, error) {
	return rt.Packed{Artifact: a}, nil
}

func (r *Runtime) InvokeHint(artifactRel string) rt.InvokeHint {
	wrapper := filepath.ToSlash(filepath.Join(artifactRel, "wrapper.js"))
	return rt.InvokeHint{
		Runtime: model.RuntimeNodeJS,
		Command: []string{"node", wrapper},
		WorkingDir: filepath.ToSlash(artifactRel),
	}
}
