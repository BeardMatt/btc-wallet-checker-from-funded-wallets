package funded

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

type mappedCache struct {
	data []byte
}

var activeMappedCache *mappedCache

// UnmapActive releases the active memory-mapped cache, if any.
func UnmapActive() {
	if activeMappedCache == nil {
		return
	}
	_ = syscall.Munmap(activeMappedCache.data)
	activeMappedCache = nil
}

// MmapLoad memory-maps path and decodes funded sets using expectedMtime and minBalance.
func MmapLoad(path string, expectedMtime time.Time, minBalance uint64) (Sets, error) {
	UnmapActive()

	f, err := os.Open(path)
	if err != nil {
		return Sets{}, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return Sets{}, err
	}
	size := int(info.Size())
	if size < CacheHeaderSize {
		return Sets{}, errors.New("cache file too small")
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return Sets{}, fmt.Errorf("mmap cache: %w", err)
	}

	sets, err := DecodeCache(data, expectedMtime, minBalance)
	if err != nil {
		_ = syscall.Munmap(data)
		return Sets{}, err
	}

	activeMappedCache = &mappedCache{data: data}
	return sets, nil
}

// LoadFromCacheFile memory-maps a cache file using mtime and min_balance from its header.
// No funded.tsv file is required.
func LoadFromCacheFile(path string) (Sets, error) {
	hdr, err := ReadCacheHeader(path)
	if err != nil {
		return Sets{}, err
	}
	return MmapLoad(path, hdr.Mtime, hdr.MinBalanceSats)
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