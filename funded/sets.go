package funded

import (
	"bytes"
	"sort"
	"sync"
	"time"

	"btcfind/bitcoin"

	"github.com/bits-and-blooms/bloom/v3"
)

const (
	// DefaultMinBalanceSats is the minimum address balance indexed by default.
	DefaultMinBalanceSats = 30000

	// BloomFalsePositiveRate is the target false-positive rate for bloom filters.
	BloomFalsePositiveRate = 0.0001
)

// Sets holds partitioned funded address hashes and optional bloom filters.
type Sets struct {
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

func (s Sets) Total() int {
	return len(s.Legacy) + len(s.P2SH) + len(s.SegwitV0) + len(s.TaprootV1) + len(s.Other)
}

func (s *Sets) Sort() {
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

func (s Sets) bloomsComplete() bool {
	return bloomReady(s.LegacyBloom, len(s.Legacy)) &&
		bloomReady(s.P2SHBloom, len(s.P2SH)) &&
		bloomReady(s.SegwitV0Bloom, len(s.SegwitV0)) &&
		bloomReady(s.TaprootV1Bloom, len(s.TaprootV1))
}

func bloomReady(bf *bloom.BloomFilter, entries int) bool {
	return entries == 0 || bf != nil
}

// EnsureBlooms builds bloom filters when any bucket is missing one.
func (s *Sets) EnsureBlooms(rep Reporter) {
	if s.bloomsComplete() {
		return
	}
	s.BuildBlooms(rep)
}

// BuildBlooms constructs bloom filters for all non-empty buckets.
func (s *Sets) BuildBlooms(rep Reporter) {
	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		s.LegacyBloom = BuildBloom20(s.Legacy)
		wg.Done()
	}()
	go func() {
		s.P2SHBloom = BuildBloom20(s.P2SH)
		wg.Done()
	}()
	go func() {
		s.SegwitV0Bloom = BuildBloom20(s.SegwitV0)
		wg.Done()
	}()
	go func() {
		s.TaprootV1Bloom = BuildBloom32(s.TaprootV1)
		wg.Done()
	}()
	wg.Wait()

	elapsed := time.Since(start)
	if rep != nil {
		rep.LogBloomBuild(*s, elapsed)
		if rep.Verbose() {
			rep.PrintlnErr("Bloom filter details:")
			logBloomBucket(rep, "legacy", s.LegacyBloom, len(s.Legacy))
			logBloomBucket(rep, "p2sh", s.P2SHBloom, len(s.P2SH))
			logBloomBucket(rep, "segwit", s.SegwitV0Bloom, len(s.SegwitV0))
			logBloomBucket(rep, "taproot", s.TaprootV1Bloom, len(s.TaprootV1))
		}
	}
}

// BuildBloom20 builds a bloom filter for a sorted [20]byte hash set.
func BuildBloom20(set [][20]byte) *bloom.BloomFilter {
	if len(set) == 0 {
		return nil
	}
	bf := bloom.NewWithEstimates(uint(len(set)), BloomFalsePositiveRate)
	for _, entry := range set {
		bf.Add(entry[:])
	}
	return bf
}

// BuildBloom32 builds a bloom filter for a sorted [32]byte hash set.
func BuildBloom32(set [][32]byte) *bloom.BloomFilter {
	if len(set) == 0 {
		return nil
	}
	bf := bloom.NewWithEstimates(uint(len(set)), BloomFalsePositiveRate)
	for _, entry := range set {
		bf.Add(entry[:])
	}
	return bf
}

func logBloomBucket(rep Reporter, name string, bf *bloom.BloomFilter, entries int) {
	if bf == nil {
		return
	}
	fp := bloom.EstimateFalsePositiveRate(bf.Cap(), bf.K(), uint(entries))
	rep.PrintfErr(
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
	return InFunded20(set, key)
}

func maybeFunded32(bf *bloom.BloomFilter, set [][32]byte, key [32]byte) bool {
	if bf != nil && !bf.Test(key[:]) {
		return false
	}
	return InFunded32(set, key)
}

// InFunded20 reports whether key is present in a sorted [20]byte set.
func InFunded20(set [][20]byte, key [20]byte) bool {
	idx := sort.Search(len(set), func(i int) bool {
		return bytes.Compare(set[i][:], key[:]) >= 0
	})
	return idx < len(set) && set[idx] == key
}

// InFunded32 reports whether key is present in a sorted [32]byte set.
func InFunded32(set [][32]byte, key [32]byte) bool {
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

// BalanceForMatch returns the satoshi balance for a funded match kind.
func (s Sets) BalanceForMatch(kind bitcoin.MatchKind, keys bitcoin.LookupKeys) uint64 {
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

// Match reports whether keys match any funded address in mask-selected formats.
func (s Sets) Match(keys bitcoin.LookupKeys, mask bitcoin.FormatMask) (bitcoin.MatchKind, bool) {
	if mask.LegacyCompressed && maybeFunded20(s.LegacyBloom, s.Legacy, keys.CompressedHash) {
		return bitcoin.MatchLegacyCompressed, true
	}
	if mask.LegacyUncompressed && maybeFunded20(s.LegacyBloom, s.Legacy, keys.UncompressedHash) {
		return bitcoin.MatchLegacyUncompressed, true
	}
	if mask.Segwit && maybeFunded20(s.SegwitV0Bloom, s.SegwitV0, keys.CompressedHash) {
		return bitcoin.MatchSegwitV0, true
	}
	if mask.P2SH && maybeFunded20(s.P2SHBloom, s.P2SH, keys.P2SHHash) {
		return bitcoin.MatchP2SH, true
	}
	if mask.Taproot && maybeFunded32(s.TaprootV1Bloom, s.TaprootV1, keys.TaprootKey) {
		return bitcoin.MatchTaproot, true
	}
	return 0, false
}

// Match is a package-level helper compatible with legacy matchFunded call sites.
func Match(sets Sets, keys bitcoin.LookupKeys, mask bitcoin.FormatMask) (bitcoin.MatchKind, bool) {
	return sets.Match(keys, mask)
}

// LookupHit20 tests bloom then set membership for a 20-byte key.
func LookupHit20(bf *bloom.BloomFilter, set [][20]byte, key [20]byte) bool {
	if bf != nil && !bf.Test(key[:]) {
		return false
	}
	return InFunded20(set, key)
}

// LookupHit32 tests bloom then set membership for a 32-byte key.
func LookupHit32(bf *bloom.BloomFilter, set [][32]byte, key [32]byte) bool {
	if bf != nil && !bf.Test(key[:]) {
		return false
	}
	return InFunded32(set, key)
}

// MatchKindName returns a stable label for a match kind.
func MatchKindName(kind bitcoin.MatchKind) string {
	switch kind {
	case bitcoin.MatchLegacyCompressed:
		return "legacy_compressed"
	case bitcoin.MatchLegacyUncompressed:
		return "legacy_uncompressed"
	case bitcoin.MatchSegwitV0:
		return "segwit_v0"
	case bitcoin.MatchP2SH:
		return "p2sh"
	case bitcoin.MatchTaproot:
		return "taproot"
	default:
		return "unknown"
	}
}