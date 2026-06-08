package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
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
	if cfg.formatsSet {
		appUI.Infof("  Formats: %s\n", cfg.formats.String())
	}
	if cfg.minBalance != defaultMinBalanceSats {
		appUI.Infof("  Min balance: %s sats\n", appUI.formatInt(int64(cfg.minBalance)))
	}

	defer unmapActiveCache()

	ensureFunded()
	fundedSets := loadFunded(cfg.minBalance)

	appUI.Section("Search")
	if cfg.forever {
		appUI.BeginForeverSearch()
		runForeverSearch(cfg, workers, fundedSets)
		return
	}

	appUI.BeginSearch(cfg.numKeys)
	hits := runBoundedSearch(cfg, workers, fundedSets, cfg.numKeys, 0)
	_ = hits
}

func runBoundedSearch(cfg cliConfig, workers int, fundedSets FundedSets, numKeys int, sessionBase uint64) uint64 {
	start := time.Now()
	ch := newWallet(workers, fundedSets, cfg.formats)
	lastProgress := time.Now()
	processed := 0
	var hits uint64

	for processed < numKeys {
		batch := <-ch
		for _, wallet := range batch {
			if processed >= numKeys {
				break
			}

			keyIndex := int(sessionBase) + processed + 1
			hit := processWallet(cfg, fundedSets, wallet, keyIndex)
			if hit {
				hits++
			}
			processed++
		}

		now := time.Now()
		if now.Sub(lastProgress) >= 100*time.Millisecond {
			appUI.SearchProgress(processed, numKeys, now.Sub(start))
			lastProgress = now
		}
	}

	took := time.Since(start)
	avg := float64(numKeys) / took.Seconds()
	appUI.PrintSummary(took, numKeys, avg)
	return hits
}

func runForeverSearch(cfg cliConfig, workers int, fundedSets FundedSets) {
	session, err := initForeverSession(cfg.resetSession)
	if err != nil {
		appUI.Warnf("could not reset session: %v\n", err)
	}
	if cfg.resetSession && !appUI.quiet {
		appUI.Infof("  Session reset — starting fresh stats in %s\n", sessionFile)
	}

	var stopFlag atomic.Bool
	var maxKeysReached bool
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		stopFlag.Store(true)
	}()

	start := time.Now()
	ch := newWallet(workers, fundedSets, cfg.formats)
	lastProgress := time.Now()
	lastCheckpoint := time.Now()

	processed := uint64(0)
	runProcessed := uint64(0)
	hits := session.TotalHits

	for !stopFlag.Load() {
		if cfg.maxKeys > 0 && runProcessed >= cfg.maxKeys {
			maxKeysReached = true
			break
		}

		batch := <-ch
		for _, wallet := range batch {
			if stopFlag.Load() {
				break
			}
			if cfg.maxKeys > 0 && runProcessed >= cfg.maxKeys {
				maxKeysReached = true
				break
			}

			keyIndex := int(session.TotalKeysTried + processed + 1)
			if processWallet(cfg, fundedSets, wallet, keyIndex) {
				hits++
			}
			processed++
			runProcessed++
		}
		if maxKeysReached {
			break
		}

		now := time.Now()
		elapsed := now.Sub(start)
		if now.Sub(lastProgress) >= 100*time.Millisecond {
			appUI.ForeverProgress(session.TotalKeysTried+processed, elapsed, hits)
			lastProgress = now
		}
		if now.Sub(lastCheckpoint) >= cfg.checkpointInterval {
			kps := float64(processed) / elapsed.Seconds()
			if kps > session.BestKeysPerSecond {
				session.BestKeysPerSecond = kps
			}
			session.TotalKeysTried += processed
			session.TotalHits = hits
			_ = writeSession(session)
			processed = 0
			start = time.Now()
			lastCheckpoint = now
		}
	}

	session.TotalKeysTried += processed
	session.TotalHits = hits
	kps := float64(session.TotalKeysTried) / time.Since(session.StartedAt).Seconds()
	if kps > session.BestKeysPerSecond {
		session.BestKeysPerSecond = kps
	}
	_ = writeSession(session)
	appUI.ClearProgress()

	switch {
	case maxKeysReached:
		appUI.Infof(
			"Run limit reached (--max-keys %s); tried %s keys this run. Session saved to %s (%s lifetime keys, %s hits)\n",
			appUI.formatInt(int64(cfg.maxKeys)),
			appUI.formatInt(int64(runProcessed)),
			sessionFile,
			appUI.formatInt(int64(session.TotalKeysTried)),
			appUI.formatInt(int64(session.TotalHits)),
		)
	case stopFlag.Load():
		appUI.Infof(
			"Interrupted; session saved to %s (%s lifetime keys, %s hits)\n",
			sessionFile,
			appUI.formatInt(int64(session.TotalKeysTried)),
			appUI.formatInt(int64(session.TotalHits)),
		)
	default:
		appUI.Infof(
			"Session checkpoint saved to %s (%s lifetime keys, %s hits)\n",
			sessionFile,
			appUI.formatInt(int64(session.TotalKeysTried)),
			appUI.formatInt(int64(session.TotalHits)),
		)
	}
}

func processWallet(cfg cliConfig, fundedSets FundedSets, wallet bitcoin.Wallet, keyIndex int) bool {
	keys := wallet.Keys
	if cfg.injectIndexHit != "" && keyIndex == cfg.simulateHitAt {
		fundedAddr, err := applyInjectIndexHit(fundedSets, &keys, cfg.injectIndexHit)
		if err != nil {
			panic(err)
		}
		appUI.Infof("  inject-index-hit: bucket=%s funded_address=%s\n", cfg.injectIndexHit, fundedAddr)
	}

	kind, ok := matchFunded(fundedSets, keys, cfg.formats)
	if ok {
		wallet.Keys = keys
		appUI.PrintHit(wallet, kind, keyIndex, fundedSets.balanceForMatch(kind, keys), false)
	}

	if cfg.verifyLookup && keyIndex == cfg.simulateHitAt {
		printLookupVerify(fundedSets, keys, kind, ok)
	}

	if cfg.simulateHit && keyIndex == cfg.simulateHitAt && !ok {
		wallet.Keys = keys
		appUI.PrintHit(wallet, simulateDisplayKind(keys, kind, ok), keyIndex, 0, true)
	}

	return ok
}

func newWallet(n int, sets FundedSets, mask bitcoin.FormatMask) chan []bitcoin.Wallet {
	ch := make(chan []bitcoin.Wallet, n)
	for range n {
		go func() {
			batch := make([]bitcoin.Wallet, 0, keyBatchSize)
			for {
				batch = append(batch, genWalletStaged(sets, mask))
				if len(batch) >= keyBatchSize {
					ch <- batch
					batch = make([]bitcoin.Wallet, 0, keyBatchSize)
				}
			}
		}()
	}
	return ch
}

func loadFunded(minBalance uint64) FundedSets {
	if minBalance == 0 {
		minBalance = defaultMinBalanceSats
	}
	loadStart := time.Now()

	tsvInfo, err := os.Stat("funded.tsv")
	if err != nil {
		panic("couldn't stat funded.tsv")
	}
	srcMtime := tsvInfo.ModTime()
	fileSize := tsvInfo.Size()

	appUI.Section("Loading funded wallets")

	var sets FundedSets
	var cacheErr error
	cachePath := fundedCachePath(minBalance)
	sets, cacheErr = mmapFundedCache(cachePath, srcMtime, minBalance)
	if cacheErr != nil {
		sets, cacheErr = readFundedCache(cachePath, srcMtime, minBalance)
	}
	if cacheErr == nil {
		if !sets.bloomsComplete() {
			appUI.ProgressIndeterminate("Upgrading cache (building bloom filters)")
			sets.ensureBlooms()
			appUI.ClearProgress()
			if err := writeFundedCache(cachePath, sets, srcMtime); err != nil {
				appUI.Warnf("could not upgrade cache: %v\n", err)
			}
		}
		appUI.LogFundedLoad(sets, time.Since(loadStart), true)
		return sets
	}

	sets = parseFundedTSV(fileSize, minBalance)
	appUI.ProgressIndeterminate("Sorting funded wallets")
	sortStart := time.Now()
	sets.Sort()
	appUI.ClearProgress()
	appUI.Infof("  Sorted in %.2fs\n", time.Since(sortStart).Seconds())

	sets.ensureBlooms()

	if err := writeFundedCache(cachePath, sets, srcMtime); err != nil {
		appUI.Warnf("could not write cache: %v\n", err)
	} else {
		appUI.Infof("  Wrote %s\n", cachePath)
	}

	appUI.LogFundedLoad(sets, time.Since(loadStart), false)
	return sets
}

func parseFundedTSV(fileSize int64, minBalance uint64) FundedSets {
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

	builder := newFundedBuilder(minBalance)

	lastProgress := time.Now()
	for scanner.Scan() {
		if time.Since(lastProgress) >= 200*time.Millisecond {
			appUI.Progress("Parsing funded.tsv", counter.n, fileSize)
			lastProgress = time.Now()
		}
		addr, balance, ok := parseTSVLine(scanner.Bytes())
		if !ok {
			continue
		}
		builder.add(string(addr), uint64(balance))
	}
	appUI.ClearProgress()
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	return builder.build()
}