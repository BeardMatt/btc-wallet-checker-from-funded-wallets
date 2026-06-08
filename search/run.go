package search

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"time"

	"btcfind/bitcoin"
	"btcfind/funded"
)

const sessionFile = "btcfind.session"

// Stats summarizes keys processed during a search run.
type Stats struct {
	KeysProcessed uint64
	Hits          uint64
}

type sessionState struct {
	StartedAt         time.Time `json:"started_at"`
	LastCheckpoint    time.Time `json:"last_checkpoint"`
	TotalKeysTried    uint64    `json:"total_keys_tried"`
	TotalHits         uint64    `json:"total_hits"`
	BestKeysPerSecond float64   `json:"best_keys_per_second"`
}

// Run executes a bounded or forever search until completion, cancellation, or error.
func Run(ctx context.Context, cfg Config, sets funded.Sets, hooks Hooks) (Stats, error) {
	workers := cfg.Threads
	if workers < 1 {
		workers = runtime.NumCPU()
	}

	if cfg.Forever {
		return runForever(ctx, cfg, workers, sets, hooks)
	}
	return runBounded(ctx, cfg, workers, sets, hooks)
}

func runBounded(ctx context.Context, cfg Config, workers int, sets funded.Sets, hooks Hooks) (Stats, error) {
	start := time.Now()
	ch := startWorkers(ctx, workers, sets, cfg.Formats)
	lastProgress := time.Now()
	processed := 0
	var stats Stats

	for processed < cfg.NumKeys {
		select {
		case <-ctx.Done():
			return stats, ctx.Err()
		default:
		}

		var batch []bitcoin.Wallet
		select {
		case batch = <-ch:
		case <-ctx.Done():
			return stats, ctx.Err()
		}

		for _, wallet := range batch {
			if processed >= cfg.NumKeys {
				break
			}

			keyIndex := int(cfg.SessionBase) + processed + 1
			hit, err := ProcessWallet(cfg, sets, wallet, keyIndex, hooks)
			if err != nil {
				return stats, err
			}
			if hit {
				stats.Hits++
			}
			processed++
			stats.KeysProcessed++
		}

		now := time.Now()
		if hooks.OnProgress != nil && now.Sub(lastProgress) >= 100*time.Millisecond {
			hooks.OnProgress(processed, cfg.NumKeys, now.Sub(start))
			lastProgress = now
		}
	}

	took := time.Since(start)
	if hooks.OnSummary != nil && cfg.NumKeys > 0 {
		hooks.OnSummary(took, cfg.NumKeys, float64(cfg.NumKeys)/took.Seconds())
	}
	return stats, nil
}

func runForever(ctx context.Context, cfg Config, workers int, sets funded.Sets, hooks Hooks) (Stats, error) {
	session, err := initForeverSession(cfg.ResetSession)
	if err != nil {
		return Stats{}, err
	}

	start := time.Now()
	ch := startWorkers(ctx, workers, sets, cfg.Formats)
	lastProgress := time.Now()
	lastCheckpoint := time.Now()

	var stats Stats
	processed := uint64(0)
	runProcessed := uint64(0)
	hits := session.TotalHits

	for {
		select {
		case <-ctx.Done():
			session.TotalKeysTried += processed
			session.TotalHits = hits
			finalizeSessionBestKPS(&session)
			_ = writeSession(session)
			return stats, ctx.Err()
		default:
		}

		if cfg.MaxKeys > 0 && runProcessed >= cfg.MaxKeys {
			break
		}

		var batch []bitcoin.Wallet
		select {
		case batch = <-ch:
		case <-ctx.Done():
			session.TotalKeysTried += processed
			session.TotalHits = hits
			finalizeSessionBestKPS(&session)
			_ = writeSession(session)
			return stats, ctx.Err()
		}

		maxKeysReached := false
		for _, wallet := range batch {
			select {
			case <-ctx.Done():
				session.TotalKeysTried += processed
				session.TotalHits = hits
				finalizeSessionBestKPS(&session)
				_ = writeSession(session)
				return stats, ctx.Err()
			default:
			}

			if cfg.MaxKeys > 0 && runProcessed >= cfg.MaxKeys {
				maxKeysReached = true
				break
			}

			keyIndex := int(session.TotalKeysTried + processed + 1)
			hit, err := ProcessWallet(cfg, sets, wallet, keyIndex, hooks)
			if err != nil {
				session.TotalKeysTried += processed
				session.TotalHits = hits
				_ = writeSession(session)
				return stats, err
			}
			if hit {
				hits++
				stats.Hits++
			}
			processed++
			runProcessed++
			stats.KeysProcessed++
		}
		if maxKeysReached {
			break
		}

		now := time.Now()
		elapsed := now.Sub(start)
		if hooks.OnForeverProgress != nil && now.Sub(lastProgress) >= 100*time.Millisecond {
			hooks.OnForeverProgress(session.TotalKeysTried+processed, elapsed, hits)
			lastProgress = now
		}
		if cfg.CheckpointInterval > 0 && now.Sub(lastCheckpoint) >= cfg.CheckpointInterval {
			kps := float64(processed) / elapsed.Seconds()
			if kps > session.BestKeysPerSecond {
				session.BestKeysPerSecond = kps
			}
			session.TotalKeysTried += processed
			session.TotalHits = hits
			_ = writeSession(session)
			processed = 0
			start = time.Now()
			lastCheckpoint = now
		}
	}

	session.TotalKeysTried += processed
	session.TotalHits = hits
	finalizeSessionBestKPS(&session)
	_ = writeSession(session)
	return stats, nil
}

func finalizeSessionBestKPS(session *sessionState) {
	kps := float64(session.TotalKeysTried) / time.Since(session.StartedAt).Seconds()
	if kps > session.BestKeysPerSecond {
		session.BestKeysPerSecond = kps
	}
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