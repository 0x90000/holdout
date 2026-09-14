package runner

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestEnvironmentRejectsNonLinuxExecution(t *testing.T) {
	if err := (Environment{Linux: false}).Validate(); err == nil {
		t.Fatal("expected non-Linux execution to be rejected")
	}
}

func TestEnvironmentAllowsLinuxExecution(t *testing.T) {
	if err := (Environment{Linux: true}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDetectEnvironmentRequiresExactOuterVMMarker(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "outer-vm")
	if err := ioutil.WriteFile(marker, []byte("firecracker\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(marker, 0444); err != nil {
		t.Fatal(err)
	}
	previous := os.Getenv("HOLDOUT_OUTER_VM_MARKER")
	defer os.Setenv("HOLDOUT_OUTER_VM_MARKER", previous)
	if err := os.Setenv("HOLDOUT_OUTER_VM_MARKER", marker); err != nil {
		t.Fatal(err)
	}
	if !DetectEnvironment().OuterVM {
		t.Fatal("expected exact Firecracker marker to be accepted")
	}
	if err := os.Chmod(marker, 0666); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(marker, []byte("firecracker"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(marker, 0444); err != nil {
		t.Fatal(err)
	}
	if DetectEnvironment().OuterVM {
		t.Fatal("expected marker without newline to be rejected")
	}
	if err := os.Chmod(marker, 0666); err != nil {
		t.Fatal(err)
	}
	if DetectEnvironment().OuterVM {
		t.Fatal("expected writable marker to be rejected")
	}
	if err := os.Chmod(marker, 0444); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(marker, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := os.Setenv("HOLDOUT_OUTER_VM_MARKER", link); err != nil {
		t.Fatal(err)
	}
	if DetectEnvironment().OuterVM {
		t.Fatal("expected symlink marker to be rejected")
	}
}
