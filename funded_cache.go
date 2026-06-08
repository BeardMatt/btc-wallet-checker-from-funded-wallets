package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
)

const fundedCacheFile = "funded.cache"

func fundedCachePath(minBalance uint64) string {
	if minBalance == 0 || minBalance == defaultMinBalanceSats {
		return fundedCacheFile
	}
	return fmt.Sprintf("funded.%d.cache", minBalance)
}

var fundedCacheMagic = [4]byte{'B', 'F', 'N', 'D'}

const (
	fundedCacheVersionV3   uint32 = 3
	fundedCacheHeaderSize        = 4 + 4 + 8 + 8 + 4*4 // magic + version + mtime + min_balance + 4 counts
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
		binary.LittleEndian.PutUint32(header[4:8], fundedCacheVersionV3)
		binary.LittleEndian.PutUint64(header[8:16], uint64(srcMtime.UnixNano()))
		minBal := sets.MinBalanceSats
		if minBal == 0 {
			minBal = defaultMinBalanceSats
		}
		binary.LittleEndian.PutUint64(header[16:24], minBal)
		binary.LittleEndian.PutUint32(header[24:28], uint32(len(sets.Legacy)))
		binary.LittleEndian.PutUint32(header[28:32], uint32(len(sets.P2SH)))
		binary.LittleEndian.PutUint32(header[32:36], uint32(len(sets.SegwitV0)))
		binary.LittleEndian.PutUint32(header[36:40], uint32(len(sets.TaprootV1)))

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
		if err := writeBalanceSlice(f, sets.LegacyBalance); err != nil {
			return err
		}
		if err := writeBalanceSlice(f, sets.P2SHBalance); err != nil {
			return err
		}
		if err := writeBalanceSlice(f, sets.SegwitV0Balance); err != nil {
			return err
		}
		if err := writeBalanceSlice(f, sets.TaprootV1Balance); err != nil {
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

func writeBalanceSlice(w io.Writer, balances []uint64) error {
	if len(balances) == 0 {
		return nil
	}
	buf := make([]byte, len(balances)*8)
	for i, b := range balances {
		binary.LittleEndian.PutUint64(buf[i*8:(i+1)*8], b)
	}
	_, err := w.Write(buf)
	return err
}

func readFundedCache(path string, expectedMtime time.Time, minBalance uint64) (FundedSets, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FundedSets{}, err
	}
	return decodeFundedCache(data, expectedMtime, minBalance)
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

func removeFundedCache() {
	if err := os.Remove(fundedCacheFile); err != nil && !os.IsNotExist(err) {
		ui().Warnf("could not remove %s: %v\n", fundedCacheFile, err)
	}
}