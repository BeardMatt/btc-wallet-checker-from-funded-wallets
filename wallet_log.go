package main

import (
	"fmt"
	"os"
	"time"

	"btcfind/bitcoin"
)

const walletsLogFile = "wallets.txt"

var walletsLogPath = walletsLogFile

func appendWalletHit(keyIndex int, kind bitcoin.MatchKind, address, wif string) error {
	return appendWalletHitAt(walletsLogPath, keyIndex, kind, address, wif)
}

func appendWalletHitAt(path string, keyIndex int, kind bitcoin.MatchKind, address, wif string) error {
	_, statErr := os.Stat(path)
	newFile := os.IsNotExist(statErr)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if newFile {
		if err := os.Chmod(path, 0600); err != nil {
			return err
		}
	}

	id, label := kindLabels(kind)
	if _, err := fmt.Fprintf(f,
		"--- wallet found %s ---\nkey_index: %d\nformat: %s (%s)\naddress: %s\nwif: %s\n\n",
		time.Now().Format(time.RFC3339),
		keyIndex,
		id,
		label,
		address,
		wif,
	); err != nil {
		return err
	}

	return f.Sync()
}