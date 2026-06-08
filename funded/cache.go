package funded

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
)

const (
	// DefaultCacheFile is the on-disk cache path for DefaultMinBalanceSats.
	DefaultCacheFile = "funded.cache"

	// DefaultTSVFile is the default funded.tsv filename.
	DefaultTSVFile = "funded.tsv"

	// CacheVersionV3 is the current funded cache format version.
	CacheVersionV3 uint32 = 3

	// CacheHeaderSize is the byte length of the funded cache header.
	CacheHeaderSize = 4 + 4 + 8 + 8 + 4*4 // magic + version + mtime + min_balance + 4 counts
)

// CacheMagic identifies a valid funded cache file.
var CacheMagic = [4]byte{'B', 'F', 'N', 'D'}

// CachePath returns the cache file path for the given minimum balance threshold.
func CachePath(minBalance uint64) string {
	if minBalance == 0 || minBalance == DefaultMinBalanceSats {
		return DefaultCacheFile
	}
	return fmt.Sprintf("funded.%d.cache", minBalance)
}

// CacheHeader holds metadata stored in a funded cache file header.
type CacheHeader struct {
	Mtime          time.Time
	MinBalanceSats uint64
	LegacyCount    int
	P2SHCount      int
	SegwitCount    int
	TaprootCount   int
}

// ReadCacheHeader reads and validates the header from a cache file.
func ReadCacheHeader(path string) (CacheHeader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CacheHeader{}, err
	}
	return ParseCacheHeader(data)
}

// ParseCacheHeader parses a cache header from raw bytes.
func ParseCacheHeader(data []byte) (CacheHeader, error) {
	if len(data) < CacheHeaderSize {
		return CacheHeader{}, errors.New("cache data too small")
	}
	header := data[:CacheHeaderSize]

	if [4]byte(header[0:4]) != CacheMagic {
		return CacheHeader{}, errors.New("invalid cache magic")
	}
	version := binary.LittleEndian.Uint32(header[4:8])
	if version != CacheVersionV3 {
		return CacheHeader{}, errors.New("cache version needs rebuild")
	}

	mtime := time.Unix(0, int64(binary.LittleEndian.Uint64(header[8:16])))
	minBalance := binary.LittleEndian.Uint64(header[16:24])

	return CacheHeader{
		Mtime:          mtime,
		MinBalanceSats: minBalance,
		LegacyCount:    int(binary.LittleEndian.Uint32(header[24:28])),
		P2SHCount:      int(binary.LittleEndian.Uint32(header[28:32])),
		SegwitCount:    int(binary.LittleEndian.Uint32(header[32:36])),
		TaprootCount:   int(binary.LittleEndian.Uint32(header[36:40])),
	}, nil
}

// CacheETag returns an ETag string of the form "mtimeNano-minBalance" for HTTP cache sync.
func CacheETag(path string) (string, error) {
	hdr, err := ReadCacheHeader(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d-%d", hdr.Mtime.UnixNano(), hdr.MinBalanceSats), nil
}

// WriteCache atomically writes sets to path with srcMtime recorded in the header.
func WriteCache(path string, sets Sets, srcMtime time.Time) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	writeErr := func() error {
		header := make([]byte, CacheHeaderSize)
		copy(header[0:4], CacheMagic[:])
		binary.LittleEndian.PutUint32(header[4:8], CacheVersionV3)
		binary.LittleEndian.PutUint64(header[8:16], uint64(srcMtime.UnixNano()))
		minBal := sets.MinBalanceSats
		if minBal == 0 {
			minBal = DefaultMinBalanceSats
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

// ReadCache loads sets from path, validating mtime and minBalance against the header.
func ReadCache(path string, expectedMtime time.Time, minBalance uint64) (Sets, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Sets{}, err
	}
	return DecodeCache(data, expectedMtime, minBalance)
}

// DecodeCache decodes sets from raw cache bytes.
func DecodeCache(data []byte, expectedMtime time.Time, minBalance uint64) (Sets, error) {
	if len(data) < CacheHeaderSize {
		return Sets{}, errors.New("cache data too small")
	}
	header := data[:CacheHeaderSize]

	if [4]byte(header[0:4]) != CacheMagic {
		return Sets{}, errors.New("invalid cache magic")
	}
	version := binary.LittleEndian.Uint32(header[4:8])
	if version != CacheVersionV3 {
		return Sets{}, errors.New("cache version needs rebuild")
	}
	cacheMtime := time.Unix(0, int64(binary.LittleEndian.Uint64(header[8:16])))
	if !cacheMtime.Equal(expectedMtime) {
		return Sets{}, errors.New("cache mtime mismatch")
	}

	cacheMinBalance := binary.LittleEndian.Uint64(header[16:24])
	if cacheMinBalance != minBalance {
		return Sets{}, errors.New("cache min_balance mismatch")
	}

	legacyCount := int(binary.LittleEndian.Uint32(header[24:28]))
	p2shCount := int(binary.LittleEndian.Uint32(header[28:32]))
	segwitCount := int(binary.LittleEndian.Uint32(header[32:36]))
	taprootCount := int(binary.LittleEndian.Uint32(header[36:40]))

	offset := CacheHeaderSize

	legacy, offset, err := viewHash20At(data, offset, legacyCount)
	if err != nil {
		return Sets{}, err
	}
	p2sh, offset, err := viewHash20At(data, offset, p2shCount)
	if err != nil {
		return Sets{}, err
	}
	segwit, offset, err := viewHash20At(data, offset, segwitCount)
	if err != nil {
		return Sets{}, err
	}
	taproot, offset, err := viewHash32At(data, offset, taprootCount)
	if err != nil {
		return Sets{}, err
	}

	legacyBal, offset, err := viewBalanceAt(data, offset, legacyCount)
	if err != nil {
		return Sets{}, err
	}
	p2shBal, offset, err := viewBalanceAt(data, offset, p2shCount)
	if err != nil {
		return Sets{}, err
	}
	segwitBal, offset, err := viewBalanceAt(data, offset, segwitCount)
	if err != nil {
		return Sets{}, err
	}
	taprootBal, offset, err := viewBalanceAt(data, offset, taprootCount)
	if err != nil {
		return Sets{}, err
	}

	sets := Sets{
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
		return Sets{}, err
	}
	sets.P2SHBloom, offset, err = readBloomAt(data, offset)
	if err != nil {
		return Sets{}, err
	}
	sets.SegwitV0Bloom, offset, err = readBloomAt(data, offset)
	if err != nil {
		return Sets{}, err
	}
	sets.TaprootV1Bloom, offset, err = readBloomAt(data, offset)
	if err != nil {
		return Sets{}, err
	}

	return sets, nil
}

// RemoveCache deletes a cache file, ignoring not-exist errors.
func RemoveCache(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// RemoveDefaultCache deletes the default funded.cache file.
func RemoveDefaultCache() error {
	return RemoveCache(DefaultCacheFile)
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