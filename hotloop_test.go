package main

import (
	"testing"

	"btcfind/bitcoin"
	"btcfind/funded"
)

func BenchmarkHotLoopMatch(b *testing.B) {
	sets := FundedSets{}
	sets.EnsureBlooms(funded.NopReporter{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wallet := genWalletStaged(sets, bitcoin.AllFormats())
		matchFunded(sets, wallet.Keys, bitcoin.AllFormats())
	}
}