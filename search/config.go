package search

import (
	"time"

	"btcfind/bitcoin"
)

// Config controls a bounded or forever search run.
type Config struct {
	Threads              int
	Formats              bitcoin.FormatMask
	NumKeys              int
	SessionBase          uint64
	Forever              bool
	MaxKeys              uint64
	CheckpointInterval   time.Duration
	SimulateHit          bool
	SimulateHitAt        int
	VerifyLookup         bool
	InjectIndexHit       string
	ResetSession         bool
}