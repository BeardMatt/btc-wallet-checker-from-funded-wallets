package main

import (
	"os"
	"testing"

	"btcfind/bitcoin"
)

func TestLazyMatchesEagerEmptyIndex(t *testing.T) {
	sets := FundedSets{}
	sets.ensureBlooms()
	mask := bitcoin.AllFormats()

	for i := 0; i < 2000; i++ {
		eager := bitcoin.GenKeypairMasked(mask)
		lazy := genWalletStaged(sets, mask)
		k1, ok1 := matchFunded(sets, eager.Keys, mask)
		k2, ok2 := matchFunded(sets, lazy.Keys, mask)
		if ok1 != ok2 || k1 != k2 {
			t.Fatalf("match mismatch at %d: eager=%v/%v lazy=%v/%v", i, k1, ok1, k2, ok2)
		}
	}
}

func TestLazyMatchesEagerFundedIndex(t *testing.T) {
	if _, err := os.Stat("funded.cache"); err != nil {
		t.Skip("funded.cache not present")
	}
	tsv, err := os.Stat("funded.tsv")
	if err != nil {
		t.Skip("funded.tsv not present")
	}

	sets, err := readFundedCache(fundedCacheFile, tsv.ModTime())
	if err != nil {
		t.Fatal(err)
	}
	mask := bitcoin.AllFormats()

	for i := 0; i < 1000; i++ {
		eager := bitcoin.GenKeypairMasked(mask)
		lazy := genWalletStaged(sets, mask)
		k1, ok1 := matchFunded(sets, eager.Keys, mask)
		k2, ok2 := matchFunded(sets, lazy.Keys, mask)
		if ok1 != ok2 || k1 != k2 {
			t.Fatalf("funded index mismatch at %d: eager=%v/%v lazy=%v/%v", i, k1, ok1, k2, ok2)
		}
	}
}

func TestShouldDeriveTaprootTaprootOnly(t *testing.T) {
	sets := FundedSets{TaprootV1: make([][32]byte, 1)}
	sets.ensureBlooms()
	mask, err := bitcoin.ParseFormatMask("taproot")
	if err != nil {
		t.Fatal(err)
	}
	if !shouldDeriveTaproot(sets, bitcoin.LookupKeys{}, mask) {
		t.Fatal("taproot-only search must always derive taproot")
	}
}