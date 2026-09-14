package sut

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/holdout-bench/holdout/internal/model"
)

type CreateSpec struct {
	Name    string
	Image   string
	TaskDir string
	Env     map[string]string
	Network string
}

type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Adapter interface {
	Create(context.Context, CreateSpec) error
	Exec(context.Context, string, time.Duration) (ExecResult, error)
	Put(context.Context, string, string) error
	Get(context.Context, string, string) error
	NetMode() string
	Describe(context.Context) (model.SutDescription, error)
	Destroy(context.Context) error
}

type DockerAdapter struct {
	binary  string
	image   string
	name    string
	created bool
	net     string
	runtime string
	product string
}

func NewDocker(image string) (*DockerAdapter, error) {
	return newDocker(image, "", "docker-runc")
}

func NewGVisor(image string) (*DockerAdapter, error) {
	return newDocker(image, "runsc", "gvisor")
}

func newDocker(image, runtimeName, product string) (*DockerAdapter, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("Docker SUT adapters require a Linux host; %s is unsupported", runtime.GOOS)
	}
	binary, err := exec.LookPath("docker")
	if err != nil {
		return nil, fmt.Errorf("docker is not installed or not on PATH: %w", err)
	}
	if image == "" {
		return nil, fmt.Errorf("a preloaded image is required; refusing implicit registry pulls")
	}
	return &DockerAdapter{binary: binary, image: image, net: "default-allow", runtime: runtimeName, product: product}, nil
}

func (d *DockerAdapter) run(ctx context.Context, args ...string) (ExecResult, error) {
	cmd := exec.CommandContext(ctx, d.binary, args...)
	out, err := cmd.Output()
	result := ExecResult{Stdout: string(out)}
	if err == nil {
		return result, nil
	}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.Stderr = string(exitErr.Stderr)
		if exitErr.ProcessState != nil {
			result.ExitCode = exitErr.ProcessState.ExitCode()
		}
	}
	return result, err
}

func (d *DockerAdapter) Create(ctx context.Context, spec CreateSpec) error {
	if d.created || d.name != "" {
		return fmt.Errorf("adapter already has a container")
	}
	d.name = spec.Name
	args := []string{"create", "--name", d.name, "--workdir", "/workspace"}
	if d.runtime != "" {
		args = append(args, "--runtime", d.runtime)
	}
	if spec.Network != "" {
		args = append(args, "--network", spec.Network)
	}
	if spec.TaskDir != "" {
		abs, err := filepath.Abs(spec.TaskDir)
		if err != nil {
			return err
		}
		args = append(args, "-v", abs+":/task:ro")
	}
	args = append(args, "-v", "holdout-workspace-"+d.name+":/workspace")
	for key, value := range spec.Env {
		args = append(args, "-e", key+"="+value)
	}
	args = append(args, d.image, "sleep", "3600")
	if _, err := d.run(ctx, args...); err != nil {
		d.name = ""
		return fmt.Errorf("docker create: %w", err)
	}
	d.created = true
	if _, err := d.run(ctx, "start", d.name); err != nil {
		_, _ = d.run(context.Background(), "rm", "-f", d.name)
		_, _ = d.run(context.Background(), "volume", "rm", "holdout-workspace-"+d.name)
		d.name = ""
		d.created = false
		return fmt.Errorf("docker start: %w", err)
	}
	return nil
}

func (d *DockerAdapter) Exec(ctx context.Context, command string, timeout time.Duration) (ExecResult, error) {
	if d.name == "" {
		return ExecResult{}, fmt.Errorf("adapter is not created")
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	result, err := d.run(ctx, "exec", d.name, "sh", "-lc", command)
	if err != nil && ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, err
}

func (d *DockerAdapter) Put(ctx context.Context, local, remote string) error {
	_, err := d.run(ctx, "cp", local, d.name+":"+remote)
	return err
}

func (d *DockerAdapter) Get(ctx context.Context, remote, local string) error {
	_, err := d.run(ctx, "cp", d.name+":"+remote, local)
	return err
}

func (d *DockerAdapter) NetMode() string { return d.net }

func (d *DockerAdapter) Describe(ctx context.Context) (model.SutDescription, error) {
	version, err := d.run(ctx, "version", "--format", "{{.Server.Version}}")
	if err != nil {
		return model.SutDescription{}, err
	}
	runtimeResult, err := d.run(ctx, "info", "--format", "{{.DefaultRuntime}}")
	if err != nil {
		return model.SutDescription{}, fmt.Errorf("docker info: %w", err)
	}
	kernel := "unknown"
	if output, kernelErr := exec.CommandContext(ctx, "uname", "-r").Output(); kernelErr == nil {
		kernel = strings.TrimSpace(string(output))
	}
	configuredRuntime := d.runtime
	if configuredRuntime == "" {
		configuredRuntime = strings.TrimSpace(runtimeResult.Stdout)
	}
	if configuredRuntime == "" {
		configuredRuntime = "runc"
	}
	imageResult, imageErr := d.run(ctx, "image", "inspect", "--format", "{{.Id}}", d.image)
	if imageErr != nil {
		return model.SutDescription{}, fmt.Errorf("docker image inspect %s: %w", d.image, imageErr)
	}
	imageDigest := strings.TrimSpace(imageResult.Stdout)
	if imageDigest == "" {
		return model.SutDescription{}, fmt.Errorf("docker image inspect %s returned an empty digest", d.image)
	}
	config := d.image + "\n" + imageDigest + "\n" + d.net + "\n" + configuredRuntime
	digest := sha256.Sum256([]byte(config))
	return model.SutDescription{
		Name: d.product, Version: strings.TrimSpace(version.Stdout), ImageDigest: imageDigest, AdapterVersion: "0.1.0",
		ConfigSHA256: hex.EncodeToString(digest[:]), Kernel: kernel,
		Runtime: configuredRuntime, Architecture: runtime.GOARCH, NetMode: d.net,
	}, nil
}

func (d *DockerAdapter) Destroy(ctx context.Context) error {
	if d.name == "" || !d.created {
		return nil
	}
	_, containerErr := d.run(ctx, "rm", "-f", d.name)
	_, volumeErr := d.run(ctx, "volume", "rm", "holdout-workspace-"+d.name)
	if containerErr != nil {
		return containerErr
	}
	d.name = ""
	d.created = false
	return volumeErr
}
