package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/bits-and-blooms/bloom/v3"
)

type mappedFundedCache struct {
	data []byte
}

var activeMappedCache *mappedFundedCache

func unmapActiveCache() {
	if activeMappedCache == nil {
		return
	}
	_ = syscall.Munmap(activeMappedCache.data)
	activeMappedCache = nil
}

func mmapFundedCache(path string, expectedMtime time.Time, minBalance uint64) (FundedSets, error) {
	unmapActiveCache()

	f, err := os.Open(path)
	if err != nil {
		return FundedSets{}, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return FundedSets{}, err
	}
	size := int(info.Size())
	if size < fundedCacheHeaderSize {
		return FundedSets{}, errors.New("cache file too small")
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return FundedSets{}, fmt.Errorf("mmap cache: %w", err)
	}

	sets, err := decodeFundedCache(data, expectedMtime, minBalance)
	if err != nil {
		_ = syscall.Munmap(data)
		return FundedSets{}, err
	}

	activeMappedCache = &mappedFundedCache{data: data}
	return sets, nil
}

func decodeFundedCache(data []byte, expectedMtime time.Time, minBalance uint64) (FundedSets, error) {
	if len(data) < fundedCacheHeaderSize {
		return FundedSets{}, errors.New("cache data too small")
	}
	header := data[:fundedCacheHeaderSize]

	if [4]byte(header[0:4]) != fundedCacheMagic {
		return FundedSets{}, errors.New("invalid cache magic")
	}
	version := uint32(header[4]) | uint32(header[5])<<8 | uint32(header[6])<<16 | uint32(header[7])<<24
	if version != fundedCacheVersionV3 {
		return FundedSets{}, errors.New("cache version needs rebuild")
	}
	cacheMtime := time.Unix(0, int64(
		uint64(header[8])|uint64(header[9])<<8|uint64(header[10])<<16|uint64(header[11])<<24|
			uint64(header[12])<<32|uint64(header[13])<<40|uint64(header[14])<<48|uint64(header[15])<<56,
	))
	if !cacheMtime.Equal(expectedMtime) {
		return FundedSets{}, errors.New("cache mtime mismatch")
	}

	cacheMinBalance := uint64(header[16]) | uint64(header[17])<<8 | uint64(header[18])<<16 | uint64(header[19])<<24 |
		uint64(header[20])<<32 | uint64(header[21])<<40 | uint64(header[22])<<48 | uint64(header[23])<<56
	if cacheMinBalance != minBalance {
		return FundedSets{}, errors.New("cache min_balance mismatch")
	}

	legacyCount := int(uint32(header[24]) | uint32(header[25])<<8 | uint32(header[26])<<16 | uint32(header[27])<<24)
	p2shCount := int(uint32(header[28]) | uint32(header[29])<<8 | uint32(header[30])<<16 | uint32(header[31])<<24)
	segwitCount := int(uint32(header[32]) | uint32(header[33])<<8 | uint32(header[34])<<16 | uint32(header[35])<<24)
	taprootCount := int(uint32(header[36]) | uint32(header[37])<<8 | uint32(header[38])<<16 | uint32(header[39])<<24)

	offset := fundedCacheHeaderSize

	legacy, offset, err := viewHash20At(data, offset, legacyCount)
	if err != nil {
		return FundedSets{}, err
	}
	p2sh, offset, err := viewHash20At(data, offset, p2shCount)
	if err != nil {
		return FundedSets{}, err
	}
	segwit, offset, err := viewHash20At(data, offset, segwitCount)
	if err != nil {
		return FundedSets{}, err
	}
	taproot, offset, err := viewHash32At(data, offset, taprootCount)
	if err != nil {
		return FundedSets{}, err
	}

	legacyBal, offset, err := viewBalanceAt(data, offset, legacyCount)
	if err != nil {
		return FundedSets{}, err
	}
	p2shBal, offset, err := viewBalanceAt(data, offset, p2shCount)
	if err != nil {
		return FundedSets{}, err
	}
	segwitBal, offset, err := viewBalanceAt(data, offset, segwitCount)
	if err != nil {
		return FundedSets{}, err
	}
	taprootBal, offset, err := viewBalanceAt(data, offset, taprootCount)
	if err != nil {
		return FundedSets{}, err
	}

	sets := FundedSets{
		Legacy:           legacy,
		P2SH:             p2sh,
		SegwitV0:         segwit,
		TaprootV1:        taproot,
		LegacyBalance:    legacyBal,
		P2SHBalance:      p2shBal,
		SegwitV0Balance:  segwitBal,
		TaprootV1Balance: taprootBal,
		MinBalanceSats:   cacheMinBalance,
	}

	sets.LegacyBloom, offset, err = readBloomAt(data, offset)
	if err != nil {
		return FundedSets{}, err
	}
	sets.P2SHBloom, offset, err = readBloomAt(data, offset)
	if err != nil {
		return FundedSets{}, err
	}
	sets.SegwitV0Bloom, offset, err = readBloomAt(data, offset)
	if err != nil {
		return FundedSets{}, err
	}
	sets.TaprootV1Bloom, offset, err = readBloomAt(data, offset)
	if err != nil {
		return FundedSets{}, err
	}

	return sets, nil
}

func viewHash20At(data []byte, offset, count int) ([][20]byte, int, error) {
	if count == 0 {
		return nil, offset, nil
	}
	need := count * 20
	if offset+need > len(data) {
		return nil, offset, errors.New("truncated hash20 region")
	}
	region := data[offset : offset+need]
	slice := unsafe.Slice((*[20]byte)(unsafe.Pointer(&region[0])), count)
	return slice, offset + need, nil
}

func viewHash32At(data []byte, offset, count int) ([][32]byte, int, error) {
	if count == 0 {
		return nil, offset, nil
	}
	need := count * 32
	if offset+need > len(data) {
		return nil, offset, errors.New("truncated hash32 region")
	}
	region := data[offset : offset+need]
	slice := unsafe.Slice((*[32]byte)(unsafe.Pointer(&region[0])), count)
	return slice, offset + need, nil
}

func viewBalanceAt(data []byte, offset, count int) ([]uint64, int, error) {
	if count == 0 {
		return nil, offset, nil
	}
	need := count * 8
	if offset+need > len(data) {
		return nil, offset, errors.New("truncated balance region")
	}
	region := data[offset : offset+need]
	slice := unsafe.Slice((*uint64)(unsafe.Pointer(&region[0])), count)
	return slice, offset + need, nil
}

func readBloomAt(data []byte, offset int) (*bloom.BloomFilter, int, error) {
	if offset+4 > len(data) {
		return nil, offset, errors.New("truncated bloom length")
	}
	n := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	if n == 0 {
		return nil, offset, nil
	}
	if offset+int(n) > len(data) {
		return nil, offset, errors.New("truncated bloom data")
	}
	bloomData := data[offset : offset+int(n)]
	offset += int(n)
	bf := &bloom.BloomFilter{}
	if err := bf.UnmarshalBinary(bytes.Clone(bloomData)); err != nil {
		return nil, offset, fmt.Errorf("decode bloom: %w", err)
	}
	return bf, offset, nil
}