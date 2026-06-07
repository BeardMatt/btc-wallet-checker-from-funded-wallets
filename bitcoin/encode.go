package bitcoin

import (
	"fmt"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/btcutil/bech32"
)

const (
	mainnetPubKeyHashID = 0x00
	mainnetScriptHashID = 0x05
	mainnetBech32HRP    = "bc"
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

func EncodePubKeyHash(hash [20]byte) string {
	return encodePubKeyHash(hash)
}

func EncodeWitnessPubKeyHash(hash [20]byte) string {
	return encodeWitnessPubKeyHash(hash)
}

func EncodeScriptHash(hash [20]byte) string {
	return encodeScriptHash(hash)
}

func EncodeTaprootKey(key [32]byte) string {
	return encodeTaproot(key)
}

func encodePubKeyHash(hash [20]byte) string {
	return base58.CheckEncode(hash[:], mainnetPubKeyHashID)
}

func encodeScriptHash(hash [20]byte) string {
	return base58.CheckEncode(hash[:], mainnetScriptHashID)
}

func encodeWitnessPubKeyHash(hash [20]byte) string {
	addr, err := encodeSegWitAddress(mainnetBech32HRP, 0, hash[:])
	if err != nil {
		return ""
	}
	return addr
}

func encodeTaproot(key [32]byte) string {
	addr, err := encodeSegWitAddress(mainnetBech32HRP, 1, key[:])
	if err != nil {
		return ""
	}
	return addr
}

func encodeSegWitAddress(hrp string, witnessVersion byte, witnessProgram []byte) (string, error) {
	converted, err := bech32.ConvertBits(witnessProgram, 8, 5, true)
	if err != nil {
		return "", err
	}

	combined := make([]byte, len(converted)+1)
	combined[0] = witnessVersion
	copy(combined[1:], converted)

	switch witnessVersion {
	case 0:
		return bech32.Encode(hrp, combined)
	case 1:
		return bech32.EncodeM(hrp, combined)
	default:
		return "", fmt.Errorf("unsupported witness version %d", witnessVersion)
	}
}