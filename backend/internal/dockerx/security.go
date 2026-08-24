package dockerx

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
)

// SandboxHostConfig implements NFR-08 one-to-one. Privileged is never set.
func SandboxHostConfig(memoryMB int, cpuNano int64, socketMount, artifactMount string) (*container.HostConfig, error) {
	if memoryMB < 64 || memoryMB > 512 {
		return nil, fmt.Errorf("memory_mb out of range")
	}
	if cpuNano <= 0 {
		cpuNano = 500_000_000
	}
	mem := int64(memoryMB) * 1024 * 1024
	hc := &container.HostConfig{
		AutoRemove: false,
		Privileged: false,
		ReadonlyRootfs: true,
		NetworkMode:    "none",
		CapDrop:        []string{"ALL"},
		SecurityOpt:    []string{"no-new-privileges:true"},
		Resources: container.Resources{
			Memory:    mem,
			NanoCPUs:  cpuNano,
			PidsLimit: int64Ptr(64),
		},
		Tmpfs: map[string]string{
			"/tmp": "rw,noexec,nosuid,size=64m",
		},
		Binds: []string{
			artifactMount + ":/artifacts:ro",
			socketMount + ":/run/gscf:rw",
		},
	}
	return hc, nil
}

func BuilderHostConfig(workMount string) *container.HostConfig {
	return &container.HostConfig{
		Privileged:  false,
		CapDrop:     []string{"ALL"},
		SecurityOpt: []string{"no-new-privileges:true"},
		Resources: container.Resources{
			Memory:    512 * 1024 * 1024,
			NanoCPUs:  1_000_000_000,
			PidsLimit: int64Ptr(128),
		},
		Binds: []string{
			workMount + ":/work:rw",
		},
		AutoRemove: false,
	}
}

func int64Ptr(v int64) *int64 { return &v }
