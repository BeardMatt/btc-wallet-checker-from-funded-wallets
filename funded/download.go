package funded

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// DownloadURL is the remote source for funded.tsv.gz.
	DownloadURL = "http://addresses.loyce.club/blockchair_bitcoin_addresses_and_balance_LATEST.tsv.gz"
)

var fetchRemoteLastModified = remoteLastModified

// SetRemoteLastModifiedHook replaces the remote HEAD check (for tests).
func SetRemoteLastModifiedHook(fn func() (time.Time, error)) {
	if fn == nil {
		fetchRemoteLastModified = remoteLastModified
		return
	}
	fetchRemoteLastModified = fn
}

// LogLocalFundedFile logs local funded.tsv metadata (for tests and UI).
func LogLocalFundedFile(rep Reporter, info os.FileInfo) {
	logLocalFundedFile(rep, info)
}

// EnsureTSV ensures funded.tsv exists and is up to date, downloading when needed.
func EnsureTSV(rep Reporter) error {
	if rep == nil {
		rep = NopReporter{}
	}

	info, err := os.Stat(DefaultTSVFile)
	if os.IsNotExist(err) {
		if rep.Quiet() {
			rep.PrintfErr("funded.tsv not found, downloading...\n")
		} else {
			rep.Section("Downloading funded.tsv")
		}
		return downloadTSV(rep)
	}
	if err != nil {
		return err
	}

	if !rep.Quiet() {
		rep.Section("Funded data check")
		logLocalFundedFile(rep, info)
		if rep.Verbose() {
			if abs, err := filepath.Abs(DefaultTSVFile); err == nil {
				rep.Infof("  path: %s\n", abs)
			}
			rep.Infof("  remote URL: %s\n", DownloadURL)
		}
	}

	rep.ProgressIndeterminate("Checking for updates…")
	headStart := time.Now()
	remoteMod, err := fetchRemoteLastModified()
	headElapsed := time.Since(headStart)
	rep.ClearProgress()

	if err != nil {
		rep.Warnf("could not check for updates (%v), using local file\n", err)
		if !rep.Quiet() {
			rep.Infof("  Using local file\n")
		}
		return nil
	}

	if !rep.Quiet() {
		if rep.Verbose() {
			rep.Infof("  HEAD request: %dms\n", headElapsed.Milliseconds())
		}
		rep.Infof("  remote      %s\n", remoteMod.Format(time.RFC3339))
	}

	if remoteMod.After(info.ModTime()) {
		rep.PrintfErr(
			"funded.tsv is out of date (local: %s, remote: %s)\n",
			info.ModTime().Format(time.RFC3339),
			remoteMod.Format(time.RFC3339),
		)
		if rep.Verbose() {
			rep.Infof("  note: download will remove %s\n", DefaultCacheFile)
		}
		rep.PrintfErr("Download update? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "y" || answer == "yes" {
			return downloadTSV(rep)
		}

		rep.PrintlnErr("Using existing funded.tsv")
		return nil
	}

	if !rep.Quiet() {
		rep.Infof("  Up to date — using local file\n")
	}
	return nil
}

func logLocalFundedFile(rep Reporter, info os.FileInfo) {
	rep.Infof(
		"  %s  %s  modified %s\n",
		DefaultTSVFile,
		formatBytes(info.Size()),
		info.ModTime().UTC().Format(time.RFC3339),
	)
}

func remoteLastModified() (time.Time, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodHead, DownloadURL, nil)
	if err != nil {
		return time.Time{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("HEAD request returned %s", resp.Status)
	}

	lastModified := resp.Header.Get("Last-Modified")
	if lastModified == "" {
		return time.Time{}, fmt.Errorf("remote file has no Last-Modified header")
	}

	return time.Parse(http.TimeFormat, lastModified)
}

func downloadTSV(rep Reporter) error {
	rep.Infof("  Source: %s\n", DownloadURL)
	if rep.Verbose() {
		rep.Infof("  note: will remove %s after download\n", DefaultCacheFile)
	}

	client := &http.Client{}
	resp, err := client.Get(DownloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %s", resp.Status)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip decompress failed: %w", err)
	}
	defer gz.Close()

	tmpFile := DefaultTSVFile + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}

	written, err := copyWithProgress(rep, out, gz)
	rep.ClearProgress()
	closeErr := out.Close()
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("write failed: %w", err)
	}
	if closeErr != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("close failed: %w", closeErr)
	}

	if err := os.Rename(tmpFile, DefaultTSVFile); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("rename failed: %w", err)
	}

	if remoteMod, err := remoteLastModified(); err == nil {
		_ = os.Chtimes(DefaultTSVFile, remoteMod, remoteMod)
	}

	if err := RemoveDefaultCache(); err != nil {
		rep.Warnf("could not remove %s: %v\n", DefaultCacheFile, err)
	}

	rep.Infof("  Saved %s (%s)\n", DefaultTSVFile, formatBytes(written))
	return nil
}

func copyWithProgress(rep Reporter, dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, 1024*1024)
	var written int64
	lastReport := time.Now()

	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			wn, writeErr := dst.Write(buf[:n])
			written += int64(wn)
			if writeErr != nil {
				return written, writeErr
			}
			if wn != n {
				return written, io.ErrShortWrite
			}

			if time.Since(lastReport) >= 200*time.Millisecond {
				rep.Progress("Decompressing", written, 0)
				lastReport = time.Now()
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				return written, nil
			}
			return written, readErr
		}
	}
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