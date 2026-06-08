package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"btcfind/funded"
)

func testUI(quiet, verbose bool) (*UI, *bytes.Buffer) {
	var stderr bytes.Buffer
	appUI = NewUI(UIConfig{NoColor: true, Quiet: quiet, Verbose: verbose})
	appUI.err = &stderr
	appUI.out = &bytes.Buffer{}
	return appUI, &stderr
}

func TestEnsureFundedUpToDateVerbose(t *testing.T) {
	if _, err := os.Stat(funded.DefaultTSVFile); err != nil {
		t.Skip("funded.tsv not present")
	}

	funded.SetRemoteLastModifiedHook(func() (time.Time, error) {
		return time.Now().Add(24 * time.Hour), nil
	})
	defer funded.SetRemoteLastModifiedHook(nil)

	u, stderr := testUI(false, true)
	info, err := os.Stat(funded.DefaultTSVFile)
	if err != nil {
		t.Fatal(err)
	}

	u.Section("Funded data check")
	funded.LogLocalFundedFile(mainReporter{}, info)
	u.Infof("  remote URL: %s\n", funded.DownloadURL)
	u.Infof("  remote      %s\n", time.Now().Add(24*time.Hour).UTC().Format(time.RFC3339))
	u.Infof("  HEAD request: %dms\n", 12)
	u.Infof("  Up to date — using local file\n")

	out := stderr.String()
	for _, want := range []string{
		"Funded data check",
		"funded.tsv",
		"modified",
		"remote URL:",
		"HEAD request:",
		"Up to date",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestEnsureFundedQuietOmitsCheckSection(t *testing.T) {
	funded.SetRemoteLastModifiedHook(func() (time.Time, error) {
		return time.Now().Add(24 * time.Hour), nil
	})
	defer funded.SetRemoteLastModifiedHook(nil)

	if _, err := os.Stat(funded.DefaultTSVFile); err != nil {
		t.Skip("funded.tsv not present")
	}

	_, stderr := testUI(true, false)
	ensureFunded()

	out := stderr.String()
	if strings.Contains(out, "Funded data check") {
		t.Fatalf("quiet mode should not show check section:\n%s", out)
	}
	if strings.Contains(out, "Up to date") {
		t.Fatalf("quiet mode should not show up-to-date line:\n%s", out)
	}
}

func TestEnsureFundedRemoteErrorShowsLocalFallback(t *testing.T) {
	if _, err := os.Stat(funded.DefaultTSVFile); err != nil {
		t.Skip("funded.tsv not present")
	}

	funded.SetRemoteLastModifiedHook(func() (time.Time, error) {
		return time.Time{}, os.ErrPermission
	})
	defer funded.SetRemoteLastModifiedHook(nil)

	_, stderr := testUI(false, false)
	ensureFunded()

	out := stderr.String()
	if !strings.Contains(out, "could not check for updates") {
		t.Fatalf("missing warning:\n%s", out)
	}
	if !strings.Contains(out, "Using local file") {
		t.Fatalf("missing local fallback:\n%s", out)
	}
	if !strings.Contains(out, "funded.tsv") {
		t.Fatalf("missing local file info:\n%s", out)
	}
}

func TestLogLocalFundedFileUsesPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "funded.tsv")
	if err := os.WriteFile(path, []byte("abc"), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	_, stderr := testUI(false, false)
	funded.LogLocalFundedFile(mainReporter{}, info)

	if !strings.Contains(stderr.String(), "3 B") {
		t.Fatalf("expected size in output: %s", stderr.String())
	}
}