package main

import (
	"btcfind/bitcoin"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

func formatWIF(privKeyBytes []byte) string {
	privKey, _ := btcec.PrivKeyFromBytes(privKeyBytes)
	wif, err := btcutil.NewWIF(privKey, &chaincfg.MainNetParams, true)
	if err != nil {
		panic(err)
	}
	return wif.String()
}

func simulateDisplayKind(keys bitcoin.LookupKeys, matchedKind bitcoin.MatchKind, matched bool) bitcoin.MatchKind {
	if matched {
		return matchedKind
	}
	return bitcoin.MatchLegacyCompressed
}