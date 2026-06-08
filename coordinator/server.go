package coordinator

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"btcfind/cluster"
	"btcfind/funded"
)

// Run starts the coordinator: load funded data, serve API, wait for Enter, run dashboard.
func Run(ctx context.Context, cfg Config) error {
	if cfg.MinBalance == 0 {
		cfg.MinBalance = funded.DefaultMinBalanceSats
	}
	if cfg.AuthToken == "" {
		cfg.AuthToken = os.Getenv("BTCFIND_AUTH_TOKEN")
	}
	if !cfg.Insecure && (cfg.TLSCert == "" || cfg.TLSKey == "") {
		return fmt.Errorf("TLS cert and key required (or pass --insecure for dev HTTP)")
	}
	if cfg.AuthToken == "" {
		return fmt.Errorf("auth token required (--auth-token or BTCFIND_AUTH_TOKEN)")
	}

	ui := NewUI(cfg.Quiet, cfg.NoColor)
	srv := New(cfg, ui)

	ui.Section("Funded data")
	funded.EnsureTSV(UIReporter{UI: ui})
	sets, err := funded.Load(funded.LoadOpts{
		MinBalance: cfg.MinBalance,
		Reporter:   UIReporter{UI: ui},
	})
	if err != nil {
		return err
	}
	srv.fundedSets = sets
	srv.cachePath = funded.CachePath(cfg.MinBalance)
	etag, err := funded.CacheETag(srv.cachePath)
	if err != nil {
		return err
	}
	srv.setRunConfig(etag)

	mux := http.NewServeMux()
	srv.mountAPI(mux)

	httpSrv := &http.Server{Addr: cfg.Listen, Handler: mux}
	tlsCfg, err := tlsConfig(cfg)
	if err != nil {
		return err
	}
	httpSrv.TLSConfig = tlsCfg

	go func() {
		var err error
		if cfg.Insecure {
			ui.Warnf("running without TLS (--insecure)\n")
			err = httpSrv.ListenAndServe()
		} else {
			err = httpSrv.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
		}
		if err != nil && err != http.ErrServerClosed {
			ui.Warnf("server error: %v\n", err)
		}
	}()

	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				srv.pruneStale()
			}
		}
	}()

	go srv.dashboardLoop(ctx)

	if !cfg.AutoStart && termIsTTY() {
		waitEnterOrWorkers(srv, ui)
	} else {
		srv.startRun()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
	case <-sigCh:
	}

	srv.stopRun()
	ui.Infof("\nStopping cluster…\n")
	time.Sleep(2 * time.Second)
	srv.endRun()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)

	state, total, kps, hits, _ := srv.clusterSnapshot()
	ui.Infof("Run ended — %s keys tried, %.0f keys/s, %d hits\n", formatUint(total), kps, hits)
	_ = state
	funded.UnmapActive()
	return nil
}

func waitEnterOrWorkers(srv *Server, ui *UI) {
	ui.Infof("Workers may connect. Press Enter to start search…\n")
	enterCh := make(chan struct{})
	go func() {
		reader := bufio.NewReader(os.Stdin)
		_, _ = reader.ReadString('\n')
		close(enterCh)
	}()
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		workers := srv.workerSnapshot()
		if len(workers) > 0 {
			threads := 0
			for _, w := range workers {
				threads += w.Threads
			}
			ui.Infof("Workers connected: %d (%d threads total)\n", len(workers), threads)
			for _, w := range workers {
				ui.Infof("  %s  %dt\n", w.Hostname, w.Threads)
			}
		}
		select {
		case <-enterCh:
			srv.startRun()
			ui.Infof("Search started.\n")
			return
		case <-t.C:
		}
	}
}

func (s *Server) dashboardLoop(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			state, total, kps, hits, _ := s.clusterSnapshot()
			if state == cluster.StateRunning || state == cluster.StateLobby {
				s.ui.renderDashboard(state, s.workerSnapshot(), total, kps, hits)
			}
		}
	}
}

func termIsTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func formatUint(n uint64) string {
	return fmt.Sprintf("%d", n)
}