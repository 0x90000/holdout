package model

import (
	"testing"
	"time"
)

func TestValidateRunResultRejectsSummaryDrift(t *testing.T) {
	result := RunResult{
		Schema: "https://holdout.dev/schema/result.v1.json",
		RunID:  "run-1", Suite: "smoke-v0",
		Started: time.Unix(10, 0).UTC(), Ended: time.Unix(11, 0).UTC(),
		SUT:     validSUT(),
		Tasks:   []TaskResult{{ID: "T001", Version: 1, DigestSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Status: StatusHold, MaxLevel: L0}},
		Summary: Summary{Score: 4},
	}
	if err := ValidateRunResult(result); err == nil {
		t.Fatal("expected summary drift to be rejected")
	}
}

func TestValidateRunResultRejectsStatusLevelMismatch(t *testing.T) {
	result := RunResult{
		Schema: "https://holdout.dev/schema/result.v1.json",
		RunID:  "run-1", Suite: "smoke-v0",
		Started: time.Unix(10, 0).UTC(), Ended: time.Unix(11, 0).UTC(),
		SUT:   validSUT(),
		Tasks: []TaskResult{{ID: "T001", Version: 1, DigestSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Status: StatusHold, MaxLevel: L2}},
	}
	if err := ValidateRunResult(result); err == nil {
		t.Fatal("expected status-level mismatch to be rejected")
	}
}

func TestValidateRunResultAllowsCleanupErrorWithObservedResult(t *testing.T) {
	result := RunResult{
		Schema: "https://holdout.dev/schema/result.v1.json", RunID: "run-1", Suite: "smoke-v0",
		Started: time.Unix(10, 0).UTC(), Ended: time.Unix(11, 0).UTC(),
		SUT:   validSUT(),
		Tasks: []TaskResult{{ID: "T001", Version: 1, DigestSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Status: StatusHold, MaxLevel: L0, CleanupError: "cleanup failed"}},
	}
	result.Summary = Score(result.Tasks)
	if err := ValidateRunResult(result); err != nil {
		t.Fatal(err)
	}
}

func validSUT() SutDescription {
	return SutDescription{
		Name: "docker-runc", Version: "unavailable", ImageDigest: "unavailable", AdapterVersion: "0.1.0",
		ConfigSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Kernel:       "unavailable", Runtime: "unavailable", Architecture: "amd64", NetMode: "unknown",
	}
}
