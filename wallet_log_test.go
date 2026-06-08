package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"btcfind/bitcoin"
)

func TestAppendWalletHit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wallets.txt")

	wallet := bitcoin.GenKeypair()
	addr := bitcoin.EncodeMatchAddress(bitcoin.MatchLegacyCompressed, wallet.Keys)
	wif := formatWIF(wallet.PrivKey)

	if err := appendWalletHitAt(path, 1, bitcoin.MatchLegacyCompressed, addr, wif, 123456789); err != nil {
		t.Fatal(err)
	}
	if err := appendWalletHitAt(path, 42, bitcoin.MatchLegacyCompressed, addr, wif, 50000); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("want mode 0600, got %o", info.Mode().Perm())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Count(content, "--- wallet found ") != 2 {
		t.Fatalf("want 2 entries, got:\n%s", content)
	}
	for _, want := range []string{
		"key_index: 1",
		"key_index: 42",
		"format: legacy_compressed (P2PKH compressed)",
		"address: " + addr,
		"balance_sats: 123456789",
		"wif: " + wif,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("missing %q in:\n%s", want, content)
		}
	}
}

func TestPrintHitSimulatedNoWalletLog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wallets.txt")

	prev := walletsLogPath
	walletsLogPath = path
	defer func() { walletsLogPath = prev }()

	appUI = NewUI(UIConfig{NoColor: true})
	appUI.err = &bytes.Buffer{}
	appUI.out = &bytes.Buffer{}

	wallet := bitcoin.GenKeypair()
	appUI.PrintHit(wallet, bitcoin.MatchLegacyCompressed, 1, 0, true)

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("simulated hit must not create wallets.txt")
	}
}

func TestPrintHitRealWritesWalletLog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wallets.txt")

	prev := walletsLogPath
	walletsLogPath = path
	defer func() { walletsLogPath = prev }()

	appUI = NewUI(UIConfig{NoColor: true})
	appUI.err = &bytes.Buffer{}
	appUI.out = &bytes.Buffer{}

	wallet := bitcoin.GenKeypair()
	appUI.PrintHit(wallet, bitcoin.MatchSegwitV0, 7, 100000, false)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "key_index: 7") {
		t.Fatalf("missing log entry:\n%s", data)
	}
}