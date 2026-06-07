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

func TestParseCLIInject(t *testing.T) {
	cfg, err := parseCLI([]string{"1", "--inject-index-hit=legacy", "--simulate-hit-verify-lookup"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.injectIndexHit != "legacy" || !cfg.verifyLookup || cfg.simulateHit {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}