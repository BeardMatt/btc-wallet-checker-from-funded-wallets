package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"btcfind/funded"
	"btcfind/search"
)

func main() {
	cfg, err := parseCLI(os.Args[1:])
	if err != nil {
		ui().PrintfErr("%v\n", err)
		printUsage()
		return
	}

	appUI = NewUI(UIConfig{
		NoColor: cfg.noColor,
		Quiet:   cfg.quiet,
		Verbose: cfg.verbose,
	})

	workers := cfg.threads
	if workers < 1 {
		workers = runtime.NumCPU()
	}

	appUI.Banner()
	appUI.Section(fmt.Sprintf("Workers  %d", workers))
	if cfg.simulateHit {
		appUI.Infof("  Simulate hit at key %d\n", cfg.simulateHitAt)
	}
	if cfg.injectIndexHit != "" {
		appUI.Infof("  Inject index hit: bucket=%q key=%d\n", cfg.injectIndexHit, cfg.simulateHitAt)
	}
	if cfg.formatsSet {
		appUI.Infof("  Formats: %s\n", cfg.formats.String())
	}
	if cfg.minBalance != defaultMinBalanceSats {
		appUI.Infof("  Min balance: %s sats\n", appUI.formatInt(int64(cfg.minBalance)))
	}

	defer funded.UnmapActive()

	funded.EnsureTSV(mainReporter{})
	fundedSets := loadFunded(cfg.minBalance)

	appUI.Section("Search")
	searchCfg := search.Config{
		Threads:            workers,
		Formats:            cfg.formats,
		NumKeys:            cfg.numKeys,
		Forever:            cfg.forever,
		MaxKeys:            cfg.maxKeys,
		CheckpointInterval: cfg.checkpointInterval,
		ResetSession:       cfg.resetSession,
		SimulateHit:        cfg.simulateHit,
		SimulateHitAt:      cfg.simulateHitAt,
		VerifyLookup:       cfg.verifyLookup,
		InjectIndexHit:     cfg.injectIndexHit,
	}

	hooks := mainSearchHooks(cfg, fundedSets)

	ctx := context.Background()
	if cfg.forever {
		appUI.BeginForeverSearch()
		_, _ = search.Run(ctx, searchCfg, fundedSets, hooks)
		printForeverExit(cfg)
		return
	}

	appUI.BeginSearch(cfg.numKeys)
	_, _ = search.Run(ctx, searchCfg, fundedSets, hooks)
}

type mainReporter struct{}

func (mainReporter) Section(title string)                         { ui().Section(title) }
func (mainReporter) Infof(format string, args ...any)             { ui().Infof(format, args...) }
func (mainReporter) Warnf(format string, args ...any)             { ui().Warnf(format, args...) }
func (mainReporter) PrintfErr(format string, args ...any)         { ui().PrintfErr(format, args...) }
func (mainReporter) PrintlnErr(args ...any)                       { ui().PrintlnErr(fmt.Sprint(args...)) }
func (mainReporter) Progress(label string, cur, total int64)      { ui().Progress(label, cur, total) }
func (mainReporter) ProgressIndeterminate(label string)             { ui().ProgressIndeterminate(label) }
func (mainReporter) ClearProgress()                               { ui().ClearProgress() }
func (mainReporter) LogFundedLoad(sets funded.Sets, elapsed time.Duration, fromCache bool) {
	ui().LogFundedLoad(toMainSets(sets), elapsed, fromCache)
}
func (mainReporter) LogBloomBuild(sets funded.Sets, elapsed time.Duration) {
	ui().LogBloomBuild(toMainSets(sets), elapsed)
}
func (mainReporter) Quiet() bool { return ui().quiet }
func (mainReporter) Verbose() bool {
	return ui().verbose
}

func mainSearchHooks(cfg cliConfig, sets funded.Sets) search.Hooks {
	return search.Hooks{
		OnProgress: func(processed, total int, elapsed time.Duration) {
			appUI.SearchProgress(processed, total, elapsed)
		},
		OnForeverProgress: func(totalKeys uint64, elapsed time.Duration, hits uint64) {
			appUI.ForeverProgress(totalKeys, elapsed, hits)
		},
		OnSummary: func(took time.Duration, numKeys int, avg float64) {
			appUI.PrintSummary(took, numKeys, avg)
		},
		OnHit: func(ev search.HitEvent) error {
			ui().PrintHit(ev.Wallet, ev.Kind, ev.KeyIndex, ev.BalanceSats, ev.Simulated)
			return nil
		},
		OnInjectLog: func(bucket, addr string) {
			ui().Infof("  inject-index-hit: bucket=%s funded_address=%s\n", bucket, addr)
		},
		OnLookupVerify: func(ev search.LookupVerifyEvent) {
			u := ui()
			u.Printf(
				"lookup verify: legacy_compressed=%t legacy_uncompressed=%t segwit_v0=%t p2sh=%t taproot=%t real_match=%t",
				ev.LegacyCompressed, ev.LegacyUncompressed, ev.SegwitV0, ev.P2SH, ev.Taproot, ev.RealMatch,
			)
			if ev.RealMatch {
				u.Printf(" kind=%s", ev.KindName)
			}
			u.PrintlnErr("")
		},
	}
}

func loadFunded(minBalance uint64) funded.Sets {
	if minBalance == 0 {
		minBalance = defaultMinBalanceSats
	}
	sets, err := funded.Load(funded.LoadOpts{
		MinBalance: minBalance,
		Reporter:   mainReporter{},
	})
	if err != nil {
		panic(err)
	}
	return sets
}

func toMainSets(s funded.Sets) FundedSets { return s }

func printForeverExit(cfg cliConfig) {
	appUI.ClearProgress()
	// Session messages are written by search package; mirror prior UX when max-keys set.
	if cfg.maxKeys > 0 && !appUI.quiet {
		appUI.Infof("Forever run finished (--max-keys %s)\n", appUI.formatInt(int64(cfg.maxKeys)))
	}
}

