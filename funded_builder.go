package main

import "btcfind/funded"

func newFundedBuilder(minBalance uint64) *funded.Builder {
	return funded.NewBuilder(minBalance)
}