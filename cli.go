package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type cliConfig struct {
	numKeys          int
	threads          int
	simulateHit      bool
	simulateHitAt    int
	verifyLookup     bool
	injectIndexHit   string
}

func parseCLI(args []string) (cliConfig, error) {
	cfg := cliConfig{
		simulateHitAt: 1,
	}

	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--simulate-hit":
			cfg.simulateHit = true
		case arg == "--simulate-hit-verify-lookup":
			cfg.verifyLookup = true
		case strings.HasPrefix(arg, "--simulate-hit-at="):
			v, err := strconv.Atoi(strings.TrimPrefix(arg, "--simulate-hit-at="))
			if err != nil || v < 1 {
				return cfg, fmt.Errorf("invalid --simulate-hit-at value")
			}
			cfg.simulateHitAt = v
		case arg == "--simulate-hit-at":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--simulate-hit-at requires a value")
			}
			i++
			v, err := strconv.Atoi(args[i])
			if err != nil || v < 1 {
				return cfg, fmt.Errorf("invalid --simulate-hit-at value")
			}
			cfg.simulateHitAt = v
		case strings.HasPrefix(arg, "--inject-index-hit="):
			cfg.injectIndexHit = strings.TrimPrefix(arg, "--inject-index-hit=")
		case arg == "--inject-index-hit":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--inject-index-hit requires a value")
			}
			i++
			cfg.injectIndexHit = args[i]
		case strings.HasPrefix(arg, "-"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			positionals = append(positionals, arg)
		}
	}

	if len(positionals) < 1 {
		return cfg, fmt.Errorf("missing <number of wallets to try>")
	}

	numKeys, err := strconv.Atoi(positionals[0])
	if err != nil {
		return cfg, fmt.Errorf("invalid number of wallets")
	}
	cfg.numKeys = numKeys

	if len(positionals) >= 2 {
		threads, err := strconv.Atoi(positionals[1])
		if err != nil {
			return cfg, fmt.Errorf("invalid threads")
		}
		cfg.threads = threads
	}

	if cfg.injectIndexHit != "" {
		if err := validateInjectBucket(cfg.injectIndexHit); err != nil {
			return cfg, err
		}
	}

	if cfg.verifyLookup && cfg.simulateHitAt > cfg.numKeys {
		return cfg, fmt.Errorf("--simulate-hit-at %d exceeds num_keys %d", cfg.simulateHitAt, cfg.numKeys)
	}
	if cfg.simulateHit && cfg.simulateHitAt > cfg.numKeys {
		return cfg, fmt.Errorf("--simulate-hit-at %d exceeds num_keys %d", cfg.simulateHitAt, cfg.numKeys)
	}

	return cfg, nil
}

func validateInjectBucket(bucket string) error {
	switch bucket {
	case "legacy", "legacy-uncompressed", "p2sh", "segwit", "taproot":
		return nil
	default:
		return fmt.Errorf("unknown inject bucket %q (want legacy, legacy-uncompressed, p2sh, segwit, taproot)", bucket)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <number of wallets to try> [threads] [flags]\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --simulate-hit                 Force hit output on key N (tests WIF/address path)")
	fmt.Fprintln(os.Stderr, "  --simulate-hit-at N            Key index for simulate/inject (default 1)")
	fmt.Fprintln(os.Stderr, "  --simulate-hit-verify-lookup   Log whether wallet matches funded index")
	fmt.Fprintln(os.Stderr, "  --inject-index-hit BUCKET        Replace wallet hash with index[0] from bucket")
	fmt.Fprintln(os.Stderr, "                                 (legacy, legacy-uncompressed, p2sh, segwit, taproot)")
}