package main

import (
	"bytes"
	"sort"

	"btcfind/bitcoin"
)

type FundedSets struct {
	Legacy    [][20]byte
	P2SH      [][20]byte
	SegwitV0  [][20]byte
	TaprootV1 [][32]byte
	Other     []string // non-standard entries that fail address decode
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

func inFundedOther(set []string, address string) bool {
	idx := sort.SearchStrings(set, address)
	return idx < len(set) && set[idx] == address
}

func inFunded(sets FundedSets, address string) bool {
	kind, hash, err := bitcoin.DecodeFundedAddress(address)
	if err != nil {
		return inFundedOther(sets.Other, address)
	}

	switch kind {
	case bitcoin.KindLegacy:
		var key [20]byte
		copy(key[:], hash)
		return inFunded20(sets.Legacy, key)
	case bitcoin.KindP2SH:
		var key [20]byte
		copy(key[:], hash)
		return inFunded20(sets.P2SH, key)
	case bitcoin.KindSegwitV0:
		var key [20]byte
		copy(key[:], hash)
		return inFunded20(sets.SegwitV0, key)
	case bitcoin.KindTaprootV1:
		var key [32]byte
		copy(key[:], hash)
		return inFunded32(sets.TaprootV1, key)
	default:
		return false
	}
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