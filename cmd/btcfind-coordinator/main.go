package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"btcfind/bitcoin"
	"btcfind/coordinator"
	"btcfind/funded"
)

func main() {
	var (
		listen    = flag.String("listen", ":8443", "listen address")
		tlsCert   = flag.String("tls-cert", "", "TLS certificate file")
		tlsKey    = flag.String("tls-key", "", "TLS private key file")
		authToken = flag.String("auth-token", "", "bearer auth token (or BTCFIND_AUTH_TOKEN)")
		insecure  = flag.Bool("insecure", false, "allow HTTP without TLS (dev only)")
		formats   = flag.String("formats", "all", "address formats for cluster")
		minBal    = flag.Uint64("min-balance", funded.DefaultMinBalanceSats, "minimum balance sats")
		autoStart = flag.Bool("auto-start", false, "start without Enter (non-TTY)")
		quiet     = flag.Bool("quiet", false, "minimal output")
		noColor   = flag.Bool("no-color", false, "disable ANSI colors")
		mtlsCA    = flag.String("mtls-ca", "", "client CA PEM for mTLS")
		mtlsReq   = flag.Bool("mtls-require-client-cert", false, "require client certificate")
		maxWorkers = flag.Int("max-workers", 0, "max workers (0=unlimited)")
		allowCIDR  = flag.String("allow-cidr", "", "comma-separated CIDRs bypassing rate limits")
		rateReg    = flag.Int("rate-register", 0, "register requests per minute per IP")
		rateCache  = flag.Int("rate-cache", 0, "cache downloads per hour per IP")
		rateAPI    = flag.Int("rate-api", 0, "API requests per minute per IP")
	)
	flag.Parse()

	mask, err := bitcoin.ParseFormatMask(*formats)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var cidrs []string
	if *allowCIDR != "" {
		cidrs = strings.Split(*allowCIDR, ",")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := coordinator.Config{
		Listen:                *listen,
		TLSCert:               *tlsCert,
		TLSKey:                *tlsKey,
		AuthToken:             *authToken,
		Insecure:              *insecure,
		Formats:               mask,
		MinBalance:            *minBal,
		AutoStart:             *autoStart,
		Quiet:                 *quiet,
		NoColor:               *noColor,
		MTLSCA:                *mtlsCA,
		MTLSRequireClientCert: *mtlsReq,
		MaxWorkers:            *maxWorkers,
		AllowCIDR:             cidrs,
		RateRegister:          *rateReg,
		RateCache:             *rateCache,
		RateAPI:               *rateAPI,
	}

	if err := coordinator.Run(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "coordinator: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: btcfind-coordinator [flags]\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nMin balance default: %s\n", strconv.FormatUint(funded.DefaultMinBalanceSats, 10))
	}
}