package dockerx

import "testing"

func TestSandboxHostConfigEnforcesNFR08(t *testing.T) {
	hc, err := SandboxHostConfig(128, 500_000_000, "/host/sock", "/host/art")
	if err != nil {
		t.Fatal(err)
	}
	if hc.Privileged {
		t.Fatal("privileged must be false")
	}
	if !hc.ReadonlyRootfs {
		t.Fatal("rootfs must be ro")
	}
	if string(hc.NetworkMode) != "none" {
		t.Fatalf("network %s", hc.NetworkMode)
	}
	if len(hc.CapDrop) == 0 || hc.CapDrop[0] != "ALL" {
		t.Fatal("cap-drop ALL")
	}
	found := false
	for _, s := range hc.SecurityOpt {
		if s == "no-new-privileges:true" {
			found = true
		}
	}
	if !found {
		t.Fatal("no-new-privileges")
	}
	if hc.Resources.PidsLimit == nil || *hc.Resources.PidsLimit != 64 {
		t.Fatal("pids-limit 64")
	}
	if hc.Resources.Memory != 128*1024*1024 {
		t.Fatal("memory")
	}
	if _, ok := hc.Tmpfs["/tmp"]; !ok {
		t.Fatal("tmpfs /tmp")
	}
}

func TestSandboxMemoryRange(t *testing.T) {
	if _, err := SandboxHostConfig(32, 1, "a", "b"); err == nil {
		t.Fatal("32mb should fail")
	}
}

func TestBuilderNotPrivileged(t *testing.T) {
	hc := BuilderHostConfig("/work")
	if hc.Privileged {
		t.Fatal("builder privileged")
	}
}
