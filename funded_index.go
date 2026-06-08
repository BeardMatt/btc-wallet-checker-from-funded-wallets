package main

import (
	"bytes"
	"sort"
	"sync"
	"time"

	"btcfind/bitcoin"

	"github.com/bits-and-blooms/bloom/v3"
)

const bloomFalsePositiveRate = 0.0001

const defaultMinBalanceSats = 30000

type FundedSets struct {
	Legacy    [][20]byte
	P2SH      [][20]byte
	SegwitV0  [][20]byte
	TaprootV1 [][32]byte
	Other     []string // non-standard entries that fail address decode

	LegacyBalance    []uint64
	P2SHBalance      []uint64
	SegwitV0Balance  []uint64
	TaprootV1Balance []uint64
	MinBalanceSats   uint64

	LegacyBloom    *bloom.BloomFilter
	P2SHBloom      *bloom.BloomFilter
	SegwitV0Bloom  *bloom.BloomFilter
	TaprootV1Bloom *bloom.BloomFilter
}

func (s FundedSets) Total() int {
	return len(s.Legacy) + len(s.P2SH) + len(s.SegwitV0) + len(s.TaprootV1) + len(s.Other)
}

func appendHash20(slice [][20]byte, hash []byte) [][20]byte {
	if len(hash) != 20 {
		return slice
	}
	var fixed [20]byte
	copy(fixed[:], hash)
	return append(slice, fixed)
}

func appendHash32(slice [][32]byte, hash []byte) [][32]byte {
	if len(hash) != 32 {
		return slice
	}
	var fixed [32]byte
	copy(fixed[:], hash)
	return append(slice, fixed)
}

func (s *FundedSets) Sort() {
	sort.Slice(s.Legacy, func(i, j int) bool {
		return bytes.Compare(s.Legacy[i][:], s.Legacy[j][:]) < 0
	})
	sort.Slice(s.P2SH, func(i, j int) bool {
		return bytes.Compare(s.P2SH[i][:], s.P2SH[j][:]) < 0
	})
	sort.Slice(s.SegwitV0, func(i, j int) bool {
		return bytes.Compare(s.SegwitV0[i][:], s.SegwitV0[j][:]) < 0
	})
	sort.Slice(s.TaprootV1, func(i, j int) bool {
		return bytes.Compare(s.TaprootV1[i][:], s.TaprootV1[j][:]) < 0
	})
	sort.Strings(s.Other)
}

func (s FundedSets) bloomsComplete() bool {
	return bloomReady(s.LegacyBloom, len(s.Legacy)) &&
		bloomReady(s.P2SHBloom, len(s.P2SH)) &&
		bloomReady(s.SegwitV0Bloom, len(s.SegwitV0)) &&
		bloomReady(s.TaprootV1Bloom, len(s.TaprootV1))
}

func bloomReady(bf *bloom.BloomFilter, entries int) bool {
	return entries == 0 || bf != nil
}

func (s *FundedSets) ensureBlooms() {
	if s.bloomsComplete() {
		return
	}
	s.BuildBlooms()
}

func (s *FundedSets) BuildBlooms() {
	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		s.LegacyBloom = buildBloom20(s.Legacy)
		wg.Done()
	}()
	go func() {
		s.P2SHBloom = buildBloom20(s.P2SH)
		wg.Done()
	}()
	go func() {
		s.SegwitV0Bloom = buildBloom20(s.SegwitV0)
		wg.Done()
	}()
	go func() {
		s.TaprootV1Bloom = buildBloom32(s.TaprootV1)
		wg.Done()
	}()
	wg.Wait()

	elapsed := time.Since(start)
	u := ui()
	u.LogBloomBuild(*s, elapsed)
	if u.verbose {
		u.PrintlnErr("Bloom filter details:")
		logBloomBucket("legacy", s.LegacyBloom, len(s.Legacy))
		logBloomBucket("p2sh", s.P2SHBloom, len(s.P2SH))
		logBloomBucket("segwit", s.SegwitV0Bloom, len(s.SegwitV0))
		logBloomBucket("taproot", s.TaprootV1Bloom, len(s.TaprootV1))
	}
}

func buildBloom20(set [][20]byte) *bloom.BloomFilter {
	if len(set) == 0 {
		return nil
	}
	bf := bloom.NewWithEstimates(uint(len(set)), bloomFalsePositiveRate)
	for _, entry := range set {
		bf.Add(entry[:])
	}
	return bf
}

func buildBloom32(set [][32]byte) *bloom.BloomFilter {
	if len(set) == 0 {
		return nil
	}
	bf := bloom.NewWithEstimates(uint(len(set)), bloomFalsePositiveRate)
	for _, entry := range set {
		bf.Add(entry[:])
	}
	return bf
}

func logBloomBucket(name string, bf *bloom.BloomFilter, entries int) {
	if bf == nil {
		return
	}
	fp := bloom.EstimateFalsePositiveRate(bf.Cap(), bf.K(), uint(entries))
	ui().PrintfErr(
		"    %s: %d entries, %d bits, %d hashes, est FP %.6f\n",
		name,
		entries,
		bf.Cap(),
		bf.K(),
		fp,
	)
}

func maybeFunded20(bf *bloom.BloomFilter, set [][20]byte, key [20]byte) bool {
	if bf != nil && !bf.Test(key[:]) {
		return false
	}
	return inFunded20(set, key)
}

func maybeFunded32(bf *bloom.BloomFilter, set [][32]byte, key [32]byte) bool {
	if bf != nil && !bf.Test(key[:]) {
		return false
	}
	return inFunded32(set, key)
}

func inFunded20(set [][20]byte, key [20]byte) bool {
	idx := sort.Search(len(set), func(i int) bool {
		return bytes.Compare(set[i][:], key[:]) >= 0
	})
	return idx < len(set) && set[idx] == key
}

func inFunded32(set [][32]byte, key [32]byte) bool {
	idx := sort.Search(len(set), func(i int) bool {
		return bytes.Compare(set[i][:], key[:]) >= 0
	})
	return idx < len(set) && set[idx] == key
}

func balanceAt20(set [][20]byte, balances []uint64, key [20]byte) uint64 {
	idx := sort.Search(len(set), func(i int) bool {
		return bytes.Compare(set[i][:], key[:]) >= 0
	})
	if idx < len(set) && set[idx] == key && idx < len(balances) {
		return balances[idx]
	}
	return 0
}

func balanceAt32(set [][32]byte, balances []uint64, key [32]byte) uint64 {
	idx := sort.Search(len(set), func(i int) bool {
		return bytes.Compare(set[i][:], key[:]) >= 0
	})
	if idx < len(set) && set[idx] == key && idx < len(balances) {
		return balances[idx]
	}
	return 0
}

func (s FundedSets) balanceForMatch(kind bitcoin.MatchKind, keys bitcoin.LookupKeys) uint64 {
	switch kind {
	case bitcoin.MatchLegacyCompressed:
		return balanceAt20(s.Legacy, s.LegacyBalance, keys.CompressedHash)
	case bitcoin.MatchLegacyUncompressed:
		return balanceAt20(s.Legacy, s.LegacyBalance, keys.UncompressedHash)
	case bitcoin.MatchSegwitV0:
		return balanceAt20(s.SegwitV0, s.SegwitV0Balance, keys.CompressedHash)
	case bitcoin.MatchP2SH:
		return balanceAt20(s.P2SH, s.P2SHBalance, keys.P2SHHash)
	case bitcoin.MatchTaproot:
		return balanceAt32(s.TaprootV1, s.TaprootV1Balance, keys.TaprootKey)
	default:
		return 0
	}
}

func matchFunded(sets FundedSets, keys bitcoin.LookupKeys, mask bitcoin.FormatMask) (bitcoin.MatchKind, bool) {
	if mask.LegacyCompressed && maybeFunded20(sets.LegacyBloom, sets.Legacy, keys.CompressedHash) {
		return bitcoin.MatchLegacyCompressed, true
	}
	if mask.LegacyUncompressed && maybeFunded20(sets.LegacyBloom, sets.Legacy, keys.UncompressedHash) {
		return bitcoin.MatchLegacyUncompressed, true
	}
	if mask.Segwit && maybeFunded20(sets.SegwitV0Bloom, sets.SegwitV0, keys.CompressedHash) {
		return bitcoin.MatchSegwitV0, true
	}
	if mask.P2SH && maybeFunded20(sets.P2SHBloom, sets.P2SH, keys.P2SHHash) {
		return bitcoin.MatchP2SH, true
	}
	if mask.Taproot && maybeFunded32(sets.TaprootV1Bloom, sets.TaprootV1, keys.TaprootKey) {
		return bitcoin.MatchTaproot, true
	}
	return 0, false
}

func (s *FundedSets) addAddress(addr string) {
	kind, hash, err := bitcoin.DecodeFundedAddressForLoad(addr)
	if err != nil {
		s.Other = append(s.Other, addr)
		return
	}

	switch kind {
	case bitcoin.KindLegacy:
		s.Legacy = appendHash20(s.Legacy, hash)
	case bitcoin.KindP2SH:
		s.P2SH = appendHash20(s.P2SH, hash)
	case bitcoin.KindSegwitV0:
		s.SegwitV0 = appendHash20(s.SegwitV0, hash)
	case bitcoin.KindTaprootV1:
		s.TaprootV1 = appendHash32(s.TaprootV1, hash)
	}
}