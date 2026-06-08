package cluster

import (
	"strings"

	"btcfind/bitcoin"
)

// Run states.
const (
	StateLobby    = "lobby"
	StateRunning  = "running"
	StateStopping = "stopping"
	StateEnded    = "ended"
)

// RegisterRequest is POST /api/v1/register body.
type RegisterRequest struct {
	Hostname string `json:"hostname"`
	Threads  int    `json:"threads"`
	Version  string `json:"version"`
}

// RegisterResponse is returned after registration.
type RegisterResponse struct {
	WorkerID string    `json:"worker_id"`
	RunState string    `json:"run_state"`
	Config   RunConfig `json:"config"`
}

// HeartbeatRequest is POST /api/v1/heartbeat body.
type HeartbeatRequest struct {
	WorkerID string `json:"worker_id"`
	State    string `json:"state"` // waiting | running
}

// RunConfig is coordinator-owned search parameters.
type RunConfig struct {
	Formats        []string `json:"formats"`
	MinBalanceSats uint64   `json:"min_balance_sats"`
	CacheETag      string   `json:"cache_etag"`
}

// RunResponse is GET /api/v1/run body.
type RunResponse struct {
	State  string    `json:"state"`
	Config RunConfig `json:"config"`
}

// StatsRequest is POST /api/v1/stats body.
type StatsRequest struct {
	WorkerID       string  `json:"worker_id"`
	KeysTriedDelta uint64  `json:"keys_tried_delta"`
	KeysPerSecond  float64 `json:"keys_per_second"`
	RunKeysTried   uint64  `json:"run_keys_tried"`
}

// HitRequest is POST /api/v1/hit body.
type HitRequest struct {
	WorkerID    string `json:"worker_id"`
	Address     string `json:"address"`
	WIF         string `json:"wif"`
	Format      string `json:"format"`
	BalanceSats uint64 `json:"balance_sats"`
	KeyIndex    int    `json:"key_index"`
}

// FormatList converts a FormatMask to API string tokens.
func FormatList(mask bitcoin.FormatMask) []string {
	var out []string
	if mask.LegacyCompressed {
		out = append(out, "legacy-compressed")
	}
	if mask.LegacyUncompressed {
		out = append(out, "legacy-uncompressed")
	}
	if mask.Segwit {
		out = append(out, "segwit")
	}
	if mask.P2SH {
		out = append(out, "p2sh")
	}
	if mask.Taproot {
		out = append(out, "taproot")
	}
	return out
}

// ParseFormatList builds a FormatMask from coordinator config tokens.
func ParseFormatList(tokens []string) (bitcoin.FormatMask, error) {
	if len(tokens) == 0 {
		return bitcoin.AllFormats(), nil
	}
	return bitcoin.ParseFormatMask(strings.Join(tokens, ","))
}