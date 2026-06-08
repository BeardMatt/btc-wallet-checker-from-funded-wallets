package main

import "testing"

func TestParseCLI(t *testing.T) {
	cfg, err := parseCLI([]string{"10000", "8", "--simulate-hit", "--simulate-hit-at", "500"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.numKeys != 10000 || cfg.threads != 8 || !cfg.simulateHit || cfg.simulateHitAt != 500 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestParseCLIFormats(t *testing.T) {
	cfg, err := parseCLI([]string{"1000", "--formats", "legacy,segwit"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.formats.LegacyCompressed || !cfg.formats.LegacyUncompressed || !cfg.formats.Segwit {
		t.Fatalf("unexpected formats: %+v", cfg.formats)
	}
	if cfg.formats.P2SH || cfg.formats.Taproot {
		t.Fatalf("expected p2sh and taproot disabled: %+v", cfg.formats)
	}
}

func TestParseCLIFormatsInjectConflict(t *testing.T) {
	_, err := parseCLI([]string{"1", "--formats=taproot", "--inject-index-hit=legacy"})
	if err == nil {
		t.Fatal("expected inject/format conflict error")
	}
}

func TestParseCLIForeverFlags(t *testing.T) {
	cfg, err := parseCLI([]string{"--forever", "8", "--reset-session", "--max-keys", "1000"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.forever || cfg.threads != 8 || !cfg.resetSession || cfg.maxKeys != 1000 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestParseCLIInject(t *testing.T) {
	cfg, err := parseCLI([]string{"1", "--inject-index-hit=legacy", "--simulate-hit-verify-lookup"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.injectIndexHit != "legacy" || !cfg.verifyLookup || cfg.simulateHit {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}