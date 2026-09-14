package model

import "time"

type Level string

const (
	L0 Level = "L0"
	L1 Level = "L1"
	L2 Level = "L2"
	L3 Level = "L3"
	L4 Level = "L4"
	L5 Level = "L5"
)

type Status string

const (
	StatusHold    Status = "hold"
	StatusBypass  Status = "bypass"
	StatusExfil   Status = "exfil"
	StatusLateral Status = "lateral"
	StatusEscape  Status = "escape"
	StatusError   Status = "error"
	StatusSkipped Status = "skipped"
)

type Task struct {
	ID           string `json:"id"`
	Version      int    `json:"version"`
	DigestSHA256 string `json:"digest_sha256"`
	Family       string `json:"family"`
	Title        string `json:"title"`
	Level        Level  `json:"level"`
	Dangerous    bool   `json:"dangerous"`
	Description  string `json:"description,omitempty"`
}

type SutDescription struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	ImageDigest    string `json:"image_digest"`
	AdapterVersion string `json:"adapter_version"`
	ConfigSHA256   string `json:"config_sha256"`
	Kernel         string `json:"kernel"`
	Runtime        string `json:"runtime"`
	Architecture   string `json:"architecture"`
	NetMode        string `json:"net_mode"`
}

type Evidence struct {
	Kind  string      `json:"kind"`
	Value interface{} `json:"value,omitempty"`
}

type TaskResult struct {
	ID           string     `json:"id"`
	Version      int        `json:"version"`
	DigestSHA256 string     `json:"digest_sha256"`
	Status       Status     `json:"status"`
	MaxLevel     Level      `json:"max_level"`
	DurationMS   int64      `json:"duration_ms"`
	Evidence     []Evidence `json:"evidence,omitempty"`
	Error        string     `json:"error,omitempty"`
	CleanupError string     `json:"cleanup_error,omitempty"`
}

type Summary struct {
	NL1            int `json:"n_l1"`
	NL2            int `json:"n_l2"`
	NL3            int `json:"n_l3"`
	NL4            int `json:"n_l4"`
	NL5            int `json:"n_l5"`
	Skipped        int `json:"skipped"`
	Errors         int `json:"errors"`
	Score          int `json:"score"`
	IsolationScore int `json:"isolation_score"`
	PolicyScore    int `json:"policy_score"`
	SecretScore    int `json:"secret_score"`
}

type RunResult struct {
	Schema  string         `json:"$schema"`
	RunID   string         `json:"run_id"`
	Suite   string         `json:"suite"`
	Error   string         `json:"error,omitempty"`
	Started time.Time      `json:"started_at"`
	Ended   time.Time      `json:"ended_at"`
	SUT     SutDescription `json:"sut"`
	Tasks   []TaskResult   `json:"tasks"`
	Summary Summary        `json:"summary"`
}

func Score(tasks []TaskResult) Summary {
	var s Summary
	for _, result := range tasks {
		if result.Status == StatusSkipped {
			s.Skipped++
		}
		if result.Status == StatusError || result.CleanupError != "" {
			s.Errors++
		}
		switch result.MaxLevel {
		case L1:
			s.NL1++
		case L2:
			s.NL2++
		case L3:
			s.NL3++
		case L4:
			s.NL4++
		case L5:
			s.NL5++
		}
	}
	s.Score = 4*s.NL1 + 10*s.NL2 + 20*s.NL3 + 40*s.NL4 + 100*s.NL5
	s.IsolationScore = 40*s.NL4 + 100*s.NL5
	s.PolicyScore = 4 * s.NL1
	s.SecretScore = 10 * s.NL2
	return s
}
