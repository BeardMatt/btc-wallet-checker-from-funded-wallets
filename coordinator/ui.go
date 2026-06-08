package coordinator

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"btcfind/cluster"
	"btcfind/funded"

	"golang.org/x/term"
)

// UI provides coordinator terminal output.
type UI struct {
	color bool
	quiet bool
	tty   bool
	err   io.Writer
}

func NewUI(quiet, noColor bool) *UI {
	tty := term.IsTerminal(int(os.Stderr.Fd())) || term.IsTerminal(int(os.Stdout.Fd()))
	return &UI{
		color: tty && !noColor,
		quiet: quiet,
		tty:   tty,
		err:   os.Stderr,
	}
}

func (u *UI) Warnf(format string, args ...any) {
	fmt.Fprintf(u.err, "Warning: "+format, args...)
}

func (u *UI) Infof(format string, args ...any) {
	if u.quiet {
		return
	}
	fmt.Fprintf(u.err, format, args...)
}

func (u *UI) Section(title string) {
	if u.quiet {
		return
	}
	fmt.Fprintf(u.err, "▸ %s\n", title)
}

// Eventf prints important cluster events even in --quiet mode.
func (u *UI) Eventf(format string, args ...any) {
	fmt.Fprintf(u.err, format, args...)
}

func (u *UI) LogWorkerConnected(hostname string, threads, totalWorkers, totalThreads int) {
	u.Eventf("▸ Worker connected: %s (%d threads) — %d worker(s), %d threads total\n",
		hostname, threads, totalWorkers, totalThreads)
}

func (u *UI) LogListening(addr string) {
	u.Eventf("▸ Coordinator listening on %s — waiting for workers\n", addr)
}

// UIReporter adapts UI to funded.Reporter.
type UIReporter struct {
	*UI
	verbose bool
}

func (r UIReporter) PrintfErr(format string, args ...any) { r.UI.Infof(format, args...) }
func (r UIReporter) PrintlnErr(args ...any)               { fmt.Fprintln(r.UI.err, args...) }
func (r UIReporter) Progress(string, int64, int64)        {}
func (r UIReporter) ProgressIndeterminate(string)         {}
func (r UIReporter) ClearProgress()                       {}
func (r UIReporter) LogFundedLoad(sets funded.Sets, elapsed time.Duration, fromCache bool) {
	if r.UI.quiet {
		return
	}
	source := "parsed TSV"
	if fromCache {
		source = "cache hit"
	}
	r.UI.Infof("  Funded loaded (%s, %.1fs, %d entries)\n", source, elapsed.Seconds(), sets.Total())
}
func (r UIReporter) LogBloomBuild(funded.Sets, time.Duration) {}
func (r UIReporter) Quiet() bool                              { return r.UI.quiet }
func (r UIReporter) Verbose() bool                            { return r.verbose }

func (u *UI) PrintClusterHit(address, wif, id, label string, balanceSats uint64, keyIndex int) {
	u.Infof("\n╔══════════════════════════════════════════════════════════════╗\n")
	u.Infof("║  WALLET FOUND (cluster)                                      ║\n")
	u.Infof("╠══════════════════════════════════════════════════════════════╣\n")
	u.Infof("║  Address   %-49s ║\n", truncate(address, 49))
	u.Infof("║  Format    %-49s ║\n", truncate(fmt.Sprintf("%s (%s)", id, label), 49))
	u.Infof("║  WIF       %-49s ║\n", truncate(wif, 49))
	if balanceSats > 0 {
		u.Infof("║  Balance   %-49s ║\n", truncate(fmt.Sprintf("%d sats", balanceSats), 49))
	}
	u.Infof("║  Key #     %-49d ║\n", keyIndex)
	u.Infof("╚══════════════════════════════════════════════════════════════╝\n")
	u.Infof("  Saved to %s\n\n", walletsLogFile)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func (u *UI) renderDashboard(state string, workers []workerEntry, total uint64, kps float64, hits uint64) {
	if u.quiet {
		u.renderDashboardPlain(state, workers, total, kps, hits)
		return
	}
	if !u.tty {
		u.renderDashboardPlain(state, workers, total, kps, hits)
		return
	}
	var b strings.Builder
	b.WriteString("\033[2J\033[H")
	b.WriteString("btcfind cluster coordinator\n")
	b.WriteString(strings.Repeat("─", 40) + "\n")
	u.writeDashboardBody(&b, state, workers, total, kps, hits)
	fmt.Fprint(u.err, b.String())
}

func (u *UI) renderDashboardPlain(state string, workers []workerEntry, total uint64, kps float64, hits uint64) {
	var b strings.Builder
	u.writeDashboardBody(&b, state, workers, total, kps, hits)
	fmt.Fprint(u.err, b.String())
}

func (u *UI) writeDashboardBody(b *strings.Builder, state string, workers []workerEntry, total uint64, kps float64, hits uint64) {
	threads := 0
	for _, w := range workers {
		threads += w.Threads
	}
	b.WriteString(fmt.Sprintf("State: %s | %.0f keys/s | %d keys | Hits: %d | Workers: %d (%d threads)\n\n",
		state, kps, total, hits, len(workers), threads))
	if len(workers) == 0 {
		b.WriteString("  (no workers connected yet)\n")
	}
	for _, w := range workers {
		b.WriteString(fmt.Sprintf("  %-16s %3dt  %-8s  %.0f keys/s  %d run keys\n",
			w.Hostname, w.Threads, w.State, w.KeysPerSec, w.RunKeys))
	}
	if state == cluster.StateLobby {
		b.WriteString("\nPress Enter to start search… (Ctrl+C to stop)\n")
	} else {
		b.WriteString("\nCtrl+C to stop\n")
	}
}