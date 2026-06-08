package funded

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"
)

// LoadOpts configures funded set loading from cache or TSV.
type LoadOpts struct {
	MinBalance uint64
	TSVPath    string
	Reporter   Reporter
}

// Load loads funded sets from cache when valid, otherwise parses TSV and writes cache.
func Load(opts LoadOpts) (Sets, error) {
	rep := opts.Reporter
	if rep == nil {
		rep = NopReporter{}
	}

	minBalance := opts.MinBalance
	if minBalance == 0 {
		minBalance = DefaultMinBalanceSats
	}

	tsvPath := opts.TSVPath
	if tsvPath == "" {
		tsvPath = DefaultTSVFile
	}

	loadStart := time.Now()

	tsvInfo, err := os.Stat(tsvPath)
	if err != nil {
		return Sets{}, fmt.Errorf("stat %s: %w", tsvPath, err)
	}
	srcMtime := tsvInfo.ModTime()
	fileSize := tsvInfo.Size()

	rep.Section("Loading funded wallets")

	var sets Sets
	var cacheErr error
	cachePath := CachePath(minBalance)
	sets, cacheErr = MmapLoad(cachePath, srcMtime, minBalance)
	if cacheErr != nil {
		sets, cacheErr = ReadCache(cachePath, srcMtime, minBalance)
	}
	if cacheErr == nil {
		if !sets.bloomsComplete() {
			rep.ProgressIndeterminate("Upgrading cache (building bloom filters)")
			sets.EnsureBlooms(rep)
			rep.ClearProgress()
			if err := WriteCache(cachePath, sets, srcMtime); err != nil {
				rep.Warnf("could not upgrade cache: %v\n", err)
			}
		}
		rep.LogFundedLoad(sets, time.Since(loadStart), true)
		return sets, nil
	}

	sets, err = parseTSV(tsvPath, fileSize, minBalance, rep)
	if err != nil {
		return Sets{}, err
	}

	rep.ProgressIndeterminate("Sorting funded wallets")
	sortStart := time.Now()
	sets.Sort()
	rep.ClearProgress()
	rep.Infof("  Sorted in %.2fs\n", time.Since(sortStart).Seconds())

	sets.EnsureBlooms(rep)

	if err := WriteCache(cachePath, sets, srcMtime); err != nil {
		rep.Warnf("could not write cache: %v\n", err)
	} else {
		rep.Infof("  Wrote %s\n", cachePath)
	}

	rep.LogFundedLoad(sets, time.Since(loadStart), false)
	return sets, nil
}

func parseTSV(path string, fileSize int64, minBalance uint64, rep Reporter) (Sets, error) {
	file, err := os.Open(path)
	if err != nil {
		return Sets{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	counter := &byteCounter{r: file}
	scanner := bufio.NewScanner(counter)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	if scanner.Scan() {
		// header consumed
	}

	builder := NewBuilder(minBalance)

	lastProgress := time.Now()
	for scanner.Scan() {
		if time.Since(lastProgress) >= 200*time.Millisecond {
			rep.Progress("Parsing funded.tsv", counter.n, fileSize)
			lastProgress = time.Now()
		}
		addr, balance, ok := ParseTSVLine(scanner.Bytes())
		if !ok {
			continue
		}
		builder.Add(string(addr), uint64(balance))
	}
	rep.ClearProgress()
	if err := scanner.Err(); err != nil {
		return Sets{}, err
	}

	return builder.Build(), nil
}

type byteCounter struct {
	r io.Reader
	n int64
}

func (b *byteCounter) Read(p []byte) (int, error) {
	n, err := b.r.Read(p)
	b.n += int64(n)
	return n, err
}