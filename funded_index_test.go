package main

import (
	"testing"
)

func TestBloomNoFalseNegatives(t *testing.T) {
	sets := FundedSets{
		Legacy: [][20]byte{
			{0, 0, 0, 0, 1},
			{0, 0, 0, 0, 2},
			{0xff, 0xff, 0xff, 0xff, 0xff},
		},
		P2SH: [][20]byte{
			{1, 2, 3},
		},
		SegwitV0: [][20]byte{
			{4, 5, 6},
		},
		TaprootV1: [][32]byte{
			{7, 8, 9},
			{0xaa, 0xbb, 0xcc},
		},
	}
	sets.BuildBlooms()

	assertBloomCovers20(t, sets.LegacyBloom, sets.Legacy)
	assertBloomCovers20(t, sets.P2SHBloom, sets.P2SH)
	assertBloomCovers20(t, sets.SegwitV0Bloom, sets.SegwitV0)
	assertBloomCovers32(t, sets.TaprootV1Bloom, sets.TaprootV1)
}

func assertBloomCovers20(t *testing.T, bf interface{ Test([]byte) bool }, set [][20]byte) {
	t.Helper()
	if bf == nil {
		if len(set) > 0 {
			t.Fatal("expected bloom filter for non-empty set")
		}
		return
	}
	for i, entry := range set {
		if !bf.Test(entry[:]) {
			t.Fatalf("bloom false negative at index %d", i)
		}
	}
}

func assertBloomCovers32(t *testing.T, bf interface{ Test([]byte) bool }, set [][32]byte) {
	t.Helper()
	if bf == nil {
		if len(set) > 0 {
			t.Fatal("expected bloom filter for non-empty set")
		}
		return
	}
	for i, entry := range set {
		if !bf.Test(entry[:]) {
			t.Fatalf("bloom false negative at index %d", i)
		}
	}
}