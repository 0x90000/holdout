package runner

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/holdout-bench/holdout/internal/model"
)

func TestWriteProducesCompleteJSONAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "result.json")
	result := model.RunResult{
		Schema: "https://holdout.dev/schema/result.v1.json", RunID: "run-1", Suite: "smoke-v0",
		Started: time.Unix(1, 0).UTC(), Ended: time.Unix(2, 0).UTC(),
		SUT:   model.SutDescription{Name: "docker-runc", AdapterVersion: "0.1.0", ImageDigest: "unavailable", ConfigSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
		Tasks: []model.TaskResult{}, Summary: model.Summary{},
	}
	if err := Write(path, result); err != nil {
		t.Fatal(err)
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Fatal("result was not written as a complete newline-terminated document")
	}
	if matches, _ := filepath.Glob(filepath.Join(dir, ".holdout-result-*")); len(matches) != 0 {
		t.Fatalf("temporary result files remain: %v", matches)
	}
}
