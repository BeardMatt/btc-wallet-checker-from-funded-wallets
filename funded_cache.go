package main

import (
	"time"

	"btcfind/funded"
)

const fundedCacheFile = funded.DefaultCacheFile

func fundedCachePath(minBalance uint64) string {
	return funded.CachePath(minBalance)
}

func writeFundedCache(path string, sets FundedSets, srcMtime time.Time) error {
	return funded.WriteCache(path, sets, srcMtime)
}

func readFundedCache(path string, expectedMtime time.Time, minBalance uint64) (FundedSets, error) {
	return funded.ReadCache(path, expectedMtime, minBalance)
}

func removeFundedCache() {
	_ = funded.RemoveCache(fundedCacheFile)
}