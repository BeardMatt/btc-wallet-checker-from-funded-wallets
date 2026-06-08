package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"btcfind/worker"
)

func main() {
	var (
		coordURL   = flag.String("coordinator", "", "coordinator base URL (required)")
		authToken  = flag.String("auth-token", "", "bearer token (or BTCFIND_AUTH_TOKEN)")
		threads    = flag.Int("threads", 0, "local worker threads (0=NumCPU)")
		tlsSkip    = flag.Bool("tls-skip-verify", false, "skip TLS verify (dev/LAN only)")
		tlsCA      = flag.String("tls-ca", "", "custom CA PEM")
		tlsCert    = flag.String("tls-cert", "", "client TLS cert (mTLS)")
		tlsKey     = flag.String("tls-key", "", "client TLS key (mTLS)")
		proxyURL   = flag.String("proxy-url", "", "HTTP CONNECT proxy URL")
		dialTO     = flag.Duration("dial-timeout", 30*time.Second, "connection timeout")
		statsInt   = flag.Duration("stats-interval", 2*time.Second, "stats report interval")
		simHit     = flag.Bool("simulate-hit", false, "force simulated hit")
		simHitAt   = flag.Int("simulate-hit-at", 1, "key index for simulate hit")
		quiet      = flag.Bool("quiet", false, "minimal stderr output")
	)
	flag.Parse()

	if *coordURL == "" {
		fmt.Fprintln(os.Stderr, "error: --coordinator is required")
		os.Exit(1)
	}
	if *tlsSkip {
		fmt.Fprintln(os.Stderr, "Warning: TLS verification disabled (--tls-skip-verify)")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := worker.Config{
		CoordinatorURL: *coordURL,
		AuthToken:      *authToken,
		Threads:        *threads,
		TLSSkipVerify:  *tlsSkip,
		TLSCA:          *tlsCA,
		TLSCert:        *tlsCert,
		TLSKey:         *tlsKey,
		ProxyURL:       *proxyURL,
		DialTimeout:    *dialTO,
		StatsInterval:  *statsInt,
		SimulateHit:    *simHit,
		SimulateHitAt:  *simHitAt,
		Quiet:          *quiet,
	}

	if err := worker.Run(ctx, cfg); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "worker: %v\n", err)
		os.Exit(1)
	}
}