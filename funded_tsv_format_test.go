package main

import (
	"bufio"
	"os"
	"testing"

	"btcfind/bitcoin"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

func TestFundedTSVFormatAndAddressRoundTrip(t *testing.T) {
	f, err := os.Open("funded.tsv")
	if err != nil {
		t.Skip("funded.tsv not present")
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024), 10*1024*1024)
	if !sc.Scan() {
		t.Fatal("empty funded.tsv")
	}
	header := string(sc.Bytes())
	if header != "address\tbalance" {
		t.Fatalf("unexpected header %q, want address\\tbalance", header)
	}

	picked := map[string]int{}
	lines := 0
	badParse := 0

	for sc.Scan() {
		lines++
		addr, balance, ok := parseTSVLine(sc.Bytes())
		if !ok {
			badParse++
			continue
		}
		if balance <= 0 {
			t.Fatalf("non-positive balance on line %d: %q", lines, sc.Text())
		}

		addrStr := string(addr)
		decoded, err := btcutil.DecodeAddress(addrStr, &chaincfg.MainNetParams)
		if err != nil {
			continue
		}

		var bucket string
		switch decoded.(type) {
		case *btcutil.AddressPubKeyHash:
			bucket = "legacy"
		case *btcutil.AddressScriptHash:
			bucket = "p2sh"
		case *btcutil.AddressWitnessPubKeyHash:
			bucket = "p2wpkh"
		case *btcutil.AddressWitnessScriptHash:
			continue // P2WSH bc1q — not indexed for key search
		case *btcutil.AddressTaproot:
			bucket = "taproot"
		default:
			continue
		}

		if picked[bucket] >= 5 {
			continue
		}
		picked[bucket]++

		kind, hash, err := bitcoin.DecodeFundedAddressForLoad(addrStr)
		if err != nil {
			t.Fatalf("decode load failed for %q: %v", addrStr, err)
		}
		kind2, hash2, err := bitcoin.DecodeFundedAddress(addrStr)
		if err != nil {
			t.Fatalf("decode btcd failed for %q: %v", addrStr, err)
		}
		if kind != kind2 || string(hash) != string(hash2) {
			t.Fatalf("decode mismatch for %q: load=%d/%x btcd=%d/%x", addrStr, kind, hash, kind2, hash2)
		}

		var reencoded string
		switch kind {
		case bitcoin.KindLegacy:
			var h [20]byte
			copy(h[:], hash)
			reencoded = bitcoin.EncodePubKeyHash(h)
		case bitcoin.KindP2SH:
			var h [20]byte
			copy(h[:], hash)
			reencoded = bitcoin.EncodeScriptHash(h)
		case bitcoin.KindSegwitV0:
			var h [20]byte
			copy(h[:], hash)
			reencoded = bitcoin.EncodeWitnessPubKeyHash(h)
		case bitcoin.KindTaprootV1:
			var h [32]byte
			copy(h[:], hash)
			reencoded = bitcoin.EncodeTaprootKey(h)
		default:
			t.Fatalf("unknown kind %d for %q", kind, addrStr)
		}
		if reencoded != addrStr {
			t.Fatalf("round-trip mismatch\n  tsv: %s\n  got: %s", addrStr, reencoded)
		}

		if picked["legacy"] >= 5 && picked["p2sh"] >= 5 && picked["p2wpkh"] >= 5 && picked["taproot"] >= 5 {
			break
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if badParse > 0 {
		t.Fatalf("%d lines failed parseTSVLine in first pass", badParse)
	}
	for _, want := range []string{"legacy", "p2sh", "p2wpkh", "taproot"} {
		if picked[want] < 5 {
			t.Fatalf("insufficient %s samples: %+v", want, picked)
		}
	}
}

func TestFundedTSVLoadedCountsMatchIndexableTypes(t *testing.T) {
	if _, err := os.Stat("funded.tsv"); err != nil {
		t.Skip("funded.tsv not present")
	}

	const minBal = 30000
	counts := map[string]int{}
	other := 0

	f, err := os.Open("funded.tsv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024), 10*1024*1024)
	sc.Scan()

	for sc.Scan() {
		addr, balance, ok := parseTSVLine(sc.Bytes())
		if !ok || balance < minBal {
			continue
		}
		addrStr := string(addr)
		kind, _, err := bitcoin.DecodeFundedAddressForLoad(addrStr)
		if err != nil {
			other++
			continue
		}
		switch kind {
		case bitcoin.KindLegacy:
			counts["legacy"]++
		case bitcoin.KindP2SH:
			counts["p2sh"]++
		case bitcoin.KindSegwitV0:
			counts["segwit"]++
		case bitcoin.KindTaprootV1:
			counts["taproot"]++
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	// Spot-check against last known loaded totals (cache hit output).
	want := map[string]int{
		"legacy":  10636578,
		"p2sh":    4068571,
		"segwit":  15431958,
		"taproot": 1003097,
	}
	for k, w := range want {
		if counts[k] != w {
			t.Fatalf("%s count %d != expected %d (other=%d)", k, counts[k], w, other)
		}
	}
	// P2WSH and non-decodable rows land in other (~593k)
	if other < 500000 {
		t.Fatalf("expected large other bucket for P2WSH/non-standard, got %d", other)
	}
}

func TestHitAddressMatchesFundedTSVFormat(t *testing.T) {
	sets := FundedSets{
		Legacy:        [][20]byte{{1, 2, 3, 4, 5}},
		LegacyBalance: []uint64{50000},
	}
	keys := bitcoin.LookupKeys{}
	keys.CompressedHash = sets.Legacy[0]

	kind, ok := matchFunded(sets, keys, bitcoin.AllFormats())
	if !ok || kind != bitcoin.MatchLegacyCompressed {
		t.Fatalf("expected legacy compressed match, got %v %v", kind, ok)
	}

	addr := bitcoin.EncodeMatchAddress(kind, keys)
	if addr != bitcoin.EncodePubKeyHash(sets.Legacy[0]) {
		t.Fatalf("hit address encoding mismatch: %s vs %s", addr, bitcoin.EncodePubKeyHash(sets.Legacy[0]))
	}
}