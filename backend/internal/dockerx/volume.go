package dockerx

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

func (c *Client) VolumeMountpoint(ctx context.Context, name string) (string, error) {
	vol, err := c.cli.VolumeInspect(ctx, name)
	if err != nil {
		return "", fmt.Errorf("inspect volume %s: %w", name, err)
	}
	if strings.TrimSpace(vol.Mountpoint) == "" {
		return "", fmt.Errorf("volume %s has empty mountpoint", name)
	}
	return vol.Mountpoint, nil
}

func JoinHost(mountpoint string, elem ...string) string {
	parts := make([]string, 0, 1+len(elem))
	parts = append(parts, mountpoint)
	parts = append(parts, elem...)
	return filepath.ToSlash(filepath.Join(parts...))
}
