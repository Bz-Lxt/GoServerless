package dockerx

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"

	"github.com/gogo/goserverless/internal/logger"
)

const (
	LabelManaged = "goserverless.managed"
	LabelRole    = "goserverless.role"
	LabelRuntime = "goserverless.runtime"
	LabelSlot    = "goserverless.slot"
)

type Client struct {
	cli *client.Client
}

func New(host string) (*Client, error) {
	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}
	if host != "" {
		opts = append(opts, client.WithHost(host))
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &Client{cli: cli}, nil
}

func (c *Client) Close() error {
	if c == nil || c.cli == nil {
		return nil
	}
	return c.cli.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.cli.Ping(ctx)
	return err
}

func (c *Client) HasImage(ctx context.Context, ref string) error {
	_, err := c.cli.ImageInspect(ctx, ref)
	if err != nil {
		return fmt.Errorf("image %s missing: %w", ref, err)
	}
	return nil
}

func (c *Client) EnsureDirs(paths ...string) error {
	for _, p := range paths {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return err
		}
	}
	return nil
}

type CreateSandboxOpts struct {
	Name       string
	Image      string
	Runtime    string
	SlotID     string
	MemoryMB   int
	CPUNano    int64
	SocketDir  string
	ArtifactDir string
}

func (c *Client) CreateSandbox(ctx context.Context, opt CreateSandboxOpts) (string, error) {
	if err := os.MkdirAll(opt.SocketDir, 0o777); err != nil {
		return "", err
	}
	_ = os.Chmod(opt.SocketDir, 0o777)
	hc, err := SandboxHostConfig(opt.MemoryMB, opt.CPUNano, opt.SocketDir, opt.ArtifactDir)
	if err != nil {
		return "", err
	}
	cfg := &container.Config{
		Image: opt.Image,
		User:  "65534:65534",
		Env: []string{
			"TZ=Asia/Shanghai",
			"SOCKET_PATH=/run/gscf/agent.sock",
		},
		Labels: map[string]string{
			LabelManaged: "true",
			LabelRole:    "sandbox",
			LabelRuntime: opt.Runtime,
			LabelSlot:    opt.SlotID,
		},
		Entrypoint: []string{"/usr/local/bin/sandbox-agent"},
	}
	resp, err := c.cli.ContainerCreate(ctx, cfg, hc, nil, nil, opt.Name)
	if err != nil {
		return "", fmt.Errorf("create sandbox: %w", err)
	}
	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		_ = c.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return "", fmt.Errorf("start sandbox: %w", err)
	}
	return resp.ID, nil
}

func (c *Client) RemoveContainer(ctx context.Context, id string) error {
	err := c.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: true, RemoveVolumes: true})
	if err != nil && !isNotFound(err) {
		return err
	}
	return nil
}

func (c *Client) Kill(ctx context.Context, id string) error {
	err := c.cli.ContainerKill(ctx, id, "KILL")
	if err != nil && !isNotFound(err) {
		return err
	}
	return nil
}

func (c *Client) RunBuilder(ctx context.Context, image, workHost, outputName string) (string, error) {
	name := "gscf-builder-" + strings.ReplaceAll(fmt.Sprintf("%d", time.Now().UnixNano()), " ", "")
	cfg := &container.Config{
		Image: image,
		User:  "0:0",
		Env: []string{
			"TZ=Asia/Shanghai",
			"CGO_ENABLED=0",
			"GOOS=linux",
			"GOPROXY=https://goproxy.cn,direct",
		},
		WorkingDir: "/work",
		Cmd: []string{
			"go", "build", "-trimpath", "-ldflags=-s -w", "-o", "/work/" + outputName, ".",
		},
		Labels: map[string]string{
			LabelManaged: "true",
			LabelRole:    "builder",
		},
	}
	hc := BuilderHostConfig(workHost)
	resp, err := c.cli.ContainerCreate(ctx, cfg, hc, nil, nil, name)
	if err != nil {
		return "", fmt.Errorf("create builder: %w", err)
	}
	defer func() {
		_ = c.cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})
	}()
	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("start builder: %w", err)
	}
	statusCh, errCh := c.cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			logs := c.logs(ctx, resp.ID)
			return logs, fmt.Errorf("builder wait: %w", err)
		}
	case st := <-statusCh:
		logs := c.logs(ctx, resp.ID)
		if st.StatusCode != 0 {
			return logs, fmt.Errorf("go build exited %d", st.StatusCode)
		}
		return logs, nil
	case <-ctx.Done():
		_ = c.cli.ContainerKill(context.Background(), resp.ID, "KILL")
		return c.logs(context.Background(), resp.ID), ctx.Err()
	}
	return "", nil
}

func (c *Client) logs(ctx context.Context, id string) string {
	r, err := c.cli.ContainerLogs(ctx, id, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return err.Error()
	}
	defer r.Close()
	b, _ := io.ReadAll(r)
	return stripDockerMux(b)
}

func (c *Client) ReapOrphans(ctx context.Context) {
	f := filters.NewArgs()
	f.Add("label", LabelManaged+"=true")
	list, err := c.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: f})
	if err != nil {
		logger.Warn(ctx, "list managed containers failed", "err", err)
		return
	}
	for _, ct := range list {
		if ct.Labels[LabelRole] == "builder" {
			_ = c.RemoveContainer(ctx, ct.ID)
		}
	}
}

func (c *Client) ImagePullSilent(ctx context.Context, ref string) error {
	_, err := c.cli.ImageInspect(ctx, ref)
	if err == nil {
		return nil
	}
	rd, err := c.cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, rd)
	return rd.Close()
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "No such container") || strings.Contains(s, "not found")
}

func stripDockerMux(b []byte) string {
	if len(b) < 8 {
		return string(b)
	}
	var out strings.Builder
	i := 0
	for i+8 <= len(b) {
		size := int(b[i+4])<<24 | int(b[i+5])<<16 | int(b[i+6])<<8 | int(b[i+7])
		i += 8
		if size < 0 || i+size > len(b) {
			out.Write(b[i-8:])
			break
		}
		out.Write(b[i : i+size])
		i += size
	}
	return out.String()
}
