package worker

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"btcfind/bitcoin"
	"btcfind/cluster"
	"btcfind/funded"
	"btcfind/search"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

const workerVersion = "1"

// Run connects to coordinator, syncs cache, and searches when told to start.
func Run(ctx context.Context, cfg Config) error {
	if cfg.AuthToken == "" {
		return fmt.Errorf("auth token required")
	}
	if cfg.StatsInterval == 0 {
		cfg.StatsInterval = 2 * time.Second
	}
	threads := cfg.Threads
	if threads < 1 {
		threads = runtime.NumCPU()
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "worker"
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := runSession(ctx, cfg, client, hostname, threads); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			fmt.Fprintf(os.Stderr, "session error (reconnecting): %v\n", err)
			time.Sleep(2 * time.Second)
			continue
		}
		return nil
	}
}

func runSession(ctx context.Context, cfg Config, client *Client, hostname string, threads int) error {
	reg, err := client.Register(ctx, hostname, threads, workerVersion)
	if err != nil {
		return err
	}
	workerID := reg.WorkerID
	if workerID == "" {
		return fmt.Errorf("registration rejected (cluster full?)")
	}

	minBal := reg.Config.MinBalanceSats
	if minBal == 0 {
		minBal = funded.DefaultMinBalanceSats
	}
	cachePath := funded.CachePath(minBal)

	localETag, _ := funded.CacheETag(cachePath)
	remoteETag, err := client.DownloadCache(ctx, cachePath, localETag)
	if err != nil {
		return fmt.Errorf("cache sync: %w", err)
	}

	sets, err := funded.LoadFromCacheFile(cachePath)
	if err != nil {
		return fmt.Errorf("load cache: %w", err)
	}
	defer funded.UnmapActive()

	go heartbeatLoop(ctx, client, workerID)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		status, err := client.RunStatus(ctx)
		if err != nil {
			return err
		}

		switch status.State {
		case cluster.StateLobby:
			_ = client.Heartbeat(ctx, workerID, "waiting")
			time.Sleep(time.Second)
		case cluster.StateRunning:
			if status.Config.CacheETag != "" && status.Config.CacheETag != remoteETag {
				remoteETag, err = client.DownloadCache(ctx, cachePath, remoteETag)
				if err != nil {
					return err
				}
				funded.UnmapActive()
				sets, err = funded.LoadFromCacheFile(cachePath)
				if err != nil {
					return err
				}
			}
			formats, err := cluster.ParseFormatList(status.Config.Formats)
			if err != nil {
				return err
			}
			if err := searchUntilStop(ctx, cfg, client, workerID, threads, sets, formats); err != nil {
				return err
			}
		case cluster.StateStopping, cluster.StateEnded:
			_ = client.Heartbeat(ctx, workerID, "waiting")
			return nil
		default:
			time.Sleep(time.Second)
		}
	}
}

func heartbeatLoop(ctx context.Context, client *Client, workerID string) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	state := "waiting"
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = client.Heartbeat(ctx, workerID, state)
		}
	}
}

func searchUntilStop(ctx context.Context, cfg Config, client *Client, workerID string, threads int, sets funded.Sets, formats bitcoin.FormatMask) error {
	_ = client.Heartbeat(ctx, workerID, "running")

	searchCtx, searchCancel := context.WithCancel(ctx)
	defer searchCancel()

	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-searchCtx.Done():
				return
			case <-t.C:
				status, err := client.RunStatus(searchCtx)
				if err != nil || status.State != cluster.StateRunning {
					searchCancel()
					return
				}
			}
		}
	}()

	var runKeys atomic.Uint64
	var sinceReport atomic.Uint64
	start := time.Now()

	statsCtx, statsCancel := context.WithCancel(searchCtx)
	defer statsCancel()
	go func() {
		t := time.NewTicker(cfg.StatsInterval)
		defer t.Stop()
		for {
			select {
			case <-statsCtx.Done():
				return
			case <-t.C:
				delta := sinceReport.Swap(0)
				elapsed := time.Since(start).Seconds()
				kps := 0.0
				if elapsed > 0 {
					kps = float64(runKeys.Load()) / elapsed
				}
				_ = client.PostStats(statsCtx, cluster.StatsRequest{
					WorkerID:       workerID,
					KeysTriedDelta: delta,
					KeysPerSecond:  kps,
					RunKeysTried:   runKeys.Load(),
				})
			}
		}
	}()

	searchCfg := search.Config{
		Threads:       threads,
		Formats:       formats,
		Forever:       true,
		SimulateHit:   cfg.SimulateHit,
		SimulateHitAt: cfg.SimulateHitAt,
	}

	var lastTotal uint64
	hooks := search.Hooks{
		OnForeverProgress: func(totalKeys uint64, _ time.Duration, _ uint64) {
			delta := totalKeys - lastTotal
			lastTotal = totalKeys
			runKeys.Add(delta)
			sinceReport.Add(delta)
		},
		OnHit: func(ev search.HitEvent) error {
			addr := bitcoin.EncodeMatchAddress(ev.Kind, ev.Wallet.Keys)
			wif := formatWIF(ev.Wallet.PrivKey)
			return client.PostHit(searchCtx, cluster.HitRequest{
				WorkerID:    workerID,
				Address:     addr,
				WIF:         wif,
				Format:      funded.MatchKindName(ev.Kind),
				BalanceSats: ev.BalanceSats,
				KeyIndex:    ev.KeyIndex,
			})
		},
	}

	_, err := search.Run(searchCtx, searchCfg, sets, hooks)

	delta := sinceReport.Swap(0)
	elapsed := time.Since(start).Seconds()
	kps := 0.0
	if elapsed > 0 {
		kps = float64(runKeys.Load()) / elapsed
	}
	_ = client.PostStats(context.Background(), cluster.StatsRequest{
		WorkerID:       workerID,
		KeysTriedDelta: delta,
		KeysPerSecond:  kps,
		RunKeysTried:   runKeys.Load(),
	})
	_ = client.Heartbeat(context.Background(), workerID, "waiting")

	if err != nil && err != context.Canceled {
		return err
	}
	return nil
}

func formatWIF(privKeyBytes []byte) string {
	privKey, _ := btcec.PrivKeyFromBytes(privKeyBytes)
	wif, err := btcutil.NewWIF(privKey, &chaincfg.MainNetParams, true)
	if err != nil {
		panic(err)
	}
	return wif.String()
}