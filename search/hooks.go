package search

import (
	"time"

	"btcfind/bitcoin"
)

// HitEvent describes a funded or simulated wallet match.
type HitEvent struct {
	Wallet      bitcoin.Wallet
	Kind        bitcoin.MatchKind
	KeyIndex    int
	BalanceSats uint64
	Simulated   bool
}

// LookupVerifyEvent reports per-bucket lookup results for debug verification.
type LookupVerifyEvent struct {
	LegacyCompressed   bool
	LegacyUncompressed bool
	SegwitV0           bool
	P2SH               bool
	Taproot            bool
	RealMatch          bool
	Kind               bitcoin.MatchKind
	KindName           string
}

// Hooks carries optional progress and hit callbacks for a search run.
type Hooks struct {
	OnProgress         func(processed, total int, elapsed time.Duration)
	OnForeverProgress  func(totalKeys uint64, elapsed time.Duration, hits uint64)
	OnSummary          func(took time.Duration, numKeys int, avg float64)
	OnHit              func(HitEvent) error
	OnInjectLog        func(bucket, fundedAddress string)
	OnLookupVerify     func(LookupVerifyEvent)
}