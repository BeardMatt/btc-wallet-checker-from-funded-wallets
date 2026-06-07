package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"btcfind/bitcoin"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("%s <number of wallets to try> <threads>\n", os.Args[0])
		return
	}

	numtests, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic("couldn't get number of wallets")
	}

	threads, err := strconv.Atoi(os.Args[2])
	if err != nil {
		panic("couldn't get threads")
	}
	workers := threads
	if workers < 1 {
		workers = 1
	}

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

	file, err := os.Open("funded.tsv")
	if err != nil {
		panic("couldn't open funded.tsv")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Skip header line
	if scanner.Scan() {
		// header consumed
	}

	funded := make([]string, 0, 1000000)

	for scanner.Scan() {
		line := scanner.Text()
		spl := strings.Split(line, "\t")

		if len(spl) < 2 {
			continue
		}

		balance, err := strconv.Atoi(spl[1])
		if err != nil {
			continue
		}

		if balance >= 30000 {
			funded = append(funded, spl[0])
		}
	}

	printer := message.NewPrinter(language.English)
	printer.Printf("Loaded %d wallets\n", len(funded))

	fmt.Println("Sorting funded wallets...")
	sort.Strings(funded)
	fmt.Println("Finished sorting funded wallets...")

	return funded
}

func inFunded(funded []string, address string) bool {
	idx := sort.SearchStrings(funded, address)
	return idx < len(funded) && funded[idx] == address
}