package catalog

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/holdout-bench/holdout/internal/model"
)

// Load parses the deliberately small catalog format used by the repository.
// Keeping this parser dependency-free makes the CLI reproducible offline.
func Load(path, suite string) ([]model.Task, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var tasks []model.Task
	var current *model.Task
	declaredSuite := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "suite:") {
			declaredSuite = strings.TrimSpace(strings.TrimPrefix(line, "suite:"))
			continue
		}
		if strings.HasPrefix(line, "- id:") {
			if current != nil {
				tasks = append(tasks, *current)
			}
			current = &model.Task{ID: strings.TrimSpace(strings.TrimPrefix(line, "- id:"))}
			continue
		}
		if current == nil {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"'")
		switch key {
		case "family":
			current.Family = value
		case "version":
			parsed, scanErr := strconv.Atoi(value)
			if scanErr != nil {
				return nil, fmt.Errorf("task %s has invalid version %q", current.ID, value)
			}
			current.Version = parsed
		case "sha256", "digest_sha256":
			current.DigestSHA256 = value
		case "title":
			current.Title = value
		case "level":
			current.Level = model.Level(value)
		case "dangerous":
			current.Dangerous = value == "true"
		case "description":
			current.Description = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if current != nil {
		tasks = append(tasks, *current)
	}
	if declaredSuite != "" && suite != "" && suite != declaredSuite && !(suite == "smoke-v0" && declaredSuite == "public-v0") {
		return nil, fmt.Errorf("suite %q is not available (catalog: %q)", suite, declaredSuite)
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("catalog %s contains no tasks", path)
	}
	return tasks, nil
}

func Validate(root string, tasks []model.Task) error {
	seen := make(map[string]bool)
	for _, task := range tasks {
		if task.ID == "" || seen[task.ID] {
			return fmt.Errorf("invalid or duplicate task id %q", task.ID)
		}
		if !validID(task.ID) {
			return fmt.Errorf("task %s has unsafe id; use letters, digits, '.', '_' or '-'", task.ID)
		}
		seen[task.ID] = true
		if task.Family == "" || task.Title == "" || task.Level == "" {
			return fmt.Errorf("task %s is missing family, title, or level", task.ID)
		}
		if !validLevel(task.Level) {
			return fmt.Errorf("task %s has unknown level %q", task.ID, task.Level)
		}
		if task.Version < 1 {
			return fmt.Errorf("task %s must declare a positive version", task.ID)
		}
		if len(task.DigestSHA256) != 64 {
			return fmt.Errorf("task %s must declare a 64-character sha256 digest", task.ID)
		}
		if _, err := hex.DecodeString(task.DigestSHA256); err != nil {
			return fmt.Errorf("task %s has invalid sha256 digest: %w", task.ID, err)
		}
		if (task.Level == model.L4 || task.Level == model.L5 || task.Family == "G") && !task.Dangerous {
			return fmt.Errorf("task %s at %s or in family G must be marked dangerous", task.ID, task.Level)
		}
		dir := filepath.Join(root, "tasks", task.ID)
		for _, required := range []string{"TASK.md", "oracle.py", "attacker", "severity.yaml"} {
			if _, err := os.Stat(filepath.Join(dir, required)); err != nil {
				return fmt.Errorf("task %s missing %s: %w", task.ID, required, err)
			}
		}
		digest, err := Digest(root, task.ID)
		if err != nil {
			return fmt.Errorf("task %s digest: %w", task.ID, err)
		}
		if !strings.EqualFold(task.DigestSHA256, digest) {
			return fmt.Errorf("task %s digest mismatch: catalog %s, files %s", task.ID, task.DigestSHA256, digest)
		}
	}
	return nil
}

func validID(value string) bool {
	for i, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (i > 0 && (r == '.' || r == '_' || r == '-')) {
			continue
		}
		return false
	}
	return value != ""
}

// ValidateResult verifies that an archived result covers exactly the task
// catalog selected by its suite and that each material digest is unchanged.
func ValidateResult(root string, result model.RunResult) error {
	tasks, err := Load(filepath.Join(root, "tasks", "catalog.yaml"), result.Suite)
	if err != nil {
		return err
	}
	if err := Validate(root, tasks); err != nil {
		return err
	}
	expected := make(map[string]model.Task, len(tasks))
	for _, task := range tasks {
		expected[task.ID] = task
	}
	if len(result.Tasks) != len(expected) {
		return fmt.Errorf("result has %d tasks, catalog requires %d", len(result.Tasks), len(expected))
	}
	seen := make(map[string]bool, len(result.Tasks))
	for _, got := range result.Tasks {
		want, ok := expected[got.ID]
		if !ok {
			return fmt.Errorf("result contains task %s not present in suite %s", got.ID, result.Suite)
		}
		if seen[got.ID] {
			return fmt.Errorf("result contains duplicate task %s", got.ID)
		}
		seen[got.ID] = true
		if got.Version != want.Version || !strings.EqualFold(got.DigestSHA256, want.DigestSHA256) {
			return fmt.Errorf("task %s version or digest does not match catalog", got.ID)
		}
	}
	return nil
}

func validLevel(level model.Level) bool {
	switch level {
	case model.L0, model.L1, model.L2, model.L3, model.L4, model.L5:
		return true
	default:
		return false
	}
}

// Digest returns a stable hash of task materials. Paths are sorted and encoded
// with a forward slash so the same task has the same digest on Windows and
// Linux. Generated caches are excluded from the digest.
func Digest(root, taskID string) (string, error) {
	taskRoot := filepath.Join(root, "tasks", taskID)
	type fileEntry struct {
		path string
		data []byte
	}
	var entries []fileEntry
	err := filepath.Walk(taskRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			if info.Name() == "__pycache__" || strings.HasPrefix(info.Name(), ".") && path != taskRoot {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("task material symlink is not allowed: %s", filepath.ToSlash(path))
		}
		rel, err := filepath.Rel(taskRoot, path)
		if err != nil {
			return err
		}
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		// Text task materials are normalized so an archive checked out with
		// CRLF has the same digest as the LF form used by Linux CI. Binary
		// fixtures are hashed byte-for-byte.
		if isTextMaterial(rel) {
			data = []byte(strings.ReplaceAll(string(data), "\r\n", "\n"))
		}
		entries = append(entries, fileEntry{path: filepath.ToSlash(rel), data: data})
		return nil
	})
	if err != nil {
		return "", err
	}
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].path < entries[j-1].path; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
	h := sha256.New()
	for _, entry := range entries {
		fmt.Fprintf(h, "%s\x00%d\x00", entry.path, len(entry.data))
		_, _ = h.Write(entry.data)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func isTextMaterial(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".py", ".yaml", ".yml", ".json", ".sh", ".txt":
		return true
	default:
		return false
	}
}
