package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"btcfind/bitcoin"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("%s <number of wallets to try> [threads]\n", os.Args[0])
		return
	}

	numtests, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic("couldn't get number of wallets")
	}

	threads := 0
	if len(os.Args) >= 3 {
		threads, err = strconv.Atoi(os.Args[2])
		if err != nil {
			panic("couldn't get threads")
		}
	}

	workers := threads
	if workers < 1 {
		workers = runtime.NumCPU()
	}
	fmt.Printf("Using %d workers\n", workers)

	ensureFunded()
	fundedSets := loadFunded()

	printer := message.NewPrinter(language.English)
	printer.Printf("Testing %d keys... ", numtests)

	start := time.Now()
	ch := newWallet(workers)

	for range numtests {
		wallet := <-ch
		for _, addr := range wallet.Addresses {
			if inFunded(fundedSets, addr) {
				privKey, _ := btcec.PrivKeyFromBytes(wallet.PrivKey)
				wif, err := btcutil.NewWIF(privKey, &chaincfg.MainNetParams, true)
				if err != nil {
					panic(err)
				}
				fmt.Println(wif.String(), " : ", addr)
				break
			}
		}
	}

	took := time.Since(start)
	avg := float64(numtests) / took.Seconds()
	fmt.Printf("Took %fs... Average %.2f keys per second\n", took.Seconds(), avg)
}

func newWallet(n int) chan bitcoin.Wallet {
	ch := make(chan bitcoin.Wallet, n)
	for range n {
		go func() {
			for {
				ch <- bitcoin.GenKeypair()
			}
		}()
	}
	return ch
}

func loadFunded() FundedSets {
	fmt.Println("Loading funded wallets...")
	loadStart := time.Now()

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

	printer := message.NewPrinter(language.English)
	printer.Printf(
		"Loaded %d wallets in %.2fs (legacy=%d p2sh=%d segwit=%d taproot=%d other=%d)\n",
		sets.Total(),
		time.Since(loadStart).Seconds(),
		len(sets.Legacy),
		len(sets.P2SH),
		len(sets.SegwitV0),
		len(sets.TaprootV1),
		len(sets.Other),
	)

	fmt.Println("Sorting funded wallets...")
	sortStart := time.Now()
	sets.Sort()
	fmt.Printf("Finished sorting funded wallets in %.2fs\n", time.Since(sortStart).Seconds())

	return sets
}