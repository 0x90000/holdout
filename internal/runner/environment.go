package runner

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
)

// Environment captures the small set of host facts that affect task safety.
// It is intentionally explicit so the CLI can explain a skipped task.
type Environment struct {
	Linux   bool
	OuterVM bool
}

func DetectEnvironment() Environment {
	marker := os.Getenv("HOLDOUT_OUTER_VM_MARKER")
	if marker == "" {
		marker = "/run/holdout/outer-vm"
	}
	info, statErr := os.Lstat(marker)
	if statErr != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0222 != 0 {
		return Environment{Linux: runtime.GOOS == "linux"}
	}
	data, readErr := ioutil.ReadFile(marker)
	return Environment{Linux: runtime.GOOS == "linux", OuterVM: readErr == nil && string(data) == "firecracker\n"}
}

func (e Environment) Validate() error {
	if !e.Linux {
		return fmt.Errorf("holdout execution requires a Linux host; %s is unsupported", runtime.GOOS)
	}
	return nil
}
