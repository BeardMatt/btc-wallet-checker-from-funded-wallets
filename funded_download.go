package main

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
	fundedDownloadURL = "http://addresses.loyce.club/blockchair_bitcoin_addresses_and_balance_LATEST.tsv.gz"
	fundedFile        = "funded.tsv"
)

var fetchRemoteLastModified = remoteLastModified

func ensureFunded() {
	u := ui()
	info, err := os.Stat(fundedFile)
	if os.IsNotExist(err) {
		if u.quiet {
			u.PrintfErr("funded.tsv not found, downloading...\n")
		} else {
			u.Section("Downloading funded.tsv")
		}
		if err := downloadFunded(); err != nil {
			panic(err)
		}
		return
	}
	if err != nil {
		panic(err)
	}

	if !u.quiet {
		u.Section("Funded data check")
		logLocalFundedFile(u, info)
		if u.verbose {
			if abs, err := filepath.Abs(fundedFile); err == nil {
				u.Infof("  path: %s\n", abs)
			}
			u.Infof("  remote URL: %s\n", fundedDownloadURL)
		}
	}

	u.ProgressIndeterminate("Checking for updates…")
	headStart := time.Now()
	remoteMod, err := fetchRemoteLastModified()
	headElapsed := time.Since(headStart)
	u.ClearProgress()

	if err != nil {
		u.Warnf("could not check for updates (%v), using local file\n", err)
		if !u.quiet {
			u.Infof("  Using local file\n")
		}
		return
	}

	if !u.quiet {
		if u.verbose {
			u.Infof("  HEAD request: %dms\n", headElapsed.Milliseconds())
		}
		u.Infof("  remote      %s\n", remoteMod.Format(time.RFC3339))
	}

	if remoteMod.After(info.ModTime()) {
		u.PrintfErr(
			"funded.tsv is out of date (local: %s, remote: %s)\n",
			info.ModTime().Format(time.RFC3339),
			remoteMod.Format(time.RFC3339),
		)
		if u.verbose {
			u.Infof("  note: download will remove %s\n", fundedCacheFile)
		}
		u.PrintfErr("Download update? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "y" || answer == "yes" {
			if err := downloadFunded(); err != nil {
				panic(err)
			}
			return
		}

		u.PrintlnErr("Using existing funded.tsv")
		return
	}

	if !u.quiet {
		u.Infof("  Up to date — using local file\n")
	}
}

func logLocalFundedFile(u *UI, info os.FileInfo) {
	u.Infof(
		"  %s  %s  modified %s\n",
		fundedFile,
		formatBytes(info.Size()),
		info.ModTime().UTC().Format(time.RFC3339),
	)
}

func remoteLastModified() (time.Time, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodHead, fundedDownloadURL, nil)
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

func downloadFunded() error {
	u := ui()
	u.Infof("  Source: %s\n", fundedDownloadURL)
	if u.verbose {
		u.Infof("  note: will remove %s after download\n", fundedCacheFile)
	}

	client := &http.Client{}
	resp, err := client.Get(fundedDownloadURL)
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

	tmpFile := fundedFile + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}

	written, err := copyWithProgress(out, gz)
	u.ClearProgress()
	closeErr := out.Close()
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("write failed: %w", err)
	}
	if closeErr != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("close failed: %w", closeErr)
	}

	if err := os.Rename(tmpFile, fundedFile); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("rename failed: %w", err)
	}

	if remoteMod, err := remoteLastModified(); err == nil {
		_ = os.Chtimes(fundedFile, remoteMod, remoteMod)
	}

	removeFundedCache()

	u.Infof("  Saved %s (%s)\n", fundedFile, formatBytes(written))
	return nil
}

func copyWithProgress(dst io.Writer, src io.Reader) (int64, error) {
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
				ui().Progress("Decompressing", written, 0)
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