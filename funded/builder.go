package funded

import (
	"bytes"
	"sort"

	"btcfind/bitcoin"
)

// Builder accumulates funded addresses while parsing TSV data.
type Builder struct {
	legacy     map[[20]byte]uint64
	p2sh       map[[20]byte]uint64
	segwit     map[[20]byte]uint64
	taproot    map[[32]byte]uint64
	other      []string
	minBalance uint64
}

// NewBuilder creates a builder that keeps addresses with balance >= minBalance sats.
func NewBuilder(minBalance uint64) *Builder {
	return &Builder{
		legacy:     make(map[[20]byte]uint64),
		p2sh:       make(map[[20]byte]uint64),
		segwit:     make(map[[20]byte]uint64),
		taproot:    make(map[[32]byte]uint64),
		other:      make([]string, 0, 600_000),
		minBalance: minBalance,
	}
}

// Add inserts an address and balance into the builder.
func (b *Builder) Add(addr string, balance uint64) {
	if balance < b.minBalance {
		return
	}
	kind, hash, err := bitcoin.DecodeFundedAddressForLoad(addr)
	if err != nil {
		b.other = append(b.other, addr)
		return
	}

	switch kind {
	case bitcoin.KindLegacy:
		mergeBalance20(b.legacy, hash, balance)
	case bitcoin.KindP2SH:
		mergeBalance20(b.p2sh, hash, balance)
	case bitcoin.KindSegwitV0:
		mergeBalance20(b.segwit, hash, balance)
	case bitcoin.KindTaprootV1:
		mergeBalance32(b.taproot, hash, balance)
	}
}

func mergeBalance20(m map[[20]byte]uint64, hash []byte, balance uint64) {
	if len(hash) != 20 {
		return
	}
	var key [20]byte
	copy(key[:], hash)
	if prev, ok := m[key]; !ok || balance > prev {
		m[key] = balance
	}
}

func mergeBalance32(m map[[32]byte]uint64, hash []byte, balance uint64) {
	if len(hash) != 32 {
		return
	}
	var key [32]byte
	copy(key[:], hash)
	if prev, ok := m[key]; !ok || balance > prev {
		m[key] = balance
	}
}

// Build returns sorted Sets from accumulated entries.
func (b *Builder) Build() Sets {
	return Sets{
		Legacy:           sortedHash20(b.legacy),
		LegacyBalance:    sortedBalances20(b.legacy),
		P2SH:             sortedHash20(b.p2sh),
		P2SHBalance:      sortedBalances20(b.p2sh),
		SegwitV0:         sortedHash20(b.segwit),
		SegwitV0Balance:  sortedBalances20(b.segwit),
		TaprootV1:        sortedHash32(b.taproot),
		TaprootV1Balance: sortedBalances32(b.taproot),
		Other:            b.other,
		MinBalanceSats:   b.minBalance,
	}
}

func sortedHash20(m map[[20]byte]uint64) [][20]byte {
	keys := make([][20]byte, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i][:], keys[j][:]) < 0
	})
	return keys
}

func sortedBalances20(m map[[20]byte]uint64) []uint64 {
	keys := sortedHash20(m)
	balances := make([]uint64, len(keys))
	for i, k := range keys {
		balances[i] = m[k]
	}
	return balances
}

func sortedHash32(m map[[32]byte]uint64) [][32]byte {
	keys := make([][32]byte, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i][:], keys[j][:]) < 0
	})
	return keys
}

func sortedBalances32(m map[[32]byte]uint64) []uint64 {
	keys := sortedHash32(m)
	balances := make([]uint64, len(keys))
	for i, k := range keys {
		balances[i] = m[k]
	}
	return balances
}