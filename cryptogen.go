package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"sort"
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
	funded := loadFunded()

	printer := message.NewPrinter(language.English)
	printer.Printf("Testing %d keys... ", numtests)

	start := time.Now()
	ch := newWallet(workers)

	for range numtests {
		wallet := <-ch
		for _, addr := range wallet.Addresses {
			if inFunded(funded, addr) {
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

func loadFunded() []string {
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

	funded := make([]string, 0, 32_000_000)

	for scanner.Scan() {
		addr, balance, ok := parseTSVLine(scanner.Bytes())
		if !ok || balance < 30000 {
			continue
		}
		funded = append(funded, string(addr))
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	printer := message.NewPrinter(language.English)
	printer.Printf("Loaded %d wallets in %.2fs\n", len(funded), time.Since(loadStart).Seconds())

	fmt.Println("Sorting funded wallets...")
	sortStart := time.Now()
	sort.Strings(funded)
	fmt.Printf("Finished sorting funded wallets in %.2fs\n", time.Since(sortStart).Seconds())

	return funded
}

func inFunded(funded []string, address string) bool {
	idx := sort.SearchStrings(funded, address)
	return idx < len(funded) && funded[idx] == address
}