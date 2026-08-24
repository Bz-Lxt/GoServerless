// Package runtime 定义语言无关的构建/打包契约。Invoker 禁止按语言分支。
package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/gogo/goserverless/internal/model"
)

type Source struct {
	FunctionName string
	Version      int
	Code         string
	WorkRoot     string
}

type Workdir struct {
	Path string
}

type Artifact struct {
	Path     string // 容器内可见路径，相对 ARTIFACT_ROOT
	AbsPath  string // 后端容器内绝对路径
	Filename string
}

type Packed struct {
	Artifact
}

type InvokeHint struct {
	Runtime    model.RuntimeName
	Command    []string
	WorkingDir string
}

type Runtime interface {
	Name() model.RuntimeName
	SandboxImage(images Images) string
	Prepare(ctx context.Context, src Source) (Workdir, error)
	Build(ctx context.Context, w Workdir, builder Builder) (Artifact, error)
	Pack(ctx context.Context, a Artifact) (Packed, error)
	InvokeHint(artifactRel string) InvokeHint
	DefaultTemplate() string
}

type Images struct {
	GoSandbox   string
	NodeSandbox string
	GoBuilder   string
}

type Builder interface {
	BuildGo(ctx context.Context, workdir, output string) (log string, err error)
}

type Registry struct {
	mu   sync.RWMutex
	impl map[model.RuntimeName]Runtime
}

func NewRegistry() *Registry {
	return &Registry{impl: map[model.RuntimeName]Runtime{}}
}

func (r *Registry) Register(rt Runtime) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.impl[rt.Name()] = rt
}

func (r *Registry) Get(name model.RuntimeName) (Runtime, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rt, ok := r.impl[name]
	if !ok {
		return nil, model.Invalid(fmt.Sprintf("unsupported runtime %q", name))
	}
	return rt, nil
}

func (r *Registry) All() []Runtime {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Runtime, 0, len(r.impl))
	for _, v := range r.impl {
		out = append(out, v)
	}
	return out
}
