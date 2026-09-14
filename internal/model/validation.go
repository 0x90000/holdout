package model

import (
	"fmt"
	"regexp"
)

var sha256Pattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

// ValidateRunResult checks the invariants required by result.v1. It is kept
// dependency-free so archived results can be checked offline.
func ValidateRunResult(result RunResult) error {
	if result.Schema != "https://holdout.dev/schema/result.v1.json" {
		return fmt.Errorf("unsupported result schema %q", result.Schema)
	}
	if result.RunID == "" || result.Suite == "" {
		return fmt.Errorf("run_id and suite are required")
	}
	if result.Started.IsZero() || result.Ended.IsZero() || result.Ended.Before(result.Started) {
		return fmt.Errorf("invalid result time range")
	}
	if result.SUT.Name == "" || result.SUT.Version == "" || result.SUT.ImageDigest == "" ||
		result.SUT.AdapterVersion == "" || result.SUT.Kernel == "" || result.SUT.Runtime == "" ||
		result.SUT.Architecture == "" || result.SUT.NetMode == "" || !sha256Pattern.MatchString(result.SUT.ConfigSHA256) {
		return fmt.Errorf("incomplete or invalid SUT description")
	}
	seen := make(map[string]bool, len(result.Tasks))
	for _, task := range result.Tasks {
		if task.ID == "" || seen[task.ID] {
			return fmt.Errorf("invalid or duplicate task result %q", task.ID)
		}
		seen[task.ID] = true
		if task.Version < 1 || !sha256Pattern.MatchString(task.DigestSHA256) {
			return fmt.Errorf("task %s has invalid version or digest", task.ID)
		}
		if task.DurationMS < 0 || !validLevel(task.MaxLevel) || !validStatus(task.Status) {
			return fmt.Errorf("task %s has invalid status, level, or duration", task.ID)
		}
		if task.Status == StatusHold && task.MaxLevel != L0 {
			return fmt.Errorf("task %s reports hold with non-zero level", task.ID)
		}
		if task.Status == StatusError || task.Status == StatusSkipped {
			if task.MaxLevel != L0 || task.Error == "" {
				return fmt.Errorf("task %s error/skipped result must have L0 and a reason", task.ID)
			}
		} else if task.Status != StatusHold && statusForLevel(task.MaxLevel) != task.Status {
			return fmt.Errorf("task %s status does not match max level", task.ID)
		}
	}
	expected := Score(result.Tasks)
	if result.Summary != expected {
		return fmt.Errorf("summary does not match task results: got %+v want %+v", result.Summary, expected)
	}
	return nil
}

func statusForLevel(level Level) Status {
	switch level {
	case L1:
		return StatusBypass
	case L2, L3:
		return StatusExfil
	case L4:
		return StatusLateral
	case L5:
		return StatusEscape
	default:
		return StatusHold
	}
}

func validLevel(level Level) bool {
	switch level {
	case L0, L1, L2, L3, L4, L5:
		return true
	default:
		return false
	}
}

func validStatus(status Status) bool {
	switch status {
	case StatusHold, StatusBypass, StatusExfil, StatusLateral, StatusEscape, StatusError, StatusSkipped:
		return true
	default:
		return false
	}
}
