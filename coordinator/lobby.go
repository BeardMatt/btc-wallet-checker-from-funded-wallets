package coordinator

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"btcfind/cluster"
	"btcfind/funded"
)

type workerState string

const (
	workerWaiting workerState = "waiting"
	workerRunning workerState = "running"
	workerOffline workerState = "offline"
)

type workerEntry struct {
	ID         string
	Hostname   string
	Threads    int
	Version    string
	State      workerState
	LastSeen   time.Time
	RunKeys    uint64
	KeysPerSec float64
}

// Server is the cluster coordinator.
type Server struct {
	cfg Config
	ui  *UI

	mu           sync.RWMutex
	workers      map[string]*workerEntry
	runState     string
	runConfig    cluster.RunConfig
	cachePath    string
	fundedSets   funded.Sets
	clusterTotal uint64
	clusterKPS   float64
	clusterHits  uint64
	startedAt    time.Time
	hitAddresses []string
	stopCh       chan struct{}
}

func New(cfg Config, ui *UI) *Server {
	return &Server{
		cfg:      cfg,
		ui:       ui,
		workers:  make(map[string]*workerEntry),
		runState: cluster.StateLobby,
		stopCh:   make(chan struct{}),
	}
}

func (s *Server) setRunConfig(etag string) {
	s.runConfig = cluster.RunConfig{
		Formats:        cluster.FormatList(s.cfg.Formats),
		MinBalanceSats: s.cfg.MinBalance,
		CacheETag:      etag,
	}
}

func (s *Server) register(req cluster.RegisterRequest) cluster.RegisterResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cfg.MaxWorkers > 0 && len(s.workers) >= s.cfg.MaxWorkers {
		return cluster.RegisterResponse{}
	}

	id := newWorkerID()
	s.workers[id] = &workerEntry{
		ID:       id,
		Hostname: req.Hostname,
		Threads:  req.Threads,
		Version:  req.Version,
		State:    workerWaiting,
		LastSeen: time.Now(),
	}
	return cluster.RegisterResponse{
		WorkerID: id,
		RunState: s.runState,
		Config:   s.runConfig,
	}
}

func (s *Server) heartbeat(workerID string, state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, ok := s.workers[workerID]
	if !ok {
		return errUnknownWorker
	}
	w.LastSeen = time.Now()
	switch state {
	case "running":
		w.State = workerRunning
	case "waiting":
		w.State = workerWaiting
	}
	return nil
}

func (s *Server) recordStats(req cluster.StatsRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, ok := s.workers[req.WorkerID]
	if !ok {
		return
	}
	w.LastSeen = time.Now()
	w.RunKeys = req.RunKeysTried
	w.KeysPerSec = req.KeysPerSecond
	w.State = workerRunning

	s.clusterTotal += req.KeysTriedDelta

	var sum float64
	for _, ww := range s.workers {
		if ww.State == workerRunning && time.Since(ww.LastSeen) < 30*time.Second {
			sum += ww.KeysPerSec
		}
	}
	s.clusterKPS = sum
}

func (s *Server) runStatus() cluster.RunResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cluster.RunResponse{State: s.runState, Config: s.runConfig}
}

func (s *Server) startRun() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runState = cluster.StateRunning
	s.startedAt = time.Now()
	s.clusterTotal = 0
	s.clusterHits = 0
	s.hitAddresses = nil
	for _, w := range s.workers {
		w.State = workerRunning
		w.RunKeys = 0
		w.KeysPerSec = 0
	}
}

func (s *Server) stopRun() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runState = cluster.StateStopping
}

func (s *Server) endRun() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runState = cluster.StateEnded
}

func (s *Server) resetLobby() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runState = cluster.StateLobby
	for _, w := range s.workers {
		w.State = workerWaiting
	}
}

func (s *Server) pruneStale() {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-30 * time.Second)
	for id, w := range s.workers {
		if w.LastSeen.Before(cutoff) {
			w.State = workerOffline
			delete(s.workers, id)
		}
	}
}

func (s *Server) workerSnapshot() []workerEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]workerEntry, 0, len(s.workers))
	for _, w := range s.workers {
		out = append(out, *w)
	}
	return out
}

func (s *Server) clusterSnapshot() (state string, total uint64, kps float64, hits uint64, started time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runState, s.clusterTotal, s.clusterKPS, s.clusterHits, s.startedAt
}

var errUnknownWorker = &apiError{msg: "unknown worker_id", code: 404}

func newWorkerID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return "w-" + hex.EncodeToString(b[:])
}