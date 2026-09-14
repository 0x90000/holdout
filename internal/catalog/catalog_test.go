package catalog

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/holdout-bench/holdout/internal/model"
)

func TestLoadAndValidateRepositoryCatalog(t *testing.T) {
	root := filepath.Join("..", "..")
	tasks, err := Load(filepath.Join(root, "tasks", "catalog.yaml"), "smoke-v0")
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 12 {
		t.Fatalf("want 12 tasks, got %d", len(tasks))
	}
	if err := Validate(root, tasks); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRequiresDangerousGateForHighImpactTasks(t *testing.T) {
	root := t.TempDir()
	tasks := []model.Task{{ID: "T900", Family: "F", Title: "unsafe", Level: model.L4}}
	if err := Validate(root, tasks); err == nil {
		t.Fatal("expected L4 task without dangerous marker to be rejected")
	}

	tasks[0].Level = model.L3
	tasks[0].Family = "G"
	if err := Validate(root, tasks); err == nil {
		t.Fatal("expected G-family task without dangerous marker to be rejected")
	}
}

func TestDigestNormalizesTextLineEndings(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "tasks", "T900")
	if err := os.MkdirAll(filepath.Join(taskDir, "attacker"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"TASK.md", "oracle.py", "attacker/run.py", "severity.yaml"} {
		path := filepath.Join(taskDir, file)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := ioutil.WriteFile(path, []byte("one\r\ntwo\r\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	first, err := Digest(root, "T900")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(taskDir, "TASK.md")
	if err := ioutil.WriteFile(path, []byte("one\ntwo\n"), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := Digest(root, "T900")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("line ending normalization changed digest: %s != %s", first, second)
	}
}

func TestLoadRejectsNonExactVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.yaml")
	contents := "suite: public-v0\ntasks:\n  - id: T900\n    version: 1x\n"
	if err := ioutil.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path, "public-v0"); err == nil {
		t.Fatal("expected malformed task version to be rejected")
	}
}

func TestValidateRejectsUnsafeTaskID(t *testing.T) {
	if err := Validate(t.TempDir(), []model.Task{{ID: "../escape", Family: "A", Title: "bad", Level: model.L1, Version: 1, DigestSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}}); err == nil {
		t.Fatal("expected path-like task id to be rejected")
	}
}

func TestDigestRejectsMaterialSymlink(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "tasks", "T900")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "outside.txt")
	if err := ioutil.WriteFile(target, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(taskDir, "TASK.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := Digest(root, "T900"); err == nil {
		t.Fatal("expected task material symlink to be rejected")
	}
}
