package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"time"

	"btcfind/bitcoin"
)

const keyBatchSize = 128

func main() {
	cfg, err := parseCLI(os.Args[1:])
	if err != nil {
		ui().PrintfErr("%v\n", err)
		printUsage()
		return
	}

	appUI = NewUI(UIConfig{
		NoColor: cfg.noColor,
		Quiet:   cfg.quiet,
		Verbose: cfg.verbose,
	})

	workers := cfg.threads
	if workers < 1 {
		workers = runtime.NumCPU()
	}

	appUI.Banner()
	appUI.Section(fmt.Sprintf("Workers  %d", workers))
	if cfg.simulateHit {
		appUI.Infof("  Simulate hit at key %d\n", cfg.simulateHitAt)
	}
	if cfg.injectIndexHit != "" {
		appUI.Infof("  Inject index hit: bucket=%q key=%d\n", cfg.injectIndexHit, cfg.simulateHitAt)
	}

	ensureFunded()
	fundedSets := loadFunded()

	appUI.Section("Search")
	appUI.BeginSearch(cfg.numKeys)

	start := time.Now()
	ch := newWallet(workers)
	lastProgress := time.Now()

	processed := 0
	for processed < cfg.numKeys {
		batch := <-ch
		for _, wallet := range batch {
			if processed >= cfg.numKeys {
				break
			}

			keyIndex := processed + 1
			keys := wallet.Keys
			if cfg.injectIndexHit != "" && keyIndex == cfg.simulateHitAt {
				fundedAddr, err := applyInjectIndexHit(fundedSets, &keys, cfg.injectIndexHit)
				if err != nil {
					panic(err)
				}
				appUI.Infof("  inject-index-hit: bucket=%s funded_address=%s\n", cfg.injectIndexHit, fundedAddr)
			}

			kind, ok := matchFunded(fundedSets, keys)
			if ok {
				wallet.Keys = keys
				appUI.PrintHit(wallet, kind, keyIndex, false)
			}

			if cfg.verifyLookup && keyIndex == cfg.simulateHitAt {
				printLookupVerify(fundedSets, keys, kind, ok)
			}

			if cfg.simulateHit && keyIndex == cfg.simulateHitAt && !ok {
				wallet.Keys = keys
				appUI.PrintHit(wallet, simulateDisplayKind(keys, kind, ok), keyIndex, true)
			}

			processed++
		}

		if time.Since(lastProgress) >= 100*time.Millisecond {
			appUI.SearchProgress(processed, cfg.numKeys, time.Since(start))
			lastProgress = time.Now()
		}
	}

	took := time.Since(start)
	avg := float64(cfg.numKeys) / took.Seconds()
	appUI.PrintSummary(took, cfg.numKeys, avg)
}

func newWallet(n int) chan []bitcoin.Wallet {
	ch := make(chan []bitcoin.Wallet, n)
	for range n {
		go func() {
			batch := make([]bitcoin.Wallet, 0, keyBatchSize)
			for {
				batch = append(batch, bitcoin.GenKeypair())
				if len(batch) >= keyBatchSize {
					ch <- batch
					batch = make([]bitcoin.Wallet, 0, keyBatchSize)
				}
			}
		}()
	}
	return ch
}

func loadFunded() FundedSets {
	loadStart := time.Now()

	tsvInfo, err := os.Stat("funded.tsv")
	if err != nil {
		panic("couldn't stat funded.tsv")
	}
	srcMtime := tsvInfo.ModTime()
	fileSize := tsvInfo.Size()

	appUI.Section("Loading funded wallets")

	if sets, err := readFundedCache(fundedCacheFile, srcMtime); err == nil {
		if !sets.bloomsComplete() {
			appUI.ProgressIndeterminate("Upgrading cache (building bloom filters)")
			sets.ensureBlooms()
			appUI.ClearProgress()
			if err := writeFundedCache(fundedCacheFile, sets, srcMtime); err != nil {
				appUI.Warnf("could not upgrade cache: %v\n", err)
			} else {
				appUI.Infof("  Upgraded funded.cache to v2 (with bloom filters)\n")
			}
		}
		appUI.LogFundedLoad(sets, time.Since(loadStart), true)
		return sets
	}

	sets := parseFundedTSV(fileSize)
	appUI.ProgressIndeterminate("Sorting funded wallets")
	sortStart := time.Now()
	sets.Sort()
	appUI.ClearProgress()
	appUI.Infof("  Sorted in %.2fs\n", time.Since(sortStart).Seconds())

	sets.ensureBlooms()

	if err := writeFundedCache(fundedCacheFile, sets, srcMtime); err != nil {
		appUI.Warnf("could not write cache: %v\n", err)
	} else {
		appUI.Infof("  Wrote funded.cache\n")
	}

	appUI.LogFundedLoad(sets, time.Since(loadStart), false)
	return sets
}

func parseFundedTSV(fileSize int64) FundedSets {
	file, err := os.Open("funded.tsv")
	if err != nil {
		panic("couldn't open funded.tsv")
	}
	defer file.Close()

	counter := &byteCounter{r: file}
	scanner := bufio.NewScanner(counter)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	if scanner.Scan() {
		// header consumed
	}

	sets := FundedSets{
		Legacy:    make([][20]byte, 0, 16_000_000),
		P2SH:      make([][20]byte, 0, 10_000_000),
		SegwitV0:  make([][20]byte, 0, 6_000_000),
		TaprootV1: make([][32]byte, 0, 1_000_000),
		Other:     make([]string, 0, 600_000),
	}

	lastProgress := time.Now()
	for scanner.Scan() {
		if time.Since(lastProgress) >= 200*time.Millisecond {
			appUI.Progress("Parsing funded.tsv", counter.n, fileSize)
			lastProgress = time.Now()
		}
		addr, balance, ok := parseTSVLine(scanner.Bytes())
		if !ok || balance < 30000 {
			continue
		}
		sets.addAddress(string(addr))
	}
	appUI.ClearProgress()
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	return sets
}