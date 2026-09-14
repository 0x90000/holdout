package runner

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/holdout-bench/holdout/internal/catalog"
	"github.com/holdout-bench/holdout/internal/model"
	"github.com/holdout-bench/holdout/internal/sink"
	"github.com/holdout-bench/holdout/internal/sut"
)

type Options struct {
	Root           string
	Suite          string
	SUT            string
	Image          string
	AllowDangerous bool
	DryRun         bool
}

func Run(ctx context.Context, options Options) (result model.RunResult, runErr error) {
	started := time.Now().UTC()
	canonicalSUT := options.SUT
	if canonicalSUT == "docker" {
		canonicalSUT = "docker-runc"
	}
	placeholder := sha256.Sum256([]byte("unavailable"))
	result = model.RunResult{
		Schema: "https://holdout.dev/schema/result.v1.json",
		RunID:  newRunID(), Suite: options.Suite, Started: started,
		SUT: model.SutDescription{
			Name: canonicalSUT, Version: "unavailable", ImageDigest: "unavailable",
			AdapterVersion: "0.1.0", ConfigSHA256: hex.EncodeToString(placeholder[:]),
			Kernel: "unavailable", Runtime: "unavailable", Architecture: "unavailable", NetMode: "unknown",
		},
		Tasks: []model.TaskResult{},
	}
	defer func() {
		if result.RunID != "" && result.Ended.IsZero() {
			result.Ended = time.Now().UTC()
			result.Summary = model.Score(result.Tasks)
		}
		if runErr != nil && result.RunID != "" {
			result.Error = sanitizeError(runErr.Error())
		}
	}()
	tasks, err := catalog.Load(filepath.Join(options.Root, "tasks", "catalog.yaml"), options.Suite)
	if err != nil {
		return result, err
	}
	if err := catalog.Validate(options.Root, tasks); err != nil {
		return result, err
	}
	sutName := options.SUT
	if sutName == "docker" {
		sutName = "docker-runc"
		result.SUT.Name = sutName
	}
	if sutName != "docker-runc" && sutName != "gvisor" {
		return result, fmt.Errorf("unknown SUT %q; available: docker-runc, gvisor", options.SUT)
	}
	environment := DetectEnvironment()
	if !options.DryRun {
		if err := environment.Validate(); err != nil {
			return result, err
		}
	}
	outerVM := environment.OuterVM
	if !options.DryRun {
		adapter, adapterErr := newAdapter(sutName, options.Image)
		if adapterErr != nil {
			return result, adapterErr
		}
		describeCtx, describeCancel := context.WithTimeout(ctx, 30*time.Second)
		description, describeErr := adapter.Describe(describeCtx)
		describeCancel()
		if describeErr != nil {
			return result, fmt.Errorf("describe %s: %w", sutName, describeErr)
		}
		result.SUT = description
	} else {
		digest := sha256.Sum256([]byte("dry-run:" + options.SUT + ":" + options.Suite))
		result.SUT.ConfigSHA256 = hex.EncodeToString(digest[:])
		result.SUT.ImageDigest = "unavailable"
	}

	var sinkServer *sink.Server
	if !options.DryRun {
		bindAddress := os.Getenv("HOLDOUT_SINK_BIND_ADDR")
		if bindAddress == "" {
			return result, fmt.Errorf("HOLDOUT_SINK_BIND_ADDR must be set to the private guest address for container reachability")
		}
		bindHost, _, splitErr := net.SplitHostPort(bindAddress)
		if splitErr != nil || net.ParseIP(bindHost) == nil || net.ParseIP(bindHost).IsLoopback() {
			return result, fmt.Errorf("HOLDOUT_SINK_BIND_ADDR must use a non-loopback private IP")
		}
		sinkServer, err = sink.StartOn(bindAddress)
		if err != nil {
			return result, fmt.Errorf("start sink: %w", err)
		}
		defer sinkServer.Close()
	}

	for _, task := range tasks {
		begin := time.Now()
		if task.Dangerous && (!outerVM || !options.AllowDangerous) {
			result.Tasks = append(result.Tasks, model.TaskResult{ID: task.ID, Version: task.Version, DigestSHA256: task.DigestSHA256, Status: model.StatusSkipped, MaxLevel: model.L0, DurationMS: time.Since(begin).Milliseconds(), Error: "dangerous task requires a Firecracker outer VM marker and explicit authorization"})
			continue
		}
		if options.DryRun {
			result.Tasks = append(result.Tasks, model.TaskResult{ID: task.ID, Version: task.Version, DigestSHA256: task.DigestSHA256, Status: model.StatusSkipped, MaxLevel: model.L0, DurationMS: time.Since(begin).Milliseconds(), Error: "dry-run"})
			continue
		}

		adapter, adapterErr := newAdapter(sutName, options.Image)
		if adapterErr != nil {
			return result, adapterErr
		}
		taskRunID := result.RunID + "-" + strings.ToLower(task.ID)
		if err := sinkServer.RegisterRun(taskRunID); err != nil {
			return result, fmt.Errorf("register sink run %s: %w", task.ID, err)
		}
		containerName := "holdout-" + safeID(taskRunID)
		containerSinkURL := sinkServer.ContainerURL()
		env := map[string]string{
			"HOLDOUT_RUN_ID":       taskRunID,
			"HOLDOUT_SINK_URL":     containerSinkURL,
			"HOLDOUT_IMDS_URL":     containerSinkURL + "/events?run_id=" + taskRunID,
			"HOLDOUT_NEIGHBOR_URL": containerSinkURL + "/events?run_id=" + taskRunID,
			"HOLDOUT_TEST_SECRET":  "holdout-synthetic-secret",
		}
		createCtx, createCancel := context.WithTimeout(ctx, 30*time.Second)
		createErr := adapter.Create(createCtx, sut.CreateSpec{Name: containerName, Image: options.Image, TaskDir: filepath.Join(options.Root, "tasks", task.ID), Env: env})
		createCancel()
		if createErr != nil {
			errText := sanitizeError(createErr.Error())
			cleanupText := ""
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
			cleanupErr := adapter.Destroy(cleanupCtx)
			cleanupCancel()
			if cleanupErr != nil {
				cleanupText = sanitizeError(cleanupErr.Error())
			}
			result.Tasks = append(result.Tasks, model.TaskResult{ID: task.ID, Version: task.Version, DigestSHA256: task.DigestSHA256, Status: model.StatusError, MaxLevel: model.L0, DurationMS: time.Since(begin).Milliseconds(), Error: errText, CleanupError: cleanupText})
			continue
		}
		execResult, execErr := adapter.Exec(ctx, "python3 /task/attacker/run.py", 30*time.Second)
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		destroyErr := adapter.Destroy(cleanupCtx)
		cleanupCancel()
		events := sinkServer.Events(taskRunID)
		if execErr != nil {
			errText := sanitizeError(execErr.Error())
			cleanupText := ""
			if destroyErr != nil {
				cleanupText = sanitizeError(destroyErr.Error())
			}
			evidence := []model.Evidence{{Kind: "exit_code", Value: execResult.ExitCode}, {Kind: "sink_event_count", Value: len(events)}}
			if errors.Is(execErr, context.DeadlineExceeded) {
				evidence = append(evidence, model.Evidence{Kind: "timeout", Value: true})
			}
			result.Tasks = append(result.Tasks, model.TaskResult{ID: task.ID, Version: task.Version, DigestSHA256: task.DigestSHA256, Status: model.StatusError, MaxLevel: model.L0, DurationMS: time.Since(begin).Milliseconds(), Evidence: evidence, Error: errText, CleanupError: cleanupText})
			continue
		}

		cleanupText := ""
		if destroyErr != nil {
			cleanupText = sanitizeError(destroyErr.Error())
		}
		taskResult := model.TaskResult{ID: task.ID, Version: task.Version, DigestSHA256: task.DigestSHA256, Status: model.StatusHold, MaxLevel: model.L0, DurationMS: time.Since(begin).Milliseconds(), Evidence: []model.Evidence{{Kind: "sink_event_count", Value: len(events)}}, CleanupError: cleanupText}
		oracle, oracleErr := runOracle(ctx, filepath.Join(options.Root, "tasks", task.ID, "oracle.py"), events, task.Level)
		if oracleErr != nil {
			taskResult.Status = model.StatusError
			taskResult.Error = sanitizeError(oracleErr.Error())
		} else {
			taskResult.MaxLevel = model.Level(oracle.MaxLevel)
			taskResult.Status = model.Status(oracle.Status)
		}
		result.Tasks = append(result.Tasks, taskResult)
	}
	result.Ended = time.Now().UTC()
	result.Summary = model.Score(result.Tasks)
	return result, nil
}

func newAdapter(name, image string) (sut.Adapter, error) {
	switch name {
	case "docker-runc":
		return sut.NewDocker(image)
	case "gvisor":
		return sut.NewGVisor(image)
	default:
		return nil, fmt.Errorf("unknown SUT %q", name)
	}
}

type oracleOutput struct {
	Status   string `json:"status"`
	MaxLevel string `json:"max_level"`
}

func runOracle(parent context.Context, path string, events []sink.Event, expectedLevel model.Level) (oracleOutput, error) {
	input, err := json.Marshal(map[string]interface{}{"sink_event_count": len(events), "events": events})
	if err != nil {
		return oracleOutput{}, err
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", path)
	cmd.Stdin = bytes.NewReader(input)
	output, err := cmd.Output()
	if err != nil {
		return oracleOutput{}, fmt.Errorf("oracle %s: %w", filepath.Base(filepath.Dir(path)), err)
	}
	var result oracleOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return oracleOutput{}, fmt.Errorf("oracle %s returned invalid JSON: %w", filepath.Base(filepath.Dir(path)), err)
	}
	if result.Status == "" || result.MaxLevel == "" {
		return oracleOutput{}, fmt.Errorf("oracle %s returned incomplete result", filepath.Base(filepath.Dir(path)))
	}
	validStatuses := map[string]bool{"hold": true, "bypass": true, "exfil": true, "lateral": true, "escape": true}
	validLevels := map[string]bool{"L0": true, "L1": true, "L2": true, "L3": true, "L4": true, "L5": true}
	if !validStatuses[result.Status] || !validLevels[result.MaxLevel] {
		return oracleOutput{}, fmt.Errorf("oracle %s returned unknown status or level", filepath.Base(filepath.Dir(path)))
	}
	if len(events) == 0 && (result.Status != string(model.StatusHold) || result.MaxLevel != string(model.L0)) {
		return oracleOutput{}, fmt.Errorf("oracle %s reported a result without sink evidence", filepath.Base(filepath.Dir(path)))
	}
	if result.Status == string(model.StatusHold) && result.MaxLevel != string(model.L0) {
		return oracleOutput{}, fmt.Errorf("oracle %s returned hold with non-zero level", filepath.Base(filepath.Dir(path)))
	}
	if result.Status != string(model.StatusHold) && (result.MaxLevel != string(expectedLevel) || result.Status != string(statusForLevel(expectedLevel))) {
		return oracleOutput{}, fmt.Errorf("oracle %s returned inconsistent level or status", filepath.Base(filepath.Dir(path)))
	}
	return result, nil
}

func Write(path string, result model.RunResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	dir := filepath.Dir(path)
	tmp, err := ioutil.TempFile(dir, ".holdout-result-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func statusForLevel(level model.Level) model.Status {
	switch level {
	case model.L1:
		return model.StatusBypass
	case model.L2, model.L3:
		return model.StatusExfil
	case model.L4:
		return model.StatusLateral
	case model.L5:
		return model.StatusEscape
	default:
		return model.StatusHold
	}
}

func newRunID() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return time.Now().UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(buf)
}

func safeID(value string) string {
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, ":", "")
	return value
}

func sanitizeError(value string) string {
	for _, secret := range []string{"holdout-synthetic-secret", "HOLDOUT_TEST_SECRET"} {
		value = strings.ReplaceAll(value, secret, "[redacted]")
	}
	return value
}
