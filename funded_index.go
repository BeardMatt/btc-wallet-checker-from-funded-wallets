package main

import (
	"btcfind/bitcoin"
	"btcfind/funded"
)

const defaultMinBalanceSats = funded.DefaultMinBalanceSats

// FundedSets is an alias for the shared funded index type.
type FundedSets = funded.Sets

func matchFunded(sets FundedSets, keys bitcoin.LookupKeys, mask bitcoin.FormatMask) (bitcoin.MatchKind, bool) {
	return funded.Match(sets, keys, mask)
}