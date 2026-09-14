package catalog

import (
	"testing"

	"github.com/holdout-bench/holdout/internal/model"
)

func TestValidateResultRejectsIncompleteSuite(t *testing.T) {
	root := "../.."
	result := model.RunResult{Suite: "smoke-v0", Tasks: nil}
	if err := ValidateResult(root, result); err == nil {
		t.Fatal("expected incomplete result to be rejected")
	}
}

func TestValidateResultRejectsDuplicateTask(t *testing.T) {
	root := "../.."
	tasks, err := Load(root+"/tasks/catalog.yaml", "smoke-v0")
	if err != nil {
		t.Fatal(err)
	}
	result := model.RunResult{Suite: "smoke-v0"}
	for _, task := range tasks {
		result.Tasks = append(result.Tasks, model.TaskResult{ID: task.ID, Version: task.Version, DigestSHA256: task.DigestSHA256})
	}
	result.Tasks[len(result.Tasks)-1].ID = result.Tasks[0].ID
	if err := ValidateResult(root, result); err == nil {
		t.Fatal("expected duplicate task to be rejected")
	}
}
