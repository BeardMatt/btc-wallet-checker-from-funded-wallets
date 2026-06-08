package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testUI(quiet, verbose bool) (*UI, *bytes.Buffer) {
	var stderr bytes.Buffer
	appUI = NewUI(UIConfig{NoColor: true, Quiet: quiet, Verbose: verbose})
	appUI.err = &stderr
	appUI.out = &bytes.Buffer{}
	return appUI, &stderr
}

func TestEnsureFundedUpToDateVerbose(t *testing.T) {
	if _, err := os.Stat(fundedFile); err != nil {
		t.Skip("funded.tsv not present")
	}

	orig := fetchRemoteLastModified
	defer func() { fetchRemoteLastModified = orig }()

	fetchRemoteLastModified = func() (time.Time, error) {
		return time.Now().Add(24 * time.Hour), nil
	}

	u, stderr := testUI(false, true)
	info, err := os.Stat(fundedFile)
	if err != nil {
		t.Fatal(err)
	}

	u.Section("Funded data check")
	logLocalFundedFile(u, info)
	u.Infof("  remote URL: %s\n", fundedDownloadURL)
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
	orig := fetchRemoteLastModified
	defer func() { fetchRemoteLastModified = orig }()

	fetchRemoteLastModified = func() (time.Time, error) {
		return time.Now().Add(24 * time.Hour), nil
	}

	if _, err := os.Stat(fundedFile); err != nil {
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
	if _, err := os.Stat(fundedFile); err != nil {
		t.Skip("funded.tsv not present")
	}

	orig := fetchRemoteLastModified
	defer func() { fetchRemoteLastModified = orig }()

	fetchRemoteLastModified = func() (time.Time, error) {
		return time.Time{}, os.ErrPermission
	}

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

	u, stderr := testUI(false, false)
	logLocalFundedFile(u, info)

	if !strings.Contains(stderr.String(), "3 B") {
		t.Fatalf("expected size in output: %s", stderr.String())
	}
}