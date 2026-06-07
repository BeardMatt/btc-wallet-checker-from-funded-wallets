package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"btcfind/bitcoin"
)

func TestPrintHitBanner(t *testing.T) {
	var stderr bytes.Buffer
	appUI = NewUI(UIConfig{NoColor: true})
	appUI.err = &stderr
	appUI.out = &bytes.Buffer{}

	wallet := bitcoin.GenKeypair()
	appUI.PrintHit(wallet, bitcoin.MatchLegacyCompressed, 42, true)

	out := stderr.String()
	for _, want := range []string{
		"WALLET FOUND",
		"SIMULATED",
		"Address",
		"Format",
		"WIF",
		"Key #",
		"Recover funds",
		"Electrum",
		"Sparrow",
		"bitcoin-cli importprivkey",
		"Security:",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in hit banner:\n%s", want, out)
		}
	}
}

func TestParseCLIFlagsUX(t *testing.T) {
	cfg, err := parseCLI([]string{"100", "--no-color", "--quiet", "--verbose"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.noColor || !cfg.quiet || !cfg.verbose {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestPrintSummaryFormat(t *testing.T) {
	var stdout bytes.Buffer
	appUI = NewUI(UIConfig{NoColor: true})
	appUI.out = &stdout
	appUI.err = &bytes.Buffer{}

	appUI.PrintSummary(921*time.Millisecond, 50000, 54281.47)

	got := stdout.String()
	if !strings.Contains(got, "Took 0.921000s... Average 54281.47 keys per second") {
		t.Fatalf("unexpected summary: %q", got)
	}
}