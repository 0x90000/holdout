package runner

import (
	"context"
	"testing"

	"github.com/holdout-bench/holdout/internal/model"
)

func TestRunPreservesValidPartialResultWhenCatalogLoadFails(t *testing.T) {
	result, err := Run(context.Background(), Options{Root: t.TempDir(), Suite: "smoke-v0", SUT: "docker-runc", DryRun: true})
	if err == nil {
		t.Fatal("expected catalog load failure")
	}
	if result.RunID == "" || result.Error == "" {
		t.Fatalf("runner should return an annotated partial result: %+v", result)
	}
	if validateErr := model.ValidateRunResult(result); validateErr != nil {
		t.Fatalf("partial result should satisfy result invariants: %v", validateErr)
	}
}
