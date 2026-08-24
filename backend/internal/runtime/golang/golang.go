package golang

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gogo/goserverless/internal/model"
	rt "github.com/gogo/goserverless/internal/runtime"
	"github.com/gogo/goserverless/internal/runtime/templates"
)

type Runtime struct {
	artifactRoot string
}

type buildError struct {
	cause error
	log   string
}

func (e *buildError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%v: %s", e.cause, e.log)
}

func (e *buildError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func wrapBuildError(cause error, log string) *buildError {
	if cause == nil {
		return nil
	}
	return &buildError{cause: cause, log: log}
}

func New(artifactRoot string) *Runtime {
	return &Runtime{artifactRoot: artifactRoot}
}

func (r *Runtime) Name() model.RuntimeName { return model.RuntimeGo }

func (r *Runtime) SandboxImage(images rt.Images) string { return images.GoSandbox }

func (r *Runtime) DefaultTemplate() string { return templates.GoHandler }

func (r *Runtime) Prepare(ctx context.Context, src rt.Source) (rt.Workdir, error) {
	dir := filepath.Join(r.artifactRoot, "_build", src.FunctionName, fmt.Sprintf("v%d", src.Version))
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return rt.Workdir{}, fmt.Errorf("clean workdir: %w", err)
	}
	userDir := filepath.Join(dir, "handler")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return rt.Workdir{}, fmt.Errorf("mkdir: %w", err)
	}
	code := rewriteGoPackage(src.Code)
	if err := os.WriteFile(filepath.Join(userDir, "handler.go"), []byte(code), 0o644); err != nil {
		return rt.Workdir{}, fmt.Errorf("write user code: %w", err)
	}
	mod := "module gscf.local/fn\n\ngo 1.23\n\nrequire gscf.local/user/handler v0.0.0\nreplace gscf.local/user/handler => ./handler\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		return rt.Workdir{}, err
	}
	userMod := "module gscf.local/user/handler\n\ngo 1.23\n"
	if err := os.WriteFile(filepath.Join(userDir, "go.mod"), []byte(userMod), 0o644); err != nil {
		return rt.Workdir{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(templates.GoWrapper), 0o644); err != nil {
		return rt.Workdir{}, fmt.Errorf("write wrapper: %w", err)
	}
	return rt.Workdir{Path: dir}, nil
}

func (r *Runtime) Build(ctx context.Context, w rt.Workdir, builder rt.Builder) (rt.Artifact, error) {
	out := filepath.Join(w.Path, "handler.bin")
	log, err := builder.BuildGo(ctx, w.Path, out)
	buildErr := wrapBuildError(err, log)
	if buildErr != nil {
		return rt.Artifact{}, buildErr
	}
	return rt.Artifact{Path: out, AbsPath: out, Filename: "handler.bin"}, buildErr
}

func (r *Runtime) Pack(_ context.Context, a rt.Artifact) (rt.Packed, error) {
	return rt.Packed{Artifact: a}, nil
}

func rewriteGoPackage(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return "package handler\n"
	}
	lines := strings.Split(code, "\n")
	if strings.HasPrefix(strings.TrimSpace(lines[0]), "package ") {
		lines[0] = "package handler"
		return strings.Join(lines, "\n") + "\n"
	}
	return "package handler\n\n" + code + "\n"
}

func (r *Runtime) InvokeHint(artifactRel string) rt.InvokeHint {
	bin := filepath.ToSlash(filepath.Join(artifactRel, "handler"))
	return rt.InvokeHint{
		Runtime:    model.RuntimeGo,
		Command:    []string{bin},
		WorkingDir: filepath.ToSlash(artifactRel),
	}
}
