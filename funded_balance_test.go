package main

import (
	"testing"

	"btcfind/bitcoin"
)

func TestBalanceForMatch(t *testing.T) {
	sets := FundedSets{
		Legacy:        [][20]byte{{1, 2, 3}, {9, 9, 9}},
		LegacyBalance: []uint64{50000, 999999},
	}
	keys := bitcoin.LookupKeys{CompressedHash: [20]byte{1, 2, 3}}
	bal := sets.BalanceForMatch(bitcoin.MatchLegacyCompressed, keys)
	if bal != 50000 {
		t.Fatalf("want 50000, got %d", bal)
	}
}