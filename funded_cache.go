package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
)

const fundedCacheFile = "funded.cache"

var fundedCacheMagic = [4]byte{'B', 'F', 'N', 'D'}

const (
	fundedCacheVersionV1   uint32 = 1
	fundedCacheVersionV2   uint32 = 2
	fundedCacheHeaderSize         = 4 + 4 + 8 + 4*4 // magic + version + mtime + 4 counts
)

func writeFundedCache(path string, sets FundedSets, srcMtime time.Time) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	writeErr := func() error {
		header := make([]byte, fundedCacheHeaderSize)
		copy(header[0:4], fundedCacheMagic[:])
		binary.LittleEndian.PutUint32(header[4:8], fundedCacheVersionV2)
		binary.LittleEndian.PutUint64(header[8:16], uint64(srcMtime.UnixNano()))
		binary.LittleEndian.PutUint32(header[16:20], uint32(len(sets.Legacy)))
		binary.LittleEndian.PutUint32(header[20:24], uint32(len(sets.P2SH)))
		binary.LittleEndian.PutUint32(header[24:28], uint32(len(sets.SegwitV0)))
		binary.LittleEndian.PutUint32(header[28:32], uint32(len(sets.TaprootV1)))

		if _, err := f.Write(header); err != nil {
			return err
		}
		if err := writeHash20Slice(f, sets.Legacy); err != nil {
			return err
		}
		if err := writeHash20Slice(f, sets.P2SH); err != nil {
			return err
		}
		if err := writeHash20Slice(f, sets.SegwitV0); err != nil {
			return err
		}
		if err := writeHash32Slice(f, sets.TaprootV1); err != nil {
			return err
		}
		if err := writeBloom(f, sets.LegacyBloom); err != nil {
			return err
		}
		if err := writeBloom(f, sets.P2SHBloom); err != nil {
			return err
		}
		if err := writeBloom(f, sets.SegwitV0Bloom); err != nil {
			return err
		}
		return writeBloom(f, sets.TaprootV1Bloom)
	}()

	closeErr := f.Close()
	if writeErr != nil {
		os.Remove(tmp)
		return writeErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, path)
}

func writeHash20Slice(w io.Writer, set [][20]byte) error {
	if len(set) == 0 {
		return nil
	}
	buf := make([]byte, len(set)*20)
	for i, entry := range set {
		copy(buf[i*20:(i+1)*20], entry[:])
	}
	_, err := w.Write(buf)
	return err
}

func writeHash32Slice(w io.Writer, set [][32]byte) error {
	if len(set) == 0 {
		return nil
	}
	buf := make([]byte, len(set)*32)
	for i, entry := range set {
		copy(buf[i*32:(i+1)*32], entry[:])
	}
	_, err := w.Write(buf)
	return err
}

func readFundedCache(path string, expectedMtime time.Time) (FundedSets, error) {
	f, err := os.Open(path)
	if err != nil {
		return FundedSets{}, err
	}
	defer f.Close()

	header := make([]byte, fundedCacheHeaderSize)
	if _, err := io.ReadFull(f, header); err != nil {
		return FundedSets{}, fmt.Errorf("read cache header: %w", err)
	}

	if [4]byte(header[0:4]) != fundedCacheMagic {
		return FundedSets{}, errors.New("invalid cache magic")
	}
	version := binary.LittleEndian.Uint32(header[4:8])
	if version != fundedCacheVersionV1 && version != fundedCacheVersionV2 {
		return FundedSets{}, errors.New("unsupported cache version")
	}
	cacheMtime := time.Unix(0, int64(binary.LittleEndian.Uint64(header[8:16])))
	if !cacheMtime.Equal(expectedMtime) {
		return FundedSets{}, errors.New("cache mtime mismatch")
	}

	legacyCount := int(binary.LittleEndian.Uint32(header[16:20]))
	p2shCount := int(binary.LittleEndian.Uint32(header[20:24]))
	segwitCount := int(binary.LittleEndian.Uint32(header[24:28]))
	taprootCount := int(binary.LittleEndian.Uint32(header[28:32]))

	legacy, err := readHash20Slice(f, legacyCount)
	if err != nil {
		return FundedSets{}, err
	}
	p2sh, err := readHash20Slice(f, p2shCount)
	if err != nil {
		return FundedSets{}, err
	}
	segwit, err := readHash20Slice(f, segwitCount)
	if err != nil {
		return FundedSets{}, err
	}
	taproot, err := readHash32Slice(f, taprootCount)
	if err != nil {
		return FundedSets{}, err
	}

	sets := FundedSets{
		Legacy:    legacy,
		P2SH:      p2sh,
		SegwitV0:  segwit,
		TaprootV1: taproot,
	}

	if version == fundedCacheVersionV2 {
		sets.LegacyBloom, err = readBloom(f)
		if err != nil {
			return FundedSets{}, err
		}
		sets.P2SHBloom, err = readBloom(f)
		if err != nil {
			return FundedSets{}, err
		}
		sets.SegwitV0Bloom, err = readBloom(f)
		if err != nil {
			return FundedSets{}, err
		}
		sets.TaprootV1Bloom, err = readBloom(f)
		if err != nil {
			return FundedSets{}, err
		}
	}

	return sets, nil
}

func writeBloom(w io.Writer, bf *bloom.BloomFilter) error {
	if bf == nil {
		return binary.Write(w, binary.LittleEndian, uint32(0))
	}
	data, err := bf.MarshalBinary()
	if err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(data))); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func readBloom(r io.Reader) (*bloom.BloomFilter, error) {
	var n uint32
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return nil, fmt.Errorf("read bloom length: %w", err)
	}
	if n == 0 {
		return nil, nil
	}
	data := make([]byte, n)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("read bloom data: %w", err)
	}
	bf := &bloom.BloomFilter{}
	if err := bf.UnmarshalBinary(data); err != nil {
		return nil, fmt.Errorf("decode bloom: %w", err)
	}
	return bf, nil
}

func readHash20Slice(r io.Reader, count int) ([][20]byte, error) {
	if count == 0 {
		return nil, nil
	}
	buf := make([]byte, count*20)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("read hash20 slice: %w", err)
	}
	set := make([][20]byte, count)
	for i := range set {
		copy(set[i][:], buf[i*20:(i+1)*20])
	}
	return set, nil
}

func readHash32Slice(r io.Reader, count int) ([][32]byte, error) {
	if count == 0 {
		return nil, nil
	}
	buf := make([]byte, count*32)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("read hash32 slice: %w", err)
	}
	set := make([][32]byte, count)
	for i := range set {
		copy(set[i][:], buf[i*32:(i+1)*32])
	}
	return set, nil
}

func removeFundedCache() {
	if err := os.Remove(fundedCacheFile); err != nil && !os.IsNotExist(err) {
		ui().Warnf("could not remove %s: %v\n", fundedCacheFile, err)
	}
}