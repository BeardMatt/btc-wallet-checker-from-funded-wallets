package main

import (
	"time"

	"btcfind/funded"
)

func unmapActiveCache() {
	funded.UnmapActive()
}

func mmapFundedCache(path string, expectedMtime time.Time, minBalance uint64) (FundedSets, error) {
	return funded.MmapLoad(path, expectedMtime, minBalance)
}