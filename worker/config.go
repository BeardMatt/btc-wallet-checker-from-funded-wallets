package worker

import "time"

// Config holds btcfind-worker settings.
type Config struct {
	CoordinatorURL string
	AuthToken      string
	Threads        int

	TLSSkipVerify bool
	TLSCA         string
	TLSCert       string
	TLSKey        string

	ProxyURL     string
	DialTimeout  time.Duration
	StatsInterval time.Duration

	SimulateHit   bool
	SimulateHitAt int
	Quiet         bool
	Verbose       bool
}