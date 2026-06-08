package main

import "btcfind/funded"

const (
	fundedFile        = funded.DefaultTSVFile
	fundedDownloadURL = funded.DownloadURL
)

func ensureFunded() {
	if err := funded.EnsureTSV(mainReporter{}); err != nil {
		panic(err)
	}
}