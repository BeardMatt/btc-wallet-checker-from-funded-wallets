package main

import (
	"os"
	"path/filepath"
	"testing"
)

func withSessionDir(t *testing.T, fn func()) {
	t.Helper()
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})
	fn()
}

func TestInitForeverSessionReset(t *testing.T) {
	withSessionDir(t, func() {
		path := filepath.Join(".", sessionFile)
		if err := os.WriteFile(path, []byte(`{"total_keys_tried":999}`), 0600); err != nil {
			t.Fatal(err)
		}

		session, err := initForeverSession(true)
		if err != nil {
			t.Fatal(err)
		}
		if session.TotalKeysTried != 0 {
			t.Fatalf("want fresh session, got %+v", session)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("expected session file removed on reset before first checkpoint")
		}
	})
}

func TestInitForeverSessionResume(t *testing.T) {
	withSessionDir(t, func() {
		content := `{
  "started_at": "2026-06-07T20:00:00Z",
  "last_checkpoint": "2026-06-07T21:00:00Z",
  "total_keys_tried": 1000000000,
  "total_hits": 0,
  "best_keys_per_second": 120000
}`
		if err := os.WriteFile(sessionFile, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}

		session, err := initForeverSession(false)
		if err != nil {
			t.Fatal(err)
		}
		if session.TotalKeysTried != 1_000_000_000 {
			t.Fatalf("want resumed total, got %d", session.TotalKeysTried)
		}
	})
}