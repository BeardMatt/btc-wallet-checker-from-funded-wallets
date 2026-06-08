package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"btcfind/bitcoin"
)

type cliConfig struct {
	numKeys              int
	threads              int
	minBalance           uint64
	forever              bool
	checkpointInterval   time.Duration
	maxKeys              uint64
	simulateHit    bool
	simulateHitAt  int
	verifyLookup   bool
	injectIndexHit string
	noColor        bool
	quiet          bool
	verbose        bool
	formats        bitcoin.FormatMask
	formatsSet     bool
}

func parseCLI(args []string) (cliConfig, error) {
	cfg := cliConfig{
		simulateHitAt:      1,
		formats:            bitcoin.AllFormats(),
		minBalance:         defaultMinBalanceSats,
		checkpointInterval: 60 * time.Second,
	}

	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--forever":
			cfg.forever = true
		case strings.HasPrefix(arg, "--checkpoint-interval="):
			d, err := time.ParseDuration(strings.TrimPrefix(arg, "--checkpoint-interval="))
			if err != nil || d <= 0 {
				return cfg, fmt.Errorf("invalid --checkpoint-interval value")
			}
			cfg.checkpointInterval = d
		case arg == "--checkpoint-interval":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--checkpoint-interval requires a value")
			}
			i++
			d, err := time.ParseDuration(args[i])
			if err != nil || d <= 0 {
				return cfg, fmt.Errorf("invalid --checkpoint-interval value")
			}
			cfg.checkpointInterval = d
		case strings.HasPrefix(arg, "--max-keys="):
			v, err := strconv.ParseUint(strings.TrimPrefix(arg, "--max-keys="), 10, 64)
			if err != nil {
				return cfg, fmt.Errorf("invalid --max-keys value")
			}
			cfg.maxKeys = v
		case arg == "--max-keys":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--max-keys requires a value")
			}
			i++
			v, err := strconv.ParseUint(args[i], 10, 64)
			if err != nil {
				return cfg, fmt.Errorf("invalid --max-keys value")
			}
			cfg.maxKeys = v
		case arg == "--simulate-hit":
			cfg.simulateHit = true
		case arg == "--no-color":
			cfg.noColor = true
		case arg == "--quiet":
			cfg.quiet = true
		case arg == "--verbose":
			cfg.verbose = true
		case strings.HasPrefix(arg, "--formats="):
			mask, err := bitcoin.ParseFormatMask(strings.TrimPrefix(arg, "--formats="))
			if err != nil {
				return cfg, err
			}
			cfg.formats = mask
			cfg.formatsSet = true
		case arg == "--formats":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--formats requires a value")
			}
			i++
			mask, err := bitcoin.ParseFormatMask(args[i])
			if err != nil {
				return cfg, err
			}
			cfg.formats = mask
			cfg.formatsSet = true
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
		case strings.HasPrefix(arg, "--min-balance="):
			v, err := strconv.ParseUint(strings.TrimPrefix(arg, "--min-balance="), 10, 64)
			if err != nil {
				return cfg, fmt.Errorf("invalid --min-balance value")
			}
			cfg.minBalance = v
		case arg == "--min-balance":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--min-balance requires a value")
			}
			i++
			v, err := strconv.ParseUint(args[i], 10, 64)
			if err != nil {
				return cfg, fmt.Errorf("invalid --min-balance value")
			}
			cfg.minBalance = v
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

	if cfg.forever {
		if len(positionals) >= 1 {
			threads, err := strconv.Atoi(positionals[0])
			if err != nil {
				return cfg, fmt.Errorf("invalid threads")
			}
			cfg.threads = threads
		}
		if len(positionals) >= 2 {
			return cfg, fmt.Errorf("too many positional arguments with --forever")
		}
	} else {
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
	}

	if cfg.injectIndexHit != "" {
		if err := validateInjectBucket(cfg.injectIndexHit); err != nil {
			return cfg, err
		}
		if !bitcoin.InjectBucketAllowed(cfg.formats, cfg.injectIndexHit) {
			return cfg, fmt.Errorf("inject bucket %q is disabled by --formats %s", cfg.injectIndexHit, cfg.formats.String())
		}
	}

	if !cfg.forever {
		if cfg.verifyLookup && cfg.simulateHitAt > cfg.numKeys {
			return cfg, fmt.Errorf("--simulate-hit-at %d exceeds num_keys %d", cfg.simulateHitAt, cfg.numKeys)
		}
		if cfg.simulateHit && cfg.simulateHitAt > cfg.numKeys {
			return cfg, fmt.Errorf("--simulate-hit-at %d exceeds num_keys %d", cfg.simulateHitAt, cfg.numKeys)
		}
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
	fmt.Fprintf(os.Stderr, "       %s --forever [threads] [flags]\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --simulate-hit                 Force hit output on key N (tests WIF/address path)")
	fmt.Fprintln(os.Stderr, "  --simulate-hit-at N            Key index for simulate/inject (default 1)")
	fmt.Fprintln(os.Stderr, "  --simulate-hit-verify-lookup   Log whether wallet matches funded index")
	fmt.Fprintln(os.Stderr, "  --inject-index-hit BUCKET        Replace wallet hash with index[0] from bucket")
	fmt.Fprintln(os.Stderr, "                                 (legacy, legacy-uncompressed, p2sh, segwit, taproot)")
	fmt.Fprintln(os.Stderr, "  --no-color                       Disable ANSI colors and live progress")
	fmt.Fprintln(os.Stderr, "  --quiet                          Minimal startup output (hits and warnings only)")
	fmt.Fprintln(os.Stderr, "  --verbose                        Verbose bloom filter details at load time")
	fmt.Fprintln(os.Stderr, "  --forever                        Run until interrupted; checkpoints to btcfind.session")
	fmt.Fprintln(os.Stderr, "  --checkpoint-interval DURATION   Session checkpoint interval (default 60s)")
	fmt.Fprintln(os.Stderr, "  --max-keys N                     Safety cap on total keys in forever mode")
	fmt.Fprintln(os.Stderr, "  --min-balance SATS               Minimum funded balance to index (default 30000)")
	fmt.Fprintln(os.Stderr, "  --formats LIST                   Address types to derive/check (comma-separated)")
	fmt.Fprintln(os.Stderr, "                                 legacy, legacy-compressed, legacy-uncompressed,")
	fmt.Fprintln(os.Stderr, "                                 segwit, p2sh, taproot, or all (default: all)")
}