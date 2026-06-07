package main

import (
	"testing"

	"btcfind/bitcoin"
)

func BenchmarkHotLoopMatch(b *testing.B) {
	sets := FundedSets{}
	sets.ensureBlooms()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wallet := bitcoin.GenKeypair()
		matchFunded(sets, wallet.Keys)
	}
}