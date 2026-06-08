package main

import (
	"encoding/json"
	"os"
	"time"
)

const sessionFile = "btcfind.session"

type sessionState struct {
	StartedAt         time.Time `json:"started_at"`
	LastCheckpoint    time.Time `json:"last_checkpoint"`
	TotalKeysTried    uint64    `json:"total_keys_tried"`
	TotalHits         uint64    `json:"total_hits"`
	BestKeysPerSecond float64   `json:"best_keys_per_second"`
}

func loadSession() sessionState {
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return sessionState{
			StartedAt:      time.Now().UTC(),
			LastCheckpoint: time.Now().UTC(),
		}
	}
	var s sessionState
	if err := json.Unmarshal(data, &s); err != nil {
		return sessionState{
			StartedAt:      time.Now().UTC(),
			LastCheckpoint: time.Now().UTC(),
		}
	}
	if s.StartedAt.IsZero() {
		s.StartedAt = time.Now().UTC()
	}
	return s
}

func writeSession(s sessionState) error {
	s.LastCheckpoint = time.Now().UTC()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := sessionFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, sessionFile)
}