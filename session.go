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

func resetSessionFile() error {
	if err := os.Remove(sessionFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	tmp := sessionFile + ".tmp"
	if err := os.Remove(tmp); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func initForeverSession(reset bool) (sessionState, error) {
	now := time.Now().UTC()
	if reset {
		if err := resetSessionFile(); err != nil {
			return sessionState{}, err
		}
		return sessionState{
			StartedAt:      now,
			LastCheckpoint: now,
		}, nil
	}

	session := loadSession()
	if session.StartedAt.IsZero() {
		session.StartedAt = now
	}
	return session, nil
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