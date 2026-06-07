package bitcoin

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

type MatchKind int

const (
	MatchLegacyCompressed MatchKind = iota
	MatchLegacyUncompressed
	MatchSegwitV0
	MatchP2SH
	MatchTaproot
)

func EncodeMatchAddress(kind MatchKind, keys LookupKeys) string {
	switch kind {
	case MatchLegacyCompressed:
		return encodePubKeyHash(keys.CompressedHash)
	case MatchLegacyUncompressed:
		return encodePubKeyHash(keys.UncompressedHash)
	case MatchSegwitV0:
		return encodeWitnessPubKeyHash(keys.CompressedHash)
	case MatchP2SH:
		return encodeScriptHash(keys.P2SHHash)
	case MatchTaproot:
		return encodeTaproot(keys.TaprootKey)
	default:
		return ""
	}
}

func encodePubKeyHash(hash [20]byte) string {
	addr, err := btcutil.NewAddressPubKeyHash(hash[:], &chaincfg.MainNetParams)
	if err != nil {
		return ""
	}
	return addr.EncodeAddress()
}

func encodeWitnessPubKeyHash(hash [20]byte) string {
	addr, err := btcutil.NewAddressWitnessPubKeyHash(hash[:], &chaincfg.MainNetParams)
	if err != nil {
		return ""
	}
	return addr.EncodeAddress()
}

func encodeScriptHash(hash [20]byte) string {
	addr, err := btcutil.NewAddressScriptHashFromHash(hash[:], &chaincfg.MainNetParams)
	if err != nil {
		return ""
	}
	return addr.EncodeAddress()
}

func encodeTaproot(key [32]byte) string {
	addr, err := btcutil.NewAddressTaproot(key[:], &chaincfg.MainNetParams)
	if err != nil {
		return ""
	}
	return addr.EncodeAddress()
}