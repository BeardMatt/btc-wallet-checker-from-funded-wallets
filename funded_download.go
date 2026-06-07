package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	fundedDownloadURL = "http://addresses.loyce.club/blockchair_bitcoin_addresses_and_balance_LATEST.tsv.gz"
	fundedFile        = "funded.tsv"
)

func ensureFunded() {
	info, err := os.Stat(fundedFile)
	if os.IsNotExist(err) {
		fmt.Println("funded.tsv not found, downloading...")
		if err := downloadFunded(); err != nil {
			panic(err)
		}
		return
	}
	if err != nil {
		panic(err)
	}

	remoteMod, err := remoteLastModified()
	if err != nil {
		fmt.Printf("Warning: could not check for updates (%v), using local file\n", err)
		return
	}

	if remoteMod.After(info.ModTime()) {
		fmt.Printf(
			"funded.tsv is out of date (local: %s, remote: %s)\n",
			info.ModTime().Format(time.RFC3339),
			remoteMod.Format(time.RFC3339),
		)
		fmt.Print("Download update? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "y" || answer == "yes" {
			if err := downloadFunded(); err != nil {
				panic(err)
			}
			return
		}

		fmt.Println("Using existing funded.tsv")
	}
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
	fmt.Printf("Downloading %s\n", fundedDownloadURL)
	fmt.Println("Decompressing to funded.tsv (this may take a while)...")

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

	fmt.Printf("Saved %s (%d bytes)\n", fundedFile, written)
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

			if time.Since(lastReport) >= 5*time.Second {
				fmt.Printf("  ... %d MB written\n", written/(1024*1024))
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