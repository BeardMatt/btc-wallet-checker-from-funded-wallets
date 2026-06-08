package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"btcfind/bitcoin"

	"golang.org/x/term"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var appUI *UI

const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type UIConfig struct {
	NoColor bool
	Quiet   bool
	Verbose bool
}

type UI struct {
	color   bool
	quiet   bool
	verbose bool
	tty     bool
	out     io.Writer
	err     io.Writer
	printer *message.Printer

	progressActive     bool
	indeterminate      bool
	indeterminateLabel string
	spinnerFrame       int
	lastProgressUpdate time.Time
}

func NewUI(cfg UIConfig) *UI {
	tty := term.IsTerminal(int(os.Stdout.Fd()))
	return &UI{
		color:   tty && !cfg.NoColor,
		quiet:   cfg.Quiet,
		verbose: cfg.Verbose,
		tty:     tty,
		out:     os.Stdout,
		err:     os.Stderr,
		printer: message.NewPrinter(language.English),
	}
}

func ui() *UI {
	if appUI != nil {
		return appUI
	}
	return NewUI(UIConfig{NoColor: true})
}

func (u *UI) colorize(code, s string) string {
	if !u.color {
		return s
	}
	return code + s + ansiReset
}

func (u *UI) Banner() {
	if u.quiet {
		return
	}
	title := u.colorize(ansiBold, "btcfind — Bitcoin key search")
	u.PrintlnErr(title)
	u.PrintlnErr(strings.Repeat("─", 28))
}

func (u *UI) Section(title string) {
	if u.quiet {
		return
	}
	u.PrintfErr("%s %s\n", u.colorize(ansiCyan, "▸"), title)
}

func (u *UI) Infof(format string, args ...any) {
	if u.quiet {
		return
	}
	u.PrintfErr(format, args...)
}

func (u *UI) Warnf(format string, args ...any) {
	u.PrintfErr("Warning: "+format, args...)
}

func (u *UI) PrintfOut(format string, args ...any) {
	fmt.Fprintf(u.out, format, args...)
}

func (u *UI) PrintfErr(format string, args ...any) {
	fmt.Fprintf(u.err, format, args...)
}

func (u *UI) PrintlnErr(s string) {
	fmt.Fprintln(u.err, s)
}

func (u *UI) Printf(format string, args ...any) {
	u.PrintfErr(format, args...)
}

func (u *UI) formatInt(n int64) string {
	return u.printer.Sprintf("%d", n)
}

func (u *UI) formatIntN(n int) string {
	return u.printer.Sprintf("%d", n)
}

func (u *UI) Progress(label string, current, total int64) {
	if u.quiet {
		return
	}
	now := time.Now()
	if u.progressActive && now.Sub(u.lastProgressUpdate) < 100*time.Millisecond {
		return
	}
	u.lastProgressUpdate = now
	u.indeterminate = false
	u.progressActive = true

	var bar string
	if total > 0 {
		width := 24
		frac := float64(current) / float64(total)
		if frac > 1 {
			frac = 1
		}
		filled := int(frac * float64(width))
		bar = strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
		u.PrintfErr("\r%s %s %3.0f%% (%s / %s)   ", label, bar, frac*100, formatBytes(current), formatBytes(total))
	} else {
		u.PrintfErr("\r%s %s %s   ", label, u.spinner(), formatBytes(current))
	}
	u.spinnerFrame++
}

func (u *UI) ProgressIndeterminate(label string) {
	if u.quiet {
		return
	}
	now := time.Now()
	if u.progressActive && u.indeterminateLabel == label && now.Sub(u.lastProgressUpdate) < 100*time.Millisecond {
		return
	}
	u.lastProgressUpdate = now
	u.indeterminate = true
	u.indeterminateLabel = label
	u.progressActive = true
	u.PrintfErr("\r%s %s   ", u.spinner(), label)
	u.spinnerFrame++
}

func (u *UI) ClearProgress() {
	if !u.progressActive {
		return
	}
	u.progressActive = false
	u.indeterminate = false
	u.indeterminateLabel = ""
	if u.tty {
		u.PrintfErr("\r\033[K")
	} else {
		u.PrintfErr("\n")
	}
}

func (u *UI) spinner() string {
	return spinnerFrames[u.spinnerFrame%len(spinnerFrames)]
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func (u *UI) BeginForeverSearch() {
	if u.tty {
		return
	}
	u.PrintfOut("Running forever (Ctrl+C to stop)... ")
}

func (u *UI) ForeverProgress(processed uint64, elapsed time.Duration, hits uint64) {
	if !u.tty || u.quiet {
		return
	}
	now := time.Now()
	if u.progressActive && now.Sub(u.lastProgressUpdate) < 100*time.Millisecond {
		return
	}
	u.lastProgressUpdate = now
	u.progressActive = true
	u.indeterminate = false

	kps := float64(processed) / elapsed.Seconds()
	if elapsed < time.Millisecond {
		kps = 0
	}
	u.PrintfErr(
		"\rKeys tried: %s (%.1fk/s avg) | Hits: %s   ",
		u.formatInt(int64(processed)),
		kps/1000,
		u.formatInt(int64(hits)),
	)
}

func (u *UI) BeginSearch(total int) {
	if u.tty {
		return
	}
	u.PrintfOut("Testing %s keys... ", u.formatIntN(total))
}

func (u *UI) SearchProgress(processed, total int, elapsed time.Duration) {
	if !u.tty || u.quiet {
		return
	}
	now := time.Now()
	if u.progressActive && now.Sub(u.lastProgressUpdate) < 100*time.Millisecond {
		return
	}
	u.lastProgressUpdate = now
	u.progressActive = true
	u.indeterminate = false

	kps := float64(processed) / elapsed.Seconds()
	if elapsed < time.Millisecond {
		kps = 0
	}
	u.PrintfErr(
		"\rTesting %s keys… %s (%.1fk/s)   ",
		u.formatIntN(total),
		u.formatIntN(processed),
		kps/1000,
	)
}

func (u *UI) PrintSummary(took time.Duration, numKeys int, avg float64) {
	u.ClearProgress()
	u.PrintfOut("Took %fs... Average %.2f keys per second\n", took.Seconds(), avg)
}

func (u *UI) LogFundedLoad(sets FundedSets, elapsed time.Duration, fromCache bool) {
	if u.quiet {
		return
	}
	source := "parsed TSV"
	if fromCache {
		source = "cache hit"
	}
	minBal := sets.MinBalanceSats
	if minBal == 0 {
		minBal = defaultMinBalanceSats
	}
	u.Section(fmt.Sprintf("Funded data — %s (%.1fs, min %s sats)", source, elapsed.Seconds(), u.formatInt(int64(minBal))))
	u.PrintfErr(
		"  legacy %s  p2sh %s  segwit %s  taproot %s  other %s  (%s total)\n",
		u.formatIntN(len(sets.Legacy)),
		u.formatIntN(len(sets.P2SH)),
		u.formatIntN(len(sets.SegwitV0)),
		u.formatIntN(len(sets.TaprootV1)),
		u.formatIntN(len(sets.Other)),
		u.formatIntN(sets.Total()),
	)
}

func (u *UI) LogBloomBuild(sets FundedSets, elapsed time.Duration) {
	if u.quiet {
		return
	}
	_ = sets
	u.PrintfErr("  Bloom filters built in %.2fs\n", elapsed.Seconds())
}

func kindLabels(kind bitcoin.MatchKind) (id, label string) {
	switch kind {
	case bitcoin.MatchLegacyCompressed:
		return "legacy_compressed", "P2PKH compressed"
	case bitcoin.MatchLegacyUncompressed:
		return "legacy_uncompressed", "P2PKH uncompressed"
	case bitcoin.MatchSegwitV0:
		return "segwit_v0", "Native SegWit (bc1q)"
	case bitcoin.MatchP2SH:
		return "p2sh", "P2SH (3...)"
	case bitcoin.MatchTaproot:
		return "taproot", "Taproot (bc1p)"
	default:
		return "unknown", "Unknown"
	}
}

func formatBalanceSats(sats uint64) (btc string, satsFormatted string) {
	btcWhole := sats / 100_000_000
	btcFrac := sats % 100_000_000
	btc = fmt.Sprintf("%d.%08d BTC", btcWhole, btcFrac)
	satsFormatted = ui().formatInt(int64(sats))
	return btc, satsFormatted
}

func (u *UI) PrintHit(wallet bitcoin.Wallet, kind bitcoin.MatchKind, keyIndex int, balanceSats uint64, simulated bool) {
	addr := bitcoin.EncodeMatchAddress(kind, wallet.Keys)
	if addr == "" {
		panic("failed to encode hit address")
	}
	wif := formatWIF(wallet.PrivKey)
	id, label := kindLabels(kind)

	savedToLog := false
	if !simulated {
		if err := appendWalletHit(keyIndex, kind, addr, wif, balanceSats); err != nil {
			u.Warnf("could not write %s: %v\n", walletsLogFile, err)
		} else {
			savedToLog = true
		}
	}

	title := "WALLET FOUND"
	if simulated {
		if u.color {
			title = u.colorize(ansiYellow, "WALLET FOUND") + u.colorize(ansiDim, "  [SIMULATED]")
		} else {
			title = "WALLET FOUND  [SIMULATED]"
		}
	} else if u.color {
		title = u.colorize(ansiGreen+ansiBold, "WALLET FOUND")
	}

	const inner = 58
	border := func(s string) string {
		s = truncateString(s, inner)
		pad := inner - len(s)
		return "║  " + s + strings.Repeat(" ", pad) + "║"
	}

	u.PrintlnErr("")
	u.PrintlnErr("╔══════════════════════════════════════════════════════════════╗")
	u.PrintlnErr(border(title))
	u.PrintlnErr("╠══════════════════════════════════════════════════════════════╣")
	u.PrintlnErr(border("Address   "+addr))
	u.PrintlnErr(border(fmt.Sprintf("Format    %s (%s)", id, label)))
	u.PrintlnErr(border("WIF       "+wif))
	if balanceSats > 0 {
		btc, sats := formatBalanceSats(balanceSats)
		u.PrintlnErr(border(fmt.Sprintf("Balance   %s (%s sats)", btc, sats)))
	}
	u.PrintlnErr(border("Key #     "+u.formatIntN(keyIndex)))
	u.PrintlnErr("╚══════════════════════════════════════════════════════════════╝")
	if savedToLog {
		u.PrintfErr("  Saved to %s\n", walletsLogFile)
	}
	u.PrintlnErr("")
	u.PrintlnErr("Recover funds (do this on an offline or trusted machine):")
	u.PrintlnErr("")
	u.PrintlnErr("  1. Install a wallet: Electrum, Sparrow, or Bitcoin Core.")
	u.PrintlnErr("  2. Choose \"Import private key\" / \"Sweep private key\" (not \"watch-only\").")
	u.PrintlnErr("  3. Paste the WIF above. Use mainnet; do not change the key.")
	u.PrintlnErr("  4. Wait for sync, then send funds to a new address you control.")
	u.PrintlnErr("")
	u.PrintlnErr("  Electrum: Wallet → Private keys → Import")
	u.PrintlnErr("  Sparrow:  File → Import wallet → Import private key")
	u.PrintfErr("  Core:     bitcoin-cli importprivkey %q \"\" false\n", wif)
	u.PrintlnErr("")
	u.PrintlnErr("Security:")
	u.PrintlnErr("  • Anyone with this WIF controls the funds — store offline, never share.")
	u.PrintlnErr("  • Prefer sweeping to a new wallet; the discovered key was public in this output.")
	u.PrintlnErr("")
}

type byteCounter struct {
	r io.Reader
	n int64
}

func (b *byteCounter) Read(p []byte) (int, error) {
	n, err := b.r.Read(p)
	b.n += int64(n)
	return n, err
}