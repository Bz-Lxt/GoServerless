package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Manager struct {
	Root string
}

func New(root string) *Manager { return &Manager{Root: root} }

func (m *Manager) VersionDir(name string, version int) string {
	return filepath.Join(m.Root, name, fmt.Sprintf("v%d", version))
}

func (m *Manager) BuildDir(name string, version int) string {
	return filepath.Join(m.Root, "_build", name, fmt.Sprintf("v%d", version))
}

func (m *Manager) ContainerPath(name string, version int, file string) string {
	p := filepath.ToSlash(filepath.Join("/artifacts", name, fmt.Sprintf("v%d", version)))
	if file == "" {
		return p
	}
	return p + "/" + strings.TrimPrefix(file, "/")
}

func (m *Manager) Ensure(name string, version int) (string, error) {
	dir := m.VersionDir(name, version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (m *Manager) RemoveFunction(name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(m.Root, name)); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(m.Root, "_build", name))
}

func (m *Manager) RemoveVersion(name string, version int) error {
	if err := validateName(name); err != nil {
		return err
	}
	return os.RemoveAll(m.VersionDir(name, version))
}

func (m *Manager) Checksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (m *Manager) WriteFile(dir, name string, b []byte, mode os.FileMode) error {
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("unsafe artifact filename")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), b, mode)
}

func validateName(name string) error {
	if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("unsafe function name for artifact path")
	}
	return nil
}
