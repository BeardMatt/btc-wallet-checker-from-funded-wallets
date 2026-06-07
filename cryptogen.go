package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"time"

	"btcfind/bitcoin"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const keyBatchSize = 128

func main() {
	cfg, err := parseCLI(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		printUsage()
		return
	}

	workers := cfg.threads
	if workers < 1 {
		workers = runtime.NumCPU()
	}
	fmt.Printf("Using %d workers\n", workers)
	if cfg.simulateHit {
		fmt.Printf("Simulate hit enabled at key %d\n", cfg.simulateHitAt)
	}
	if cfg.injectIndexHit != "" {
		fmt.Printf("Inject index hit enabled for bucket %q at key %d\n", cfg.injectIndexHit, cfg.simulateHitAt)
	}

	ensureFunded()
	fundedSets := loadFunded()

	printer := message.NewPrinter(language.English)
	printer.Printf("Testing %d keys... ", cfg.numKeys)

	start := time.Now()
	ch := newWallet(workers)

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
				fmt.Printf("inject-index-hit: bucket=%s funded_address=%s\n", cfg.injectIndexHit, fundedAddr)
			}

			kind, ok := matchFunded(fundedSets, keys)
			if ok {
				wallet.Keys = keys
				printHit(wallet, kind, false)
			}

			if cfg.verifyLookup && keyIndex == cfg.simulateHitAt {
				printLookupVerify(fundedSets, keys, kind, ok)
			}

			if cfg.simulateHit && keyIndex == cfg.simulateHitAt && !ok {
				wallet.Keys = keys
				printHit(wallet, simulateDisplayKind(keys, kind, ok), true)
			}

			processed++
		}
	}

	took := time.Since(start)
	avg := float64(cfg.numKeys) / took.Seconds()
	fmt.Printf("Took %fs... Average %.2f keys per second\n", took.Seconds(), avg)
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
	fmt.Println("Loading funded wallets...")
	loadStart := time.Now()

	tsvInfo, err := os.Stat("funded.tsv")
	if err != nil {
		panic("couldn't stat funded.tsv")
	}
	srcMtime := tsvInfo.ModTime()

	if sets, err := readFundedCache(fundedCacheFile, srcMtime); err == nil {
		if !sets.bloomsComplete() {
			sets.ensureBlooms()
			if err := writeFundedCache(fundedCacheFile, sets, srcMtime); err != nil {
				fmt.Printf("Warning: could not upgrade cache: %v\n", err)
			} else {
				fmt.Println("Upgraded funded.cache to v2 (with bloom filters)")
			}
		}
		logFundedLoad(sets, loadStart, true)
		return sets
	}

	sets := parseFundedTSV()
	fmt.Println("Sorting funded wallets...")
	sortStart := time.Now()
	sets.Sort()
	fmt.Printf("Finished sorting funded wallets in %.2fs\n", time.Since(sortStart).Seconds())

	sets.ensureBlooms()

	if err := writeFundedCache(fundedCacheFile, sets, srcMtime); err != nil {
		fmt.Printf("Warning: could not write cache: %v\n", err)
	} else {
		fmt.Println("Wrote funded.cache")
	}

	logFundedLoad(sets, loadStart, false)
	return sets
}

func parseFundedTSV() FundedSets {
	file, err := os.Open("funded.tsv")
	if err != nil {
		panic("couldn't open funded.tsv")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
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

	for scanner.Scan() {
		addr, balance, ok := parseTSVLine(scanner.Bytes())
		if !ok || balance < 30000 {
			continue
		}
		sets.addAddress(string(addr))
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	return sets
}

func logFundedLoad(sets FundedSets, loadStart time.Time, fromCache bool) {
	printer := message.NewPrinter(language.English)
	source := "parsed"
	if fromCache {
		source = "cache"
	}
	printer.Printf(
		"Loaded %d wallets from %s in %.2fs (legacy=%d p2sh=%d segwit=%d taproot=%d other=%d)\n",
		sets.Total(),
		source,
		time.Since(loadStart).Seconds(),
		len(sets.Legacy),
		len(sets.P2SH),
		len(sets.SegwitV0),
		len(sets.TaprootV1),
		len(sets.Other),
	)
}