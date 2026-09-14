package model

import "testing"

func TestScoreUsesHighestLevelWeights(t *testing.T) {
	s := Score([]TaskResult{
		{MaxLevel: L1}, {MaxLevel: L2}, {MaxLevel: L3}, {MaxLevel: L4}, {MaxLevel: L5},
	})
	if s.Score != 174 || s.IsolationScore != 140 || s.PolicyScore != 4 || s.SecretScore != 10 || s.NL1 != 1 || s.NL5 != 1 {
		t.Fatalf("unexpected score: %+v", s)
	}
}

func TestScoreCountsCleanupFailureAsError(t *testing.T) {
	s := Score([]TaskResult{{Status: StatusHold, MaxLevel: L0, CleanupError: "volume busy"}})
	if s.Errors != 1 || s.Score != 0 {
		t.Fatalf("unexpected cleanup score: %+v", s)
	}
}
